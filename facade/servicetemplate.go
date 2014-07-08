// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	dutils "github.com/dotcloud/docker/utils"
	dockerclient "github.com/zenoss/go-dockerclient"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons/docker"
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

type reloadLogstashContainer func(ctx datastore.Context, f *Facade) error

var LogstashContainerReloader reloadLogstashContainer = reloadLogstashContainerImpl

var getDockerClient = func() (*dockerclient.Client, error) { return dockerclient.NewClient("unix:///var/run/docker.sock") }

//AddServiceTemplate  adds a service template to the system. Returns the id of the template added
func (f *Facade) AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate) (string, error) {
	uuid, err := utils.NewUUID36()
	if err != nil {
		return "", err
	}
	serviceTemplate.ID = uuid

	if err = f.templateStore.Put(ctx, serviceTemplate); err != nil {
		return "", err
	}

	// this takes a while so don't block the main thread
	go LogstashContainerReloader(ctx, f)
	return uuid, err
}

//UpdateServiceTemplate updates a service template
func (f *Facade) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error {
	if err := f.templateStore.Put(ctx, template); err != nil {
		return err
	}
	go LogstashContainerReloader(ctx, f) // don't block the main thread
	return nil
}

//RemoveServiceTemplate removes the service template from the system
func (f *Facade) RemoveServiceTemplate(ctx datastore.Context, id string) error {
	if _, err := f.templateStore.Get(ctx, id); err != nil {
		return fmt.Errorf("Unable to find template: %s", id)
	}

	glog.V(2).Infof("Facade.RemoveServiceTemplate: %s", id)
	if err := f.templateStore.Delete(ctx, id); err != nil {
		return err
	}

	go LogstashContainerReloader(ctx, f)
	return nil
}

func (f *Facade) GetServiceTemplates(ctx datastore.Context) (map[string]*servicetemplate.ServiceTemplate, error) {
	glog.V(2).Infof("Facade.GetServiceTemplates")
	results, err := f.templateStore.GetServiceTemplates(ctx)
	templateMap := make(map[string]*servicetemplate.ServiceTemplate)
	if err != nil {
		glog.V(2).Infof("Facade.GetServiceTemplates: err=%s", err)
		return templateMap, err
	}
	for _, st := range results {
		templateMap[st.ID] = st
	}
	return templateMap, nil
}

//DeployTemplate creates and deployes a service to the pool and returns the tenant id of the newly deployed service
func (f *Facade) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) (string, error) {
	template, err := f.templateStore.Get(ctx, templateID)
	if err != nil {
		glog.Errorf("unable to load template: %s", templateID)
		return "", err
	}

	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", poolID)
		return "", err
	}
	if pool == nil {
		return "", fmt.Errorf("poolid %s not found", poolID)
	}

	volumes := make(map[string]string)
	var tenantID string
	err = f.deployServiceDefinitions(ctx, template.Services, poolID, "", volumes, deploymentID, &tenantID)
	return tenantID, err
}

func (f *Facade) DeployService(ctx datastore.Context, parentID string, sd servicedefinition.ServiceDefinition) (string, error) {
	parent, err := service.NewStore().Get(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("could not get parent '%s': %s", parentID, err)
	}

	tenantId, err := f.GetTenantID(ctx, parentID)
	if err != nil {
		return "", fmt.Errorf("getting tenant id: %s", err)
	}

	volumes := make(map[string]string)
	serviceDefinitions := []servicedefinition.ServiceDefinition{sd}

	// TODO: need to fill in serviceID for return.
	err = f.deployServiceDefinitions(ctx, serviceDefinitions, parent.PoolID, parentID, volumes, parent.DeploymentID, &tenantId)
	return tenantId, err
}

