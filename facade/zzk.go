// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	zkservice "github.com/control-center/serviced/zzk/service"
)

type ZZK interface {
	UpdateService(ctx datastore.Context, tenantID string, svc *service.Service, setLockOnCreate, setLockOnUpdate bool) error
	UpdateServices(ctx datastore.Context, tenantID string, svc []*service.Service, setLockOnCreate, setLockOnUpdate bool) error
	SyncServiceRegistry(ctx datastore.Context, tenantID string, svc *service.Service) error
	RemoveService(poolID, serviceID string) error
	RemoveServiceEndpoints(serviceID string) error
	RemoveTenantExports(tenantID string) error
	WaitService(svc *service.Service, state service.DesiredState, cancel <-chan interface{}) error
	WaitInstance(ctx datastore.Context, svc *service.Service, instanceID int, checkInstance func(*zkservice.State, bool) bool, cancel <-chan struct{}) error
	GetPublicPort(portAddress string) (string, string, error)
	GetVHost(subdomain string) (string, string, error)
	AddHost(_host *host.Host) error
	UpdateHost(_host *host.Host) error
	RemoveHost(_host *host.Host) error
	GetActiveHosts(ctx datastore.Context, poolID string, hosts *[]string) error
	IsHostActive(poolID string, hostId string) (bool, error)
	UpdateResourcePool(_pool *pool.ResourcePool) error
	RemoveResourcePool(poolID string) error
	GetRegistryImage(id string) (*registry.Image, error)
	SetRegistryImage(rImage *registry.Image) error
	DeleteRegistryImage(id string) error
	DeleteRegistryLibrary(tenantID string) error
	LockServices(ctx datastore.Context, svcs []service.ServiceDetails) error
	UnlockServices(ctx datastore.Context, svcs []service.ServiceDetails) error
	GetServiceStates(ctx datastore.Context, poolID, serviceID string) ([]zkservice.State, error)
	GetHostStates(ctx datastore.Context, poolID, hostID string) ([]zkservice.State, error)
	GetServiceState(ctx datastore.Context, poolID, serviceID string, instanceID int) (*zkservice.State, error)
	StopServiceInstance(poolID, serviceID string, instanceID int) error
	StopServiceInstances(ctx datastore.Context, poolID, serviceID string) error
	RestartInstance(ctx datastore.Context, poolID, serviceID string, instanceID int) error
	SendDockerAction(poolID, serviceID string, instanceID int, command string, args []string) error
	GetServiceStateIDs(poolID, serviceID string) ([]zkservice.StateRequest, error)
	GetServiceNodes() ([]zkservice.ServiceNode, error)
	RegisterDfsClients(clients ...host.Host) error
	UnregisterDfsClients(clients ...host.Host) error
	GetVirtualIPHostID(poolID, ip string) (string, error)
	UpdateInstanceCurrentState(ctx datastore.Context, poolID, serviceID string, instanceID int, state service.InstanceCurrentState) error
}
