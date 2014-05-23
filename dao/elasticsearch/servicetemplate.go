// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	dutils "github.com/dotcloud/docker/utils"
	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/utils"

	"errors"
	"fmt"
	"regexp"
	"strings"
)

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateId *string) error {
	uuid, err := utils.NewUUID36()
	if err != nil {
		return err
	}
	serviceTemplate.ID = uuid

	store := servicetemplate.NewStore()
	if err = store.Put(datastore.Get(), serviceTemplate); err != nil {
		return err
	}

	*templateId = uuid
	// this takes a while so don't block the main thread
	go this.reloadLogstashContainer()
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template servicetemplate.ServiceTemplate, unused *int) error {
	store := servicetemplate.NewStore()
	if err := store.Put(datastore.Get(), template); err != nil {
		return err
	}
	go this.reloadLogstashContainer() // don't block the main thread
	return nil
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	// make sure it is a valid template first
	store := servicetemplate.NewStore()

	_, err := store.Get(datastore.Get(), id)
	if err != nil {
		return fmt.Errorf("Unable to find template: %s", id)
	}

	glog.V(2).Infof("ControlPlaneDao.RemoveServiceTemplate: %s", id)
	if err != store.Delete(datastore.Get(), id) {
		return err
	}
	go this.reloadLogstashContainer()
	return nil
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*servicetemplate.ServiceTemplate) error {
	glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates")
	store := servicetemplate.NewStore()
	results, err := store.GetServiceTemplates(datastore.Get())
	if err != nil {
		glog.V(2).Infof("ControlPlaneDao.GetServiceTemplates: err=%s", err)
		return err
	}
	templatemap := make(map[string]*servicetemplate.ServiceTemplate)
	for _, st := range results {
		templatemap[st.ID] = st
	}
	*templates = templatemap
	return nil
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantId *string) error {
	store := servicetemplate.NewStore()
	template, err := store.Get(datastore.Get(), request.TemplateID)
	if err != nil {
		glog.Errorf("unable to load template: %s", request.TemplateID)
		return err
	}

	pool, err := this.facade.GetResourcePool(datastore.Get(), request.PoolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", request.PoolID)
		return err
	}
	if pool == nil {
		return fmt.Errorf("poolid %s not found", request.PoolID)
	}

	volumes := make(map[string]string)
	return this.deployServiceDefinitions(template.Services, request.PoolID, "", volumes, request.DeploymentID, tenantId)
}

func (this *ControlPlaneDao) DeployService(request dao.ServiceDeploymentRequest, serviceId *string) error {
	parentId := strings.TrimSpace(request.ParentID)
	parent, err := service.NewStore().Get(datastore.Get(), parentId)
	if err != nil {
		return fmt.Errorf("could not get parent '%': %s", parentId, err)
	}

	tenantId, err := getTenantId(parent)
	if err != nil {
		return fmt.Errorf("getting tenant id: %s", err)
	}

	volumes := make(map[string]string)
	serviceDefinitions := []servicedefinition.ServiceDefinition{request.Service}

	// TODO: need to fill in serviceID for return. 
	return this.deployServiceDefinitions(serviceDefinitions, parent.PoolID, parentId, volumes, parent.DeploymentID, &tenantId)
}