func (f *Facade) deployServiceDefinition(ctx datastore.Context, sd servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) error {
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
		svc, err := f.GetService(ctx, svcID)
		return *svc, err
	}
	findChild := func(svcID, childName string) (service.Service, error) {
		svc, err := f.FindChildService(ctx, svcID, childName)
		return *svc, err
	}

	//for each endpoint, evaluate its Application
	if err = svc.EvaluateEndpointTemplates(getSvc, findChild); err != nil {
		return err
	}

	//for each endpoint, evaluate its Application
	if err = svc.EvaluateEndpointTemplates(getSvc, findChild); err != nil {
		return err
	}

	if parentServiceID == "" {
		*tenantId = svc.ID
	}

	// Using the tenant id, tag the base image with the tenantID
	if svc.ImageID != "" {
		name, err := renameImageID(f.dockerRegistry, svc.ImageID, *tenantId)
		if err != nil {
			glog.Errorf("malformed imageId: %s", svc.ImageID)
			return err
		}
		dc, err := getDockerClient()
		if err != nil {
			glog.Errorf("unable to start docker client")
			return err
		}
		registry, err := docker.NewDockerRegistry(f.dockerRegistry)
		if err != nil {
			glog.Errorf("unable to use docker registry: %s", err)
			return err
		}
		_, err = docker.InspectImage(*registry, dc, name)
		if err != nil {
			if err != dockerclient.ErrNoSuchImage && !strings.HasPrefix(err.Error(), "No such id:") {
				glog.Error(err)
				return err
			}
			image, err := docker.InspectImage(*registry, dc, svc.ImageID)
			if err != nil {
				msg := fmt.Errorf("could not look up image %s: %s", svc.ImageID, err)
				glog.Error(err.Error())
				return msg
			}
			options := dockerclient.TagImageOptions{
				Repo:  name,
				Force: true,
			}
			if err := docker.TagImage(*registry, dc, svc.ImageID, options); err != nil {
				glog.Errorf("could not tag image: %s options: %+v", image.ID, options)
				return err
			}
		}
		svc.ImageID = name
	}

	err = f.AddService(ctx, *svc)
	if err != nil {
		return err
	}

	return f.deployServiceDefinitions(ctx, sd.Services, pool, svc.ID, exportedVolumes, deploymentId, tenantId)
}

func (f *Facade) deployServiceDefinitions(ctx datastore.Context, sds []servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// ensure that all images in the templates exist
	imageIds := make(map[string]struct{})
	for _, svc := range sds {
		getSubServiceImageIDs(imageIds, svc)
	}

	dockerclient, err := dockerclient.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Errorf("unable to start docker client")
		return err
	}
	registry, err := docker.NewDockerRegistry(f.dockerRegistry)
	if err != nil {
		glog.Errorf("unable to use docker registry: %s", err)
		return err
	}
	for imageId, _ := range imageIds {
		_, err := docker.InspectImage(*registry, dockerclient, imageId)
		if err != nil {
			msg := fmt.Errorf("could not look up image %s: %s", imageId, err)
			glog.Error(err.Error())
			return msg
		}
	}

	for _, sd := range sds {
		if err := f.deployServiceDefinition(ctx, sd, pool, parentServiceID, volumes, deploymentId, tenantId); err != nil {
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

func renameImageID(dockerRegistry, imageId, tenantId string) (string, error) {

	repo, _ := dutils.ParseRepositoryTag(imageId)
	re := regexp.MustCompile("/?([^/]+)\\z")
	matches := re.FindStringSubmatch(repo)
	if matches == nil {
		return "", errors.New("malformed imageid")
	}
	name := matches[1]
	return fmt.Sprintf("%s/%s/%s", dockerRegistry, tenantId, name), nil
}

// writeLogstashConfiguration takes all the available
// services and writes out the filters section for logstash.
// This is required before logstash startsup
func writeLogstashConfiguration(templates map[string]*servicetemplate.ServiceTemplate) error {
	// FIXME: eventually this file should live in the DFS or the config should
	// live in zookeeper to allow the agents to get to this
	if err := dao.WriteConfigurationFile(templates); err != nil {
		return err
	}
	return nil
}

// Anytime the available service definitions are modified
// we need to restart the logstash container so it can write out
// its new filter set.
// This method depends on the elasticsearch container being up and running.
func reloadLogstashContainerImpl(ctx datastore.Context, f *Facade) error {
	templates, err := f.GetServiceTemplates(ctx)
	if err != nil {
		glog.Fatalf("Could not write logstash configuration: %s", err)
	}

	if err := writeLogstashConfiguration(templates); err != nil {
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
