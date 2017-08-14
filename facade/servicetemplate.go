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

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/audit"
	"github.com/Sirupsen/logrus"
)

var getDockerClient = func() (*dockerclient.Client, error) { return dockerclient.NewClient("unix:///var/run/docker.sock") }

//AddServiceTemplate  adds a service template to the system. Returns the id of the template added
func (f *Facade) AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate, reloadLogstashConfig bool) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddServiceTemplate"))
	alog := f.auditLogger.Message(ctx, "Adding Service Template").
		Action(audit.Add).Type(serviceTemplate.GetType())
	store := f.templateStore
	hash, err := serviceTemplate.Hash()
	if err != nil {
		return "", alog.Error(err)
	}
	serviceTemplate.ID = hash
	alog = alog.Entity(&serviceTemplate)
	// Look up the template by ID
	if st, err := store.Get(ctx, hash); err != nil && !datastore.IsErrNoSuchEntity(err) {
		glog.Errorf("Could not look up service template by hash %s: %s", hash, err)
		return "", alog.Error(err)
	} else if st != nil {
		glog.Infof("Not replacing existing template %s", hash)
		alog.Succeeded()
		return hash, nil
	}
	// Look up the template by md5 hash
	if tid, err := f.getServiceTemplateByMD5Sum(ctx, hash); err != nil {
		glog.Errorf("Could not verify existance of template by md5sum %s: %s", hash, err)
		alog.Succeeded()
		return "", nil
	} else if tid != "" {
		glog.Infof("Not replacing existing template %s", tid)
		alog.Succeeded()
		return tid, nil
	}
	// Add the template to the database
	if err := store.Put(ctx, serviceTemplate); err != nil {
		glog.Errorf("Could not add template at %s: %s", hash, err)
		return "", alog.Error(err)
	}

	if err := f.UpdateLogFilters(ctx, &serviceTemplate); err != nil {
		glog.Errorf("Could not add/update logfilters for template %s: %s", hash, err)
		return "", alog.Error(err)
	}

	glog.Infof("Added template %s (%s)", serviceTemplate.Name, serviceTemplate.ID)
	// This takes a while so don't block the main thread
	if reloadLogstashConfig {
		go LogstashContainerReloader(ctx, f)
	}
	return hash, alog.Error(err)
}

//UpdateServiceTemplate updates a service template
func (f *Facade) UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate, reloadLogstashConfig bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.UpdateServiceTemplate"))
	alog := f.auditLogger.Message(ctx, "Updating Service Template").
		Action(audit.Update).Entity(&template)
	if err := f.templateStore.Put(ctx, template); err != nil {
		return alog.Error(err)
	}

	if err := f.UpdateLogFilters(ctx, &template); err != nil {
		glog.Errorf("Could not add/update logfilters for template %s: %s", template.ID, err)
		return alog.Error(err)
	}

	glog.Infof("Updated template %s (%s)", template.Name, template.ID)
	if reloadLogstashConfig {
		go LogstashContainerReloader(ctx, f) // don't block the main thread
	}
	alog.Succeeded()
	return nil
}

//RemoveServiceTemplate removes the service template from the system
func (f *Facade) RemoveServiceTemplate(ctx datastore.Context, id string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveServiceTemplate"))
	alog := f.auditLogger.Message(ctx, "Removing Service Template").
		Action(audit.Remove).ID(id).Type(servicetemplate.GetType())
	_, err := f.templateStore.Get(ctx, id)
	if err != nil {
		return alog.Error(fmt.Errorf("Unable to find template: %s", id))
	}

	// CC-3673 - LogFilters are NOT removed to avoid breaking any deployed services that might be using those filters

	glog.Infof("Removed template: %s", id)
	if err := f.templateStore.Delete(ctx, id); err != nil {
		return alog.Error(err)
	}

	go LogstashContainerReloader(ctx, f)
	alog.Succeeded()
	return nil
}

