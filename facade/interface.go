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
	"github.com/control-center/serviced/health"

	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/domain/servicetemplate"
)

// The FacadeInterface is the API for a Facade
type FacadeInterface interface {
	AddService(ctx datastore.Context, svc service.Service) error

	GetService(ctx datastore.Context, id string) (*service.Service, error)

	GetServices(ctx datastore.Context, request dao.EntityRequest) ([]service.Service, error)

	GetServicesByImage(ctx datastore.Context, imageID string) ([]service.Service, error)

	GetServiceStates(ctx datastore.Context, serviceID string) ([]servicestate.ServiceState, error)

	GetTenantID(ctx datastore.Context, serviceID string) (string, error)

	MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error

	RemoveService(ctx datastore.Context, id string) error

	ScheduleService(ctx datastore.Context, serviceID string, autoLaunch bool, desiredState service.DesiredState) (int, error)

	UpdateService(ctx datastore.Context, svc service.Service) error

	WaitService(ctx datastore.Context, dstate service.DesiredState, timeout time.Duration, recursive bool, serviceIDs ...string) error

	AssignIPs(ctx datastore.Context, assignmentRequest addressassignment.AssignmentRequest) (err error)

	AddServiceTemplate(ctx datastore.Context, serviceTemplate servicetemplate.ServiceTemplate) (string, error)

	GetServiceTemplates(ctx datastore.Context) (map[string]servicetemplate.ServiceTemplate, error)

	RemoveServiceTemplate(ctx datastore.Context, templateID string) error

	UpdateServiceTemplate(ctx datastore.Context, template servicetemplate.ServiceTemplate) error

	DeployTemplate(ctx datastore.Context, poolID string, templateID string, deploymentID string) ([]string, error)

	DeployTemplateActive() (active []map[string]string, err error)

	DeployTemplateStatus(deploymentID string) (status string, err error)

	AddHost(ctx datastore.Context, entity *host.Host) error

	GetHost(ctx datastore.Context, hostID string) (*host.Host, error)

	GetHosts(ctx datastore.Context) ([]host.Host, error)

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
}
