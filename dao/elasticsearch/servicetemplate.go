// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/servicetemplate"
)

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateID *string) error {
	id, err := this.facade.AddServiceTemplate(datastore.Get(), serviceTemplate)
	*templateID = id
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template servicetemplate.ServiceTemplate, unused *int) error {
	return this.facade.UpdateServiceTemplate(datastore.Get(), template)
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	return this.facade.RemoveServiceTemplate(datastore.Get(), id)
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]*servicetemplate.ServiceTemplate) error {
	templatemap, err := this.facade.GetServiceTemplates(datastore.Get())
	*templates = templatemap
	return err
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantID *string) error {
	var err error
	*tenantID, err = this.facade.DeployTemplate(datastore.Get(), request.PoolID, request.TemplateID, request.DeploymentID)
	return err
}

func (this *ControlPlaneDao) DeployService(request dao.ServiceDeploymentRequest, serviceID *string) error {
	var err error
	*serviceID, err = this.facade.DeployService(datastore.Get(), request.ParentID, request.Service)
	return err
}