func (this *ControlPlaneDao) deployServiceDefinition(sd servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// Always deploy in stopped state, starting is a separate step
	ds := service.SVCStop

	exportedVolumes := make(map[string]string)
	for k, v := range volumes {
		exportedVolumes[k] = v
	}
	svc, err := service.BuildService(sd, parentServiceID, pool, ds, deploymentId)
	if err != nil {
		return err
	}

	getSvc := func(svcID string) (service.Service, error) {
		svc := service.Service{}
		err := this.GetService(svcID, &svc)
		return svc, err
	}

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(getSvc); err != nil {
		return err
	}

	//for each endpoint, evaluate it's Application
	if err = svc.EvaluateEndpointTemplates(getSvc); err != nil {
		return err
	}

	if parentServiceID == "" {
		*tenantId = svc.Id
	}

	// Using the tenant id, tag the base image with the tenantID
	if svc.ImageID != "" {
		name, err := this.renameImageID(svc.ImageID, *tenantId)
		if err != nil {
			return err
		}

		dockerclient, err := docker.NewClient("unix:///var/run/docker.sock")
		if err != nil {
			glog.Errorf("unable to start docker client")
			return err
		}
		image, err := dockerclient.InspectImage(svc.ImageID)
		if err != nil {
			msg := fmt.Errorf("could not look up image %s: %s", svc.ImageID, err)
			glog.Error(err.Error())
			return msg
		}

		repo, err := this.renameImageID(svc.ImageID, *tenantId)
		if err != nil {
			glog.Errorf("malformed imageId: %s", svc.ImageID)
			return err
		}

		options := docker.TagImageOptions{
			Repo:  repo,
			Force: true,
		}
		if err := dockerclient.TagImage(image.ID, options); err != nil {
			glog.Errorf("could not tag image: %s options: %+v", image.ID, options)
			return err
		}
		svc.ImageID = name

	}

	var serviceId string
	err = this.AddService(*svc, &serviceId)
	if err != nil {
		return err
	}

	return this.deployServiceDefinitions(sd.Services, pool, svc.Id, exportedVolumes, deploymentId, tenantId)
}

func (this *ControlPlaneDao) deployServiceDefinitions(sds []servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// ensure that all images in the templates exist
	imageIds := make(map[string]struct{})
	for _, svc := range sds {
		getSubServiceImageIDs(imageIds, svc)
	}

	dockerclient, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("unable to start docker client")
		return err
	}

	for imageId, _ := range imageIds {
		_, err := dockerclient.InspectImage(imageId)
		if err != nil {
			msg := fmt.Errorf("could not look up image %s: %s", imageId, err)
			glog.Error(err.Error())
			return msg
		}
	}

	for _, sd := range sds {
		if err := this.deployServiceDefinition(sd, pool, parentServiceID, volumes, deploymentId, tenantId); err != nil {
			return err
		}
	}
	return nil
}

func getSubServiceImageIDs(ids map[string]struct{}, svc servicedefinition.ServiceDefinition) {
	found := struct{}{}

	if len(svc.ImageID) != 0 {
		ids[svc.ImageID] = found
	}
	for _, s := range svc.Services {
		getSubServiceImageIDs(ids, s)
	}
}

func (this *ControlPlaneDao) renameImageID(imageId, tenantId string) (string, error) {

	repo, _ := dutils.ParseRepositoryTag(imageId)
	re := regexp.MustCompile("/?([^/]+)\\z")
	matches := re.FindStringSubmatch(repo)
	if matches == nil {
		return "", errors.New("malformed imageid")
	}
	name := matches[1]

	return fmt.Sprintf("%s/%s/%s", this.dockerRegistry, tenantId, name), nil
}

// writeLogstashConfiguration takes all the available
// services and writes out the filters section for logstash.
// This is required before logstash startsup
func (s *ControlPlaneDao) writeLogstashConfiguration() error {
	var templatesMap map[string]*servicetemplate.ServiceTemplate
	if err := s.GetServiceTemplates(0, &templatesMap); err != nil {
		return err
	}

	// FIXME: eventually this file should live in the DFS or the config should
	// live in zookeeper to allow the agents to get to this
	if err := dao.WriteConfigurationFile(templatesMap); err != nil {
		return err
	}
	return nil
}

// Anytime the available service definitions are modified
// we need to restart the logstash container so it can write out
// its new filter set.
// This method depends on the elasticsearch container being up and running.
func (s *ControlPlaneDao) reloadLogstashContainer() error {
	err := s.writeLogstashConfiguration()
	if err != nil {
		glog.Fatalf("Could not write logstash configuration: %s", err)
		return err
	}
	glog.V(2).Info("Starting logstash container")
	if err := isvcs.Mgr.Notify("restart logstash"); err != nil {
		glog.Fatalf("Could not start logstash container: %s", err)
		return err
	}
	return nil
}

func getTenantId (svc *service.Service) (string, error) {
	if svc.ParentServiceID == "" {
		return svc.Id, nil
	}
	parent, err := service.NewStore().Get(datastore.Get(), svc.ParentServiceID)
	if err != nil {
		return "", err
	}
	return getTenantId(parent)
}
