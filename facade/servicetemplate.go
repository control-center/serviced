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
	return pullImages(getImageIDs(template.Services...))
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
func (f *Facade) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error) {
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
		return nil, err
	}

	//check that deployment id does not already exist
	if svcs, err := f.serviceStore.GetServicesByDeployment(ctx, deploymentID); err != nil {
		glog.Errorf("unable to validate deploymentID %v while deploying %v", deploymentID, templateID)
		return nil, err
	} else if len(svcs) > 0 {
		return nil, fmt.Errorf("deployment ID %s is already in use", deploymentID)
	}

	//now that we know the template name, set it in the status
	deployments[deploymentID]["templateName"] = template.Name

	UpdateDeployTemplateStatus(deploymentID, "deploy_loading_resource_pool|"+poolID)
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", poolID)
		return nil, err
	}
	if pool == nil {
		return nil, fmt.Errorf("poolid %s not found", poolID)
	}

	UpdateDeployTemplateStatus(deploymentID, "deploy_pulling_images")
	if err := pullTemplateImages(template); err != nil {
		glog.Errorf("Unable to pull one or more images")
		return nil, err
	}

	tenantIDs := make([]string, len(template.Services))
	for i, sd := range template.Services {
		glog.Infof("Deploying application %s to %s", sd.Name, deploymentID)
		var err error
		if tenantIDs[i], err = f.deployService(ctx, "", "", deploymentID, poolID, false, sd); err != nil {
			glog.Errorf("Could not deploy application %s to %s: %s", sd.Name, deploymentID, err)
			return nil, err
		}
	}

	return tenantIDs, nil
}

// DeployService converts a service definition to a service and deploys it under
// a specific service.  If the overwrite option is enabled, existing services
// with the same name will be overwritten, otherwise services may only be added.
func (f *Facade) DeployService(ctx datastore.Context, poolID, parentID string, overwrite bool, svcDef servicedefinition.ServiceDefinition) (string, error) {
	store := f.serviceStore

	// get the parent service
	svc, err := store.Get(ctx, parentID)
	if err != nil {
		glog.Errorf("Could not get parent service %s: %s", parentID, err)
		return "", err
	}

	// Do some pool validation
	if poolID != "" {
		// check the pool ID
		if pool, err := f.GetResourcePool(ctx, poolID); err != nil {
			glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
			return "", err
		} else if pool == nil {
			err := fmt.Errorf("pool not found")
			glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
			return "", err
		}
	} else {
		// If the poolID is not specified, default to use the parent service's poolID
		poolID = svc.PoolID // I am going to assume that the pool on the parent service is correct
	}

	// get the tenant id
	tenantID := svc.ID
	if svc.ParentServiceID != "" {
		if tenantID, err = f.GetTenantID(ctx, svc.ParentServiceID); err != nil { // make this call a little cheaper
			glog.Errorf("Could not get tenant for %s (%s): %s", svc.Name, svc.ID, err)
			return "", err
		}
	}

	// pull the images
	if err := pullServiceImages(&svcDef); err != nil {
		glog.Errorf("Unable to pull one or more images")
		return "", err
	}

	// check the images
	if err := checkImages(make(map[string]struct{}), svcDef); err != nil {
		glog.Errorf("Error while validating image IDs: %s", err)
		return "", err
	}

	return f.deployService(ctx, tenantID, svc.ID, svc.DeploymentID, poolID, overwrite, svcDef)
}