// RestoreServiceTemplates restores a service template, typically from a backup
func (f *Facade) RestoreServiceTemplates(ctx datastore.Context, templates []servicetemplate.ServiceTemplate) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RestoreServiceTemplates"))
	curtemplates, err := f.GetServiceTemplates(ctx)
	if err != nil {
		glog.Errorf("Could not look up service templates: %s", err)
		return err
	}

	reloadLogstashConfig := false // defer reloading until all templates have been updated
	var alog audit.Logger
	for _, template := range templates {
		template.DatabaseVersion = 0
		alog = f.auditLogger.Entity(&template)
		if _, ok := curtemplates[template.ID]; ok {
			alog = alog.Message(ctx, "Updating Service Template").Action(audit.Update)
			if err := f.UpdateServiceTemplate(ctx, template, reloadLogstashConfig); err != nil {
				glog.Errorf("Could not update service template %s: %s", template.ID, err)
				return alog.Error(err)
			}
		} else {
			template.ID = ""
			alog = alog.Message(ctx, "Adding Service Template").Action(audit.Add)
			if _, err := f.AddServiceTemplate(ctx, template, reloadLogstashConfig); err != nil {
				glog.Errorf("Could not add service template %s: %s", template.ID, err)
				return alog.Error(err)
			}
		}
	}

	// Now that all templates ahve been update, we need to update the logstash configuration
	go LogstashContainerReloader(ctx, f)
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceTemplates"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceTemplatesAndImages"))
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

// gather a list of all active DeploymentIDs
func (f *Facade) DeployTemplateActive() (active []map[string]string, err error) {
	// we initialize the data container to something here in case it has not been initialized yet
	active = make([]map[string]string, 0)

	f.deployments.mutex.RLock()
	defer f.deployments.mutex.RUnlock()
	for _, v := range f.deployments.deployments {
		active = append(active, v.GetInfo())
	}

	return active, nil
}

// DeployTemplateStatus returns the current status of a deployed template.
// If the current status is the same as the value of the lastStatus parameter,
// block until the status changes, then return the new status.  A timeout may
// be applied to the status change wait; if the timeout is negative then return
// immediately even if the status matches; if the timeout is zero then do not
// timeout.
func (f *Facade) DeployTemplateStatus(deploymentID string, lastStatus string, timeout time.Duration) (status string, err error) {
	deployment := f.deployments.GetPendingDeployment(deploymentID)
	if deployment == nil {
		return "", nil
	}

	status, statusChanged := deployment.GetStatus()
	if status != lastStatus || timeout < 0 {
		return status, nil
	}

	var timer <-chan time.Time
	if timeout != 0 {
		timer = time.After(timeout)
	}

	select {
	case <-timer:
		return status, nil
	case <-statusChanged:
		status, _ = deployment.GetStatus()
		return status, nil
	}
}

//DeployTemplate creates and deploys a service to the pool and returns the tenant id of the newly deployed service
func (f *Facade) DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DeployTemplate"))
	alog := f.auditLogger.Message(ctx, "Deploying Service Template").
		Action(audit.Deploy).ID(templateID).Type(servicetemplate.GetType()).
		WithFields(logrus.Fields{"poolid": poolID, "deploymentid": deploymentID})
	// add an entry for reporting status
	deployment, err := f.deployments.NewPendingDeployment(deploymentID, templateID, poolID)
	if err != nil {
		return nil, alog.Error(err)
	}
	defer f.deployments.DeletePendingDeployment(deploymentID)

	deployment.UpdateStatus("deploy_loading_template|" + templateID)
	template, err := f.templateStore.Get(ctx, templateID)
	if err != nil {
		glog.Errorf("unable to load template: %s", templateID)
		return nil, alog.Error(err)
	}

	//check that deployment id does not already exist
	if svcs, err := f.serviceStore.GetServicesByDeployment(ctx, deploymentID); err != nil {
		glog.Errorf("unable to validate deploymentID %v while deploying %v", deploymentID, templateID)
		return nil, alog.Error(err)
	} else if len(svcs) > 0 {
		return nil, alog.Error(fmt.Errorf("deployment ID %s is already in use", deploymentID))
	}

	//now that we know the template name, set it in the status
	deployment.SetTemplateName(template.Name)

	deployment.UpdateStatus("deploy_loading_resource_pool|" + poolID)
	pool, err := f.GetResourcePool(ctx, poolID)
	if err != nil {
		glog.Errorf("Unable to load resource pool: %s", poolID)
		return nil, alog.Error(err)
	}
	if pool == nil {
		return nil, alog.Error(fmt.Errorf("poolid %s not found", poolID))
	}

	var statusUpdater = func(status string) {
		deployment.UpdateStatus(status)
	}

	tenantIDs := make([]string, len(template.Services))
	for i, sd := range template.Services {
		glog.Infof("Deploying application %s to %s", sd.Name, deploymentID)
		tenantID, err := f.deployService(ctx, "", "", deploymentID, poolID, false, sd, statusUpdater)
		if err != nil {
			glog.Errorf("Could not deploy application %s to %s: %s", sd.Name, deploymentID, err)
			return nil, alog.Error(err)
		}
		if err := f.dfs.Create(tenantID); err != nil {
			glog.Errorf("Could not initialize volume for tenant %s: %s", tenantID, err)
			return nil, alog.Error(err)
		}
		tenantIDs[i] = tenantID
	}
	alog.Succeeded()
	return tenantIDs, nil
}

