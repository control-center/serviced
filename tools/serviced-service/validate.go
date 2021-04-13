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
	"fmt"
	"os"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	// "github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/service"
	definition "github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

func (mc *MigrationContext) validate(req dao.ServiceMigrationRequest) error {
	var svcAll []service.Service

	// Validate service updates
	for _, svc := range req.Modified {
		if _, err := mc.validateUpdate(svc); err != nil {
			return err
		}
		svcAll = append(svcAll, *svc)
	}

	// Validate service adds
	for _, svc := range req.Added {
		if err := mc.validateAdd(svc); err != nil {
			return err
		} else if svc.ID, err = utils.NewUUID36(); err != nil {
			return err
		}
		svcAll = append(svcAll, *svc)
	}

	// Validate service deployments
	for _, sdreq := range req.Deploy {
		err := mc.validateDeployment(sdreq.ParentID, &sdreq.Service)
		if err != nil {
			return err
		}
	}

	// Validate service migration
	if err := mc.validateMigration(svcAll, req.ServiceID); err != nil {
		return err
	}

	return nil
}

// validateUpdate enforces constraints on the updated Service.
// The original Service is returned.
func (mc *MigrationContext) validateUpdate(updated *service.Service) (*service.Service, error) {

	var err error
	var original *service.Service

	// Verify that the service being updated exists.
	original, err = mc.services.Get(updated.ID)
	if err != nil {
		return nil, err
	}

	// If the name changed, verify that the new name does not already exist.
	if updated.Name != original.Name {
		if err := mc.validateName(updated.Name, updated.ParentServiceID); err != nil {
			return nil, err
		}
	}

	// Set read-only fields
	updated.CreatedAt = original.CreatedAt
	updated.DeploymentID = original.DeploymentID

	// remove any BuiltIn enabled monitoring configs
	metricConfigs := []domain.MetricConfig{}
	for _, mcfg := range updated.MonitoringProfile.MetricConfigs {
		if mcfg.ID == "metrics" {
			continue
		}
		metrics := []domain.Metric{}
		for _, m := range mcfg.Metrics {
			if !m.BuiltIn {
				metrics = append(metrics, m)
			}
		}
		mcfg.Metrics = metrics
		metricConfigs = append(metricConfigs, mcfg)
	}
	updated.MonitoringProfile.MetricConfigs = metricConfigs

	graphs := []domain.GraphConfig{}
	for _, g := range updated.MonitoringProfile.GraphConfigs {
		if !g.BuiltIn {
			graphs = append(graphs, g)
		}
	}
	updated.MonitoringProfile.GraphConfigs = graphs

	if err := validateServiceOptions(updated); err != nil {
		return nil, err
	}

	return original, nil
}

// Validates that the service doesn't have invalid options specified.
func validateServiceOptions(svc *service.Service) error {
	// ChangeOption RestartAllOnInstanceChanged and HostPolicy RequireSeparate are invalid together.
	var changeOptions = definition.ChangeOptions(svc.ChangeOptions)
	if svc.HostPolicy == definition.RequireSeparate &&
		changeOptions.Contains(definition.RestartAllOnInstanceChanged) {
		return fmt.Errorf(
			"HostPolicy RequireSeparate cannot be used with ChangeOption RestartAllOnInstanceChanged",
		)
	}
	return nil
}

// Ensures the DeploymentID matches the parent's DeploymentID
func (mc *MigrationContext) validateDeploymentID(svc *service.Service) error {
	if svc.ParentServiceID != "" {
		parentSvc, err := mc.services.Get(svc.ParentServiceID)
		if err != nil {
			return err
		}
		svc.DeploymentID = parentSvc.DeploymentID
	}
	return nil
}

func (mc *MigrationContext) validateName(name string, parentID string) error {
	existing, err := mc.services.FindChild(parentID, name)
	if err != nil {
		return err
	}
	if existing != nil {
		path, err := mc.services.GetServicePath(existing.ID)
		if err != nil {
			path = fmt.Sprintf("%v", err)
		}
		return fmt.Errorf("Service already exists on path.  path=%s", path)
	}
	return nil
}

