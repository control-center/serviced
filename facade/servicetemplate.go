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
	"fmt"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"

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
	store := f.templateStore
	hash, err := serviceTemplate.Hash()
	if err != nil {
		return "", err
	}
	serviceTemplate.ID = hash
	// Look up the template by ID
	if st, err := store.Get(ctx, hash); err != nil && !datastore.IsErrNoSuchEntity(err) {
		glog.Errorf("Could not look up service template by hash %s: %s", hash, err)
		return "", err
	} else if st != nil {
		glog.Infof("Not replacing existing template %s", hash)
		return hash, nil
	}
	// Look up the template by md5 hash
	if tid, err := f.getServiceTemplateByMD5Sum(ctx, hash); err != nil {
		glog.Errorf("Could not verify existance of template by md5sum %s: %s", hash, err)
		return "", nil
	} else if tid != "" {
		glog.Infof("Not replacing existing template %s", tid)
		return tid, nil
	}
	// Add the template to the database
	if err := store.Put(ctx, serviceTemplate); err != nil {
		glog.Errorf("Could not add template at %s: %s", hash, err)
		return "", err
	}
	// This takes a while so don't block the main thread
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

// RestoreServiceTemplates restores a service template, typically from a backup
func (f *Facade) RestoreServiceTemplates(ctx datastore.Context, templates []servicetemplate.ServiceTemplate) error {
	curtemplates, err := f.GetServiceTemplates(ctx)
	if err != nil {
		glog.Errorf("Could not look up service templates: %s", err)
		return err
	}

	for _, template := range templates {
		template.DatabaseVersion = 0
		if _, ok := curtemplates[template.ID]; ok {
			if err := f.UpdateServiceTemplate(ctx, template); err != nil {
				glog.Errorf("Could not update service template %s: %s", template.ID, err)
				return err
			}
		} else {
			template.ID = ""
			if _, err := f.AddServiceTemplate(ctx, template); err != nil {
				glog.Errorf("Could not add service template %s: %s", template.ID, err)
				return err
			}
		}
	}
	return nil
}

// getServiceTemplateByMD5Sum returns the id of the template that matches the
// given md5sum (if it exists)
func (f *Facade) getServiceTemplateByMD5Sum(ctx datastore.Context, md5Sum string) (string, error) {
	store := f.templateStore
	templates, err := store.GetServiceTemplates(ctx)
	if err != nil {
		glog.Errorf("Could not get service templates: %s", err)
		return "", err
	}
	for _, t := range templates {
		hash, err := t.Hash()
		if err != nil {
			glog.Errorf("Could not get md5sum for template %s: %s", t.ID, err)
			return "", err
		}
		if md5Sum == hash {
			return t.ID, nil
		}
	}
	return "", nil
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

func (f *Facade) GetServiceTemplatesAndImages(ctx datastore.Context) ([]servicetemplate.ServiceTemplate, []string, error) {
	glog.V(2).Infof("Facade.GetServiceTemplateImages")
	results, err := f.templateStore.GetServiceTemplates(ctx)
	if err != nil {
		glog.Errorf("Could not get service templates: %s", err)
		return nil, nil, err
	}
	var imagesMap = make(map[string]struct{})
	var images []string

	var getImages func(sds []servicedefinition.ServiceDefinition)
	getImages = func(sds []servicedefinition.ServiceDefinition) {
		for _, sd := range sds {
			if sd.ImageID != "" {
				if _, ok := imagesMap[sd.ImageID]; !ok {
					imagesMap[sd.ImageID] = struct{}{}
					images = append(images, sd.ImageID)
				}
			}
			getImages(sd.Services)
		}
	}
	templates := make([]servicetemplate.ServiceTemplate, len(results))
	for i, tpl := range results {
		getImages(tpl.Services)
		templates[i] = *tpl
	}
	return templates, images, nil
}

// FIXME: Potential race if multiple clients are deploying apps and checking deployment status
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
func (f *Facade) DeployTemplateActive() (active []map[string]string, err error) {
	// we initialize the data container to something here in case it has not been initialized yet
	active = make([]map[string]string, 0)
	for _, v := range deployments {
		active = append(active, v)
	}

	return active, nil
}

// DeployTemplateStatus sets the status of a deployed service or template
func (f *Facade) DeployTemplateStatus(deploymentID string) (status string, err error) {
	status = ""
	err = nil
	if _, ok := deployments[deploymentID]; ok {
		if deployments[deploymentID]["lastStatus"] != deployments[deploymentID]["status"] {
			deployments[deploymentID]["lastStatus"] = deployments[deploymentID]["status"]
			status = deployments[deploymentID]["status"]
		} else if deployments[deploymentID]["status"] != "" {
			time.Sleep(100 * time.Millisecond)
			status, err = f.DeployTemplateStatus(deploymentID)
		}
	}

	return status, err
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

	tenantIDs := make([]string, len(template.Services))
	for i, sd := range template.Services {
		glog.Infof("Deploying application %s to %s", sd.Name, deploymentID)
		tenantID, err := f.deployService(ctx, "", "", deploymentID, poolID, false, sd)
		if err != nil {
			glog.Errorf("Could not deploy application %s to %s: %s", sd.Name, deploymentID, err)
			return nil, err
		}
		if err := f.dfs.Create(tenantID); err != nil {
			glog.Errorf("Could not initialize volume for tenant %s: %s", tenantID, err)
			return nil, err
		}
		tenantIDs[i] = tenantID
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
	if tenantID == "" {
		tenantID = newsvc.ID
	}
	if svcDef.ImageID != "" {
		UpdateDeployTemplateStatus(newsvc.DeploymentID, "deploy_loading_image|"+newsvc.Name)
		image, err := f.dfs.Download(svcDef.ImageID, tenantID, false)
		if err != nil {
			glog.Errorf("Could not download image %s: %s", svcDef.ImageID, err)
			return "", err
		}
		newsvc.ImageID = image
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
		glog.Errorf("Could not write logstash configuration: %s", err)
		return err
	}
	if err := writeLogstashConfiguration(templates); err != nil {
		glog.Errorf("Could not write logstash configuration: %s", err)
		return err
	}
	glog.V(2).Infof("Starting logstash container")
	if err := isvcs.Mgr.Notify("restart logstash"); err != nil {
		glog.Errorf("Could not start logstash container: %s", err)
		return err
	}
	return nil
}
