// Copyright 2020 The Serviced Authors.
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

package main

import (
	// log "github.com/Sirupsen/logrus"
	"fmt"
	"os"
	"time"

	"github.com/control-center/serviced/dao"
	// "github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/service"
	// definition "github.com/control-center/serviced/domain/servicedefinition"
	// "github.com/control-center/serviced/utils"
)

// MigrationContext is an entity for migrating services
type MigrationContext struct {
	services *ServiceList
	filters  map[string]logfilter.LogFilter
}

// NewMigrationContext returns a new MigrationContext entity.
func NewMigrationContext(services *ServiceList, filters map[string]logfilter.LogFilter) *MigrationContext {
	return &MigrationContext{
		services: services,
		filters:  filters,
	}
}

// Migrate performs a batch migration on a group of services.
func (mc *MigrationContext) Migrate(req dao.ServiceMigrationRequest) (ServiceList, error) {

	var err error

	if err = mc.validate(req); err != nil {
		fmt.Fprintln(os.Stderr, "migrate: validation failed")
		return nil, err
	}

	// Migrate log filters
	for _, filter := range req.LogFilters {
		existing, found := mc.filters[filter.Name]
		if !found {
			mc.filters[filter.Name] = filter
		} else {
			existing.Filter = filter.Filter
		}
	}

	// Process modified services
	for _, svc := range req.Modified {
		if err := mc.Update(svc); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: service failed to update.  svc.Name=%v\n", svc.Name)
			return nil, err
		}
	}

	// Process newly added services
	for _, svc := range req.Added {
		if err := mc.Add(svc); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: new service not added.  svc.Name=%v\n", svc.Name)
			return nil, err
		}
	}

	// Process deployment requests for services from service definitions.
	for _, sdreq := range req.Deploy {
		if err := mc.Deploy(sdreq); err != nil {
			fmt.Fprintf(os.Stderr, "migrate: new service not deployed.  svc.Name=%v\n", sdreq.Service.Name)
			return nil, err
		}
	}

	return *mc.services, nil
}

// Update migrates an existing service; return error if the service does not exist
func (mc *MigrationContext) Update(updated *service.Service) error {
	// Updating an existing service effectively replaces the existing service with the given service.

	var err error
	var original *service.Service

	original, err = mc.services.Get(updated.ID)
	if err != nil {
		return err
	}

	if updated.OriginalConfigs == nil {
		updated.OriginalConfigs = original.OriginalConfigs
	}
	if updated.ConfigFiles == nil {
		updated.ConfigFiles = original.ConfigFiles
	}

	updated.UpdatedAt = time.Now()

	// The Put function will overwrite an existing service.
	mc.services.Put(*updated)

	return nil
}

// Add adds a service; return error if service already exists
func (mc *MigrationContext) Add(svc *service.Service) error {
	// Adding a service just adds the service.
	// The service will behave as if it had been already deployed.

	// var err error

	// svc.ID, err = utils.NewUUID36()
	// if err != nil {
	// 	return err
	// }

	// if err := mc.validateAdd(svc); err != nil {
	// 	return err
	// }

	svc.UpdatedAt = time.Now()
	svc.CreatedAt = svc.UpdatedAt

	mc.services.Append(*svc)

	return nil
}

// Deploy converts a service definition to a service and deploys it under a specific service.
func (mc *MigrationContext) Deploy(req *dao.ServiceDeploymentRequest) error {
	var parent *service.Service
	var err error

	// Get the tenant ID (this is also the ID of the root service)
	tenantID := mc.services.GetTenantID()
	if tenantID == "" {
		return fmt.Errorf("No tenant ID found")
	}

	if req.ParentID == "" {
		return fmt.Errorf("No parent service ID specified")
	}

	// Get the parent service
	parent, err = mc.services.Get(req.ParentID)
	if err != nil {
		return err
	}

	// Do some pool validation
	var poolID = parent.PoolID
	if req.PoolID != "" {
		poolID = req.PoolID
	}
	if poolID == "" {
		poolID = parent.PoolID
	}

	err = mc.validateName(req.Service.Name, req.ParentID)
	if err != nil {
		return err
	}

	return deploy(
		mc.services,
		tenantID,
		poolID,
		parent.DeploymentID,
		parent.ID,
		true,
		req.Service,
	)
}