func (mc *MigrationContext) validateAdd(svc *service.Service) error {
	if svc.ParentServiceID == "" {
		return fmt.Errorf("Cannot add a root service  name=%s", svc.Name)
	}

	var err error

	// Verify that the service ID does not already exist
	_, err = mc.services.Get(svc.ID)
	if err != nil {
		return err
	}

	err = mc.validateName(svc.Name, svc.ParentServiceID)
	if err != nil {
		return err
	}

	if err := validateServiceOptions(svc); err != nil {
		return err
	}

	// remove any BuiltIn enabled monitoring configs
	metricConfigs := []domain.MetricConfig{}
	for _, mc := range svc.MonitoringProfile.MetricConfigs {
		if mc.ID == "metrics" {
			continue
		}

		metrics := []domain.Metric{}
		for _, m := range mc.Metrics {
			if !m.BuiltIn {
				metrics = append(metrics, m)
			}
		}
		mc.Metrics = metrics
		metricConfigs = append(metricConfigs, mc)
	}
	svc.MonitoringProfile.MetricConfigs = metricConfigs

	graphs := []domain.GraphConfig{}
	for _, g := range svc.MonitoringProfile.GraphConfigs {
		if !g.BuiltIn {
			graphs = append(graphs, g)
		}
	}
	svc.MonitoringProfile.GraphConfigs = graphs

	// set service defaults
	svc.DesiredState = int(service.SVCStop)         // new services must always be stopped
	svc.CurrentState = string(service.SVCCSStopped) // new services are always stopped

	// manage service configurations
	if svc.OriginalConfigs == nil || len(svc.OriginalConfigs) == 0 {
		if svc.ConfigFiles != nil {
			svc.OriginalConfigs = svc.ConfigFiles
		} else {
			svc.OriginalConfigs = make(map[string]definition.ConfigFile)
		}
	}

	return nil
}

// validateDeployment returns the services that will be deployed
func (mc *MigrationContext) validateDeployment(
	parentID string, sd *definition.ServiceDefinition,
) error {
	var err error

	_, err = mc.services.Get(parentID)
	if err != nil {
		fmt.Fprintf(os.Stderr, "migrate: parent ID for new deployed service not found.  svc.Name=%v\n", sd.Name)
		return err
	}

	return nil
}

// validateMigration makes sure there are no collisions with the added/modified services.
func (mc *MigrationContext) validateMigration(svcs []service.Service, tenantID string) error {
	svcParentMapNameMap := make(map[string]map[string]struct{})
	endpointMap := make(map[string]string)
	for _, svc := range svcs {
		// check for name uniqueness within the set of new/modified/deployed services
		if svcNameMap, ok := svcParentMapNameMap[svc.ParentServiceID]; ok {
			if _, ok := svcNameMap[svc.Name]; ok {
				return fmt.Errorf(
					"Collision for service name %s and parent %s", svc.Name, svc.ParentServiceID,
				)
			}
			svcParentMapNameMap[svc.ParentServiceID][svc.Name] = struct{}{}
		} else {
			svcParentMapNameMap[svc.ParentServiceID] = make(map[string]struct{})
		}

		// check for endpoint name uniqueness within the set of new/modified/deployed services
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				if _, ok := endpointMap[ep.Application]; ok {
					return fmt.Errorf(
						"Endpoint %s in migrated service %s is a duplicate of an endpoint in one "+
							"of the other migrated services",
						ep.Application, svc.Name,
					)
				}
				endpointMap[ep.Application] = svc.ID
			}
		}
	}

	// Check whether migrated services' endpoints conflict with existing services.
	var iter = mc.services.Iterator()
	for iter.Next() {
		svc := iter.Item()
		for _, ep := range svc.Endpoints {
			if ep.Purpose != "export" {
				continue
			}
			newsvcID, ok := endpointMap[ep.Application]
			if !ok {
				continue
			}
			if newsvcID != svc.ID {
				return fmt.Errorf(
					"Endpoint %s in migrated service %s is a duplicate of an endpoint in one "+
						"of the other migrated services",
					ep.Application, svc.Name,
				)
			}
		}
	}
	return nil
}
