// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/docker/docker/pkg/parsers"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/isvcs"
)

type reloadLogstashContainer func(ctx datastore.Context, f FacadeInterface) error

var LogstashContainerReloader reloadLogstashContainer = reloadLogstashContainerImpl

var getDockerClient = func() (*dockerclient.Client, error) { return dockerclient.NewClient("unix:///var/run/docker.sock") }

//AddServiceTemplate  adds a service template to the system. Returns the id of the template added
func (f *Facade) AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate) (string, error) {
	hash, err := serviceTemplate.Hash()
	if err != nil {
		return "", err
	}
	serviceTemplate.ID = hash

	if st, _ := f.templateStore.Get(ctx, hash); st != nil {
		// This id already exists in the system
		glog.Infof("Not replacing existing template %s", hash)
		return hash, nil
	}

	if err = f.templateStore.Put(ctx, serviceTemplate); err != nil {
		return "", err
	}

	// this takes a while so don't block the main thread
	go LogstashContainerReloader(ctx, f)
	return hash, err
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

func (f *Facade) GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error) {
	glog.V(2).Infof("Facade.GetServiceTemplates")
	results, err := f.templateStore.GetServiceTemplates(ctx)
	templateMap := make(map[string]servicetemplate.ServiceTemplate)
	if err != nil {
		glog.V(2).Infof("Facade.GetServiceTemplates: err=%s", err)
		return templateMap, err
	}
	for _, st := range results {
		templateMap[st.ID] = *st
	}
	return templateMap, nil
}

func getImageIDs(sds ...servicedefinition.ServiceDefinition) []string {
	set := map[string]struct{}{}
	for _, sd := range sds {
		for _, img := range getImageIDs(sd.Services...) {
			set[img] = struct{}{}
		}
		if sd.ImageID != "" {
			set[sd.ImageID] = struct{}{}
		}
	}
	result := []string{}
	for img, _ := range set {
		result = append(result, img)
	}
	return result
}

func pullTemplateImages(template *servicetemplate.ServiceTemplate) error {
	for _, img := range getImageIDs(template.Services...) {
		imageID, err := commons.ParseImageID(img)
		if err != nil {
			return err
		}
		tag := imageID.Tag
		if tag == "" {
			tag = "latest"
		}
		image := fmt.Sprintf("%s:%s", imageID.BaseName(), tag)
		glog.Infof("Pulling image %s", image)
		if err := docker.PullImage(image); err != nil {
			glog.Warningf("Unable to pull image %s", image)
		}
	}
	return nil
}

var deployments = make(map[string]map[string]string)

// UpdateDeployTemplateStatus updates the deployment status of the service being deployed
func UpdateDeployTemplateStatus(deploymentID string, status string) {
	if _, ok := deployments[deploymentID]; !ok {
		deployments[deploymentID] = make(map[string]string)
	}

	deployments[deploymentID]["lastStatus"] = deployments[deploymentID]["status"]
	deployments[deploymentID]["status"] = status
}

// gather a list of all active DeploymentIDs
func (f *Facade) DeployTemplateActive(active *[]map[string]string) error {
	// we initialize the data container to something here in case it has not been initialized yet
	*active = make([]map[string]string, 0)
	for _, v := range deployments {
		*active = append(*active, v)
	}

	return nil
}

// DeployTemplateStatus sets the status of a deployed service or template
func (f *Facade) DeployTemplateStatus(deploymentID string, status *string) error {
	if _, ok := deployments[deploymentID]; ok {
		if deployments[deploymentID]["lastStatus"] != deployments[deploymentID]["status"] {
			deployments[deploymentID]["lastStatus"] = deployments[deploymentID]["status"]
			*status = deployments[deploymentID]["status"]
		} else if deployments[deploymentID]["status"] != "" {
			time.Sleep(100 * time.Millisecond)
			f.DeployTemplateStatus(deploymentID, status)
		}
	} else {
		*status = ""
	}

	return nil
}