// DeployService converts a service definition to a service and deploys it under
// a specific service.  If the overwrite option is enabled, existing services
// with the same name will be overwritten, otherwise services may only be added.
func (f *Facade) DeployService(ctx datastore.Context, poolID, parentID string, overwrite bool, svcDef servicedefinition.ServiceDefinition) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DeployService"))
	store := f.serviceStore
	alog  := f.auditLogger.Message(ctx, "Deploying Service Definition").Action(audit.Deploy).
		ID(parentID).Type(service.GetType()).
		WithFields(logrus.Fields{"poolid": poolID, "svcdefname": svcDef.GetID()})
	// get the parent service
	svc, err := store.Get(ctx, parentID)
	if err != nil {
		glog.Errorf("Could not get parent service %s: %s", parentID, err)
		return "", alog.Error(err)
	}

	// Do some pool validation
	if poolID != "" {
		// check the pool ID
		if pool, err := f.GetResourcePool(ctx, poolID); err != nil {
			glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
			return "", alog.Error(err)
		} else if pool == nil {
			err := fmt.Errorf("pool not found")
			glog.Errorf("Could not look up resource pool %s: %s", poolID, err)
			return "", alog.Error(err)
		}
	} else {
		// If the poolID is not specified, default to use the parent service's poolID
		poolID = svc.PoolID // I am going to assume that the pool on the parent service is correct
	}

	// get the tenant id
	tenantID := svc.ID
	alog = alog.WithField("tenantid", tenantID)
	if svc.ParentServiceID != "" {
		if tenantID, err = f.GetTenantID(ctx, svc.ParentServiceID); err != nil { // make this call a little cheaper
			glog.Errorf("Could not get tenant for %s (%s): %s", svc.Name, svc.ID, err)
			return "", alog.Error(err)
		}
	}

	var statusUpdater = func(status string) {}
	result, err := f.deployService(ctx, tenantID, svc.ID, svc.DeploymentID, poolID, overwrite, svcDef, statusUpdater)
	return result, alog.Error(err)
}

func (f *Facade) deployService(ctx datastore.Context, tenantID string, parentServiceID, deploymentID, poolID string, overwrite bool, svcDef servicedefinition.ServiceDefinition, updateStatus func(string)) (string, error) {
	// create the new service object
	newsvc, err := service.BuildService(svcDef, parentServiceID, poolID, int(service.SVCStop), deploymentID)
	if err != nil {
		glog.Errorf("Could not create service: %s", err)
		return "", err
	}

	updateStatus("deploy_loading_service|" + newsvc.Name)
	if err = f.evaluateEndpointTemplates(ctx, newsvc); err != nil {
		glog.Errorf("Could not evaluate endpoint templates for service %s with parent %s: %s", newsvc.Name, newsvc.ParentServiceID, err)
		return "", err
	}

	if tenantID == "" {
		tenantID = newsvc.ID
	}
	if svcDef.ImageID != "" {
		updateStatus("deploy_loading_image|" + newsvc.Name)
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
		if _, err := f.deployService(ctx, tenantID, newsvc.ID, deploymentID, poolID, overwrite, sd, updateStatus); err != nil {
			glog.Errorf("Error while trying to deploy %s at %s (%s): %s", sd.Name, newsvc.Name, newsvc.ID, err)
			return newsvc.ID, err
		}
	}
	return newsvc.ID, nil
}

func (f *Facade) evaluateEndpointTemplates(ctx datastore.Context, newsvc *service.Service) error {
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

	return newsvc.EvaluateEndpointTemplates(getService, findChildService, 0)
}
