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

package elasticsearch

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicetemplate"
)

func (this *ControlPlaneDao) AddServiceTemplate(serviceTemplate servicetemplate.ServiceTemplate, templateID *string) error {
	id, err := this.facade.AddServiceTemplate(datastore.Get(), serviceTemplate)
	*templateID = id
	return err
}

func (this *ControlPlaneDao) UpdateServiceTemplate(template servicetemplate.ServiceTemplate, _ *int) error {
	return this.facade.UpdateServiceTemplate(datastore.Get(), template)
}

func (this *ControlPlaneDao) RemoveServiceTemplate(id string, unused *int) error {
	return this.facade.RemoveServiceTemplate(datastore.Get(), id)
}

func (this *ControlPlaneDao) GetServiceTemplates(unused int, templates *map[string]servicetemplate.ServiceTemplate) error {
	templatemap, err := this.facade.GetServiceTemplates(datastore.Get())
	if templatemap != nil {
		*templates = templatemap
	} else {
		*templates = make(map[string]servicetemplate.ServiceTemplate, 0)
	}
	return err
}

func (this *ControlPlaneDao) DeployTemplate(request dao.ServiceTemplateDeploymentRequest, tenantIDs *[]string) error {
	var err error
	*tenantIDs, err = this.facade.DeployTemplate(datastore.Get(), request.PoolID, request.TemplateID, request.DeploymentID)
	return err
}

func (this *ControlPlaneDao) DeployTemplateStatus(request dao.ServiceTemplateDeploymentRequest, deployTemplateStatus *string) error {
	var err error
	err = this.facade.DeployTemplateStatus(request.DeploymentID, deployTemplateStatus)
	return err
}

func (this *ControlPlaneDao) DeployTemplateActive(notUsed string, active *[]map[string]string) error {
	var err error
	err = this.facade.DeployTemplateActive(active)
	if active == nil {
		*active = make([]map[string]string, 0)
	}
	return err
}

func (this *ControlPlaneDao) DeployService(request dao.ServiceDeploymentRequest, serviceID *string) (err error) {
	*serviceID, err = this.facade.DeployService(datastore.Get(), request.PoolID, request.ParentID, request.Overwrite, request.Service)
	return
}