//DeployTemplate creates and deployes a service to the pool and returns the tenant id of the newly deployed service
func (f *Facade) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) (string, error) {
	// add an entry for reporting status
	deployments[deploymentID] = map[string]string{
		"TemplateID":   templateID,
		"DeploymentID": deploymentID,
		"PoolID":       poolID,
		"status":       "Starting",
		"lastStatus":   "",
	}
	defer delete(deployments, deploymentID)

	UpdateDeployTemplateStatus(deploymentID, "deploy_loading_template|"+templateID)
	template, err := f.templateStore.Get(ctx, templateID)
	if err != nil {
		glog.Errorf("unable to load template: %s", templateID)
		return "", err
	}

	//check that deployment id does not already exist
	svcs, err := f.serviceStore.GetServicesByDeployment(ctx, deploymentID)
	if err != nil {
		glog.Errorf("unable to validate deploymentID %v while deploying %v", deploymentID, templateID)
		return "", err
	}
	for _, svc := range svcs {
		if svc.DeploymentID == deploymentID {
			return "", fmt.Errorf("deployment ID %v is already in use", deploymentID)
		}
	}

	//now that we know the template name, set it in the status
	deployments[deploymentID]["templateName"] = template.Name

	UpdateDeployTemplateStatus(deploymentID, "deploy_loading_resource_pool|"+poolID)
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", poolID)
		return "", err
	}
	if pool == nil {
		return "", fmt.Errorf("poolid %s not found", poolID)
	}

	UpdateDeployTemplateStatus(deploymentID, "deploy_pulling_images")
	if err := pullTemplateImages(template); err != nil {
		glog.Errorf("Unable to pull one or more images")
		return "", err
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
	return f.deployServiceDefinition(ctx, sd, parent.PoolID, parentID, volumes, parent.DeploymentID, &tenantId)
}

func (f *Facade) deployServiceDefinition(ctx datastore.Context, sd servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) (string, error) {
	// Always deploy in stopped state, starting is a separate step
	ds := int(service.SVCStop)

	exportedVolumes := make(map[string]string)
	for k, v := range volumes {
		exportedVolumes[k] = v
	}
	svc, err := service.BuildService(sd, parentServiceID, pool, ds, deploymentId)
	if err != nil {
		return "", err
	}

	UpdateDeployTemplateStatus(deploymentId, "deploy_loading_service|"+svc.Name)
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
		return "", err
	}

	//for each endpoint, evaluate its Application
	if err = svc.EvaluateEndpointTemplates(getSvc, findChild); err != nil {
		return "", err
	}

	if parentServiceID == "" {
		*tenantId = svc.ID
	}

	// Using the tenant id, tag the base image with the tenantID
	if svc.ImageID != "" {
		UpdateDeployTemplateStatus(deploymentId, "deploy_renaming_image|"+svc.Name)
		name, err := renameImageID(f.dockerRegistry, svc.ImageID, *tenantId)
		if err != nil {
			glog.Errorf("malformed imageId: %s", svc.ImageID)
			return "", err
		}

		_, err = docker.FindImage(name, false)
		if err != nil {
			if err != docker.ErrNoSuchImage && !strings.HasPrefix(err.Error(), "No such id:") {
				glog.Error(err)
				return "", err
			}
			UpdateDeployTemplateStatus(deploymentId, "deploy_loading_image|"+name)
			image, err := docker.FindImage(svc.ImageID, false)
			if err != nil {
				msg := fmt.Errorf("could not look up image %s: %s. Check your docker login and retry application deployment.", svc.ImageID, err)
				glog.Error(err.Error())
				return "", msg
			}
			UpdateDeployTemplateStatus(deploymentId, "deploy_tagging_image|"+name)
			if _, err := image.Tag(name); err != nil {
				glog.Errorf("could not tag image: %s (%v)", image.ID, err)
				return "", err
			}
		}
		svc.ImageID = name
	}

	err = f.AddService(ctx, *svc)
	if err != nil {
		return "", err
	}

	return svc.ID, f.deployServiceDefinitions(ctx, sd.Services, pool, svc.ID, exportedVolumes, deploymentId, tenantId)
}

func (f *Facade) deployServiceDefinitions(ctx datastore.Context, sds []servicedefinition.ServiceDefinition, pool string, parentServiceID string, volumes map[string]string, deploymentId string, tenantId *string) error {
	// ensure that all images in the templates exist
	imageIds := make(map[string]struct{})
	for _, svc := range sds {
		getSubServiceImageIDs(imageIds, svc)
	}

	for imageId, _ := range imageIds {
		_, err := docker.FindImage(imageId, false)
		if err != nil {
			msg := fmt.Errorf("could not look up image %s: %s. Check your docker login and retry service deployment.", imageId, err)
			glog.Error(err.Error())
			return msg
		}
	}

	for _, sd := range sds {
		if _, err := f.deployServiceDefinition(ctx, sd, pool, parentServiceID, volumes, deploymentId, tenantId); err != nil {
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

	repo, _ := parsers.ParseRepositoryTag(imageId)
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
func writeLogstashConfiguration(templates map[string]servicetemplate.ServiceTemplate) error {
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
func reloadLogstashContainerImpl(ctx datastore.Context, f FacadeInterface) error {
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
