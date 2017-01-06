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
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/health"

	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/user"
)

// The FacadeInterface is the API for a Facade
type FacadeInterface interface {
	AddService(ctx datastore.Context, svc service.Service) error

	GetService(ctx datastore.Context, id string) (*service.Service, error)

	// Get a service from serviced where all templated properties have been evaluated
	GetEvaluatedService(ctx datastore.Context, servicedID string, instanceID int) (*service.Service, error)

	GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error)

	GetTaggedServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error)

	GetTenantID(ctx datastore.Context, serviceID string) (string, error)

	SyncServiceRegistry(ctx datastore.Context, svc *service.Service) error

	MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error

	RemoveService(ctx datastore.Context, id string) error

	ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, synchronous bool, desiredState service.DesiredState) (int, error)

	UpdateService(ctx datastore.Context, svc service.Service) error

	WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error

	AssignIPs(ctx datastore.Context, assignmentRequest addressassignment.AssignmentRequest) (err error)

	AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate, reloadLogstashConfig bool) (string, error)

	GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error)

	RemoveServiceTemplate(ctx datastore.Context, templateID string) error

	UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate, reloadLogstashConfig bool) error

	DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error)

	DeployTemplateActive() (active []map[string]string, err error)

	DeployTemplateStatus(deploymentID string, lastStatus string, timeout time.Duration) (status string, err error)

	AddHost(ctx datastore.Context, entity *host.Host) ([]byte, error)

	GetHost(ctx datastore.Context, hostID string) (*host.Host, error)

	GetHosts(ctx datastore.Context) ([]host.Host, error)

	GetHostKey(ctx datastore.Context, hostID string) ([]byte, error)

	ResetHostKey(ctx datastore.Context, hostID string) ([]byte, error)

	RegisterHostKeys(ctx datastore.Context, entity *host.Host, keys []byte, prompt bool) error

	SetHostExpiration(ctx datastore.Context, hostID string, expiration int64)

	RemoveHostExpiration(ctx datastore.Context, hostID string)

	HostIsAuthenticated(ctx datastore.Context, hostid string) (bool, error)

	GetActiveHostIDs(ctx datastore.Context) ([]string, error)

	UpdateHost(ctx datastore.Context, entity *host.Host) error

	RemoveHost(ctx datastore.Context, hostID string) error

	FindHostsInPool(ctx datastore.Context, poolID string) ([]host.Host, error)

	AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error

	GetResourcePool(ctx datastore.Context, poolID string) (*pool.ResourcePool, error)

	GetResourcePools(ctx datastore.Context) ([]pool.ResourcePool, error)

	GetPoolIPs(ctx datastore.Context, poolID string) (*pool.PoolIPs, error)

	HasIP(ctx datastore.Context, poolID string, ipAddr string) (bool, error)

	RemoveResourcePool(ctx datastore.Context, id string) error

	UpdateResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error

	GetHealthChecksForService(ctx datastore.Context, id string) (map[string]health.HealthCheck, error)

	AddPublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string, usetls bool, protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error)

	RemovePublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string) error

	EnablePublicEndpointPort(ctx datastore.Context, serviceid, endpointName, portAddr string, isEnabled bool) error

	AddPublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhost string, isEnabled, restart bool) (*servicedefinition.VHost, error)

	RemovePublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhost string) error

	EnablePublicEndpointVHost(ctx datastore.Context, serviceid, endpointName, vhost string, isEnabled bool) error

	GetHostInstances(ctx datastore.Context, since time.Time, hostid string) ([]service.Instance, error)

	GetServiceInstances(ctx datastore.Context, since time.Time, serviceid string) ([]service.Instance, error)

	GetAggregateServices(ctx datastore.Context, since time.Time, serviceids []string) ([]service.AggregateService, error)

	GetReadPools(ctx datastore.Context) ([]pool.ReadPool, error)

	GetReadHosts(ctx datastore.Context) ([]host.ReadHost, error)

	FindReadHostsInPool(ctx datastore.Context, poolID string) ([]host.ReadHost, error)

	GetAllServiceDetails(ctx datastore.Context, since time.Duration) ([]service.ServiceDetails, error)

	GetServiceDetails(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error)

	GetServiceDetailsAncestry(ctx datastore.Context, serviceID string) (*service.ServiceDetails, error)

	GetServiceDetailsByParentID(ctx datastore.Context, serviceID string, since time.Duration) ([]service.ServiceDetails, error)

	GetServiceDetailsByTenantID(ctx datastore.Context, tenantID string) ([]service.ServiceDetails, error)

	GetServiceMonitoringProfile(ctx datastore.Context, serviceID string) (*domain.MonitorProfile, error)

	GetServicePublicEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.PublicEndpoint, error)

	GetAllPublicEndpoints(ctx datastore.Context) ([]service.PublicEndpoint, error)

	GetServiceAddressAssignmentDetails(ctx datastore.Context, serviceID string, children bool) ([]service.IPAssignment, error)

	GetServiceExportedEndpoints(ctx datastore.Context, serviceID string, children bool) ([]service.ExportedEndpoint, error)

	AddUser(ctx datastore.Context, newUser user.User) error

	GetUser(ctx datastore.Context, userName string) (user.User, error)

	UpdateUser(ctx datastore.Context, u user.User) error

	RemoveUser(ctx datastore.Context, userName string) error

	GetSystemUser(ctx datastore.Context) (user.User, error)

	ValidateCredentials(ctx datastore.Context, u user.User) (bool, error)

	GetServicesHealth(ctx datastore.Context) (map[string]map[int]map[string]health.HealthStatus, error)

	ReportHealthStatus(key health.HealthStatusKey, value health.HealthStatus, expires time.Duration)

	ReportInstanceDead(serviceID string, instanceID int)

	GetServiceConfigs(ctx datastore.Context, serviceID string) ([]service.Config, error)

	GetServiceConfig(ctx datastore.Context, fileID string) (*servicedefinition.ConfigFile, error)

	AddServiceConfig(ctx datastore.Context, serviceID string, conf servicedefinition.ConfigFile) error

	UpdateServiceConfig(ctx datastore.Context, fileID string, conf servicedefinition.ConfigFile) error

	DeleteServiceConfig(ctx datastore.Context, fileID string) error

	GetHostStatuses(ctx datastore.Context, hostIDs []string, since time.Time) ([]host.HostStatus, error)

	UpdateServiceCache(ctx datastore.Context) error

	CountDescendantStates(ctx datastore.Context, serviceID string) (map[string]map[int]int, error)

	ReloadLogstashConfig(ctx datastore.Context) error

	EmergencyStopService(ctx datastore.Context, request dao.ScheduleServiceRequest) (int, error)

	ClearEmergencyStopFlag(ctx datastore.Context, serviceID string) (int, error)
}