func (f *Facade) deployService(ctx datastore.Context, tenantID string, parentServiceID, deploymentID, poolID string, overwrite bool, svcDef servicedefinition.ServiceDefinition) (string, error) {
	// create the new service object
	newsvc, err := service.BuildService(svcDef, parentServiceID, poolID, int(service.SVCStop), deploymentID)
	if err != nil {
		glog.Errorf("Could not create service: %s", err)
		return "", err
	}

	UpdateDeployTemplateStatus(deploymentID, "deploy_loading_service|"+newsvc.Name)
	//for each endpoint, evaluate its Application
	getService := func(serviceID string) (service.Service, error) {
		s, err := f.GetService(ctx, serviceID)
		if err != nil {
			return service.Service{}, err
		}
		return *s, err
	}
	findChildService := func(parentID, serviceName string) (service.Service, error) {
		s, err := f.FindChildService(ctx, parentID, serviceName)
		if err != nil {
			return service.Service{}, err
		}
		return *s, err
	}
	if err = newsvc.EvaluateEndpointTemplates(getService, findChildService); err != nil {
		glog.Errorf("Could not evaluate endpoint templates for service %s with parent %s: %s", newsvc.Name, newsvc.ParentServiceID, err)
		return "", err
	}

	// set the imageID
	if tenantID == "" {
		tenantID = newsvc.ID
	}
	if err := setImageID(f.dockerRegistry, tenantID, newsvc); err != nil {
		glog.Errorf("Could not set image id for service %s at parent %s: %s", newsvc.Name, newsvc.ParentServiceID, err)
		return "", err
	}

	// find the service
	store := f.serviceStore
	if svc, err := store.FindChildService(ctx, newsvc.DeploymentID, newsvc.ParentServiceID, newsvc.Name); err != nil {
		glog.Errorf("Could not look up child service for %s with parent %s: %s", newsvc.Name, newsvc.ParentServiceID, err)
		return "", err
	} else if svc != nil {
		if overwrite {
			newsvc.ID = svc.ID
			newsvc.CreatedAt = svc.CreatedAt
			if err := f.UpdateService(ctx, *newsvc); err != nil {
				glog.Errorf("Could not overwrite service %s (%s): %s", newsvc.Name, newsvc.ID, err)
				return "", err
			}
		} else {
			err := fmt.Errorf("service exists")
			glog.Errorf("Service %s found at %s", newsvc.Name, newsvc.ID)
			return "", err
		}
	} else {
		if err := f.AddService(ctx, *newsvc); err != nil {
			glog.Errorf("Could not add service %s (%s) at %s: %s", newsvc.Name, newsvc.ID, parentServiceID, err)
			return "", err
		}
	}

	// walk child services
	for _, sd := range svcDef.Services {
		if _, err := f.deployService(ctx, tenantID, newsvc.ID, deploymentID, poolID, overwrite, sd); err != nil {
			glog.Errorf("Error while trying to deploy %s at %s (%s): %s", sd.Name, newsvc.Name, newsvc.ID, err)
			return newsvc.ID, err
		}
	}
	return newsvc.ID, nil
}

func checkImages(imap map[string]struct{}, svcdef servicedefinition.ServiceDefinition) error {
	if _, ok := imap[svcdef.ImageID]; !ok {
		if _, err := docker.FindImage(svcdef.ImageID, false); err != nil {
			glog.Errorf("Could not get image %s: %s", svcdef.ImageID, err)
			return err
		}
		imap[svcdef.ImageID] = struct{}{}
	}
	for _, childDef := range svcdef.Services {
		if err := checkImages(imap, childDef); err != nil {
			return err
		}
	}
	return nil
}

func setImageID(registry, tenantID string, svc *service.Service) error {
	if svc.ImageID == "" {
		return nil
	}

	UpdateDeployTemplateStatus(svc.DeploymentID, "deploy_renaming_image|"+svc.Name)
	imageID, err := renameImageID(registry, svc.ImageID, tenantID)
	if err != nil {
		glog.Errorf("malformed imageID %s: %s", svc.ImageID, err)
		return err
	}

	if _, err := docker.FindImage(imageID, false); err == docker.ErrNoSuchImage {
		UpdateDeployTemplateStatus(svc.DeploymentID, "deploy_loading_image|"+svc.Name)
		// tagged image not found, so look for the base image
		image, err := docker.FindImage(svc.ImageID, false)
		if err != nil {
			glog.Errorf("Could not search for image %s: %s", svc.ImageID, err)
			return err
		}
		UpdateDeployTemplateStatus(svc.DeploymentID, "deploy_tagging_image|"+svc.Name)
		// now tag the image
		if _, err := image.Tag(imageID); err != nil {
			glog.Errorf("Could not add tag %s to image %s: %s", imageID, svc.ImageID, err)
			return err
		}
	}
	svc.ImageID = imageID
	return nil
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

func pullServiceImages(svcDef *servicedefinition.ServiceDefinition) error {
	return pullImages(getImageIDs(*svcDef))
}

func pullImages(imgs []string) error {
	for _, img := range imgs {
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
