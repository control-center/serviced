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
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	zkregistry "github.com/control-center/serviced/zzk/registry"
)

type ZZK interface {
	UpdateService(svc *service.Service, setLockOnCreate, setLockOnUpdate bool) error
	RemoveService(svc *service.Service) error
	WaitService(svc *service.Service, state service.DesiredState, cancel <-chan interface{}) error
	GetServiceStates(poolID string, states *[]servicestate.ServiceState, serviceIDs ...string) error
	UpdateServiceState(poolID string, state *servicestate.ServiceState) error
	StopServiceInstance(poolID, hostID, stateID string) error
	CheckRunningPublicEndpoint(publicendpoint zkregistry.PublicEndpointKey, serviceID string) error
	AddHost(_host *host.Host) error
	UpdateHost(_host *host.Host) error
	RemoveHost(_host *host.Host) error
	GetActiveHosts(poolID string, hosts *[]string) error
	UpdateResourcePool(_pool *pool.ResourcePool) error
	RemoveResourcePool(poolID string) error
	AddVirtualIP(vip *pool.VirtualIP) error
	RemoveVirtualIP(vip *pool.VirtualIP) error
	GetRegistryImage(id string) (*registry.Image, error)
	SetRegistryImage(rImage *registry.Image) error
	DeleteRegistryImage(id string) error
	DeleteRegistryLibrary(tenantID string) error
	LockServices(svcs []service.Service) error
	UnlockServices(svcs []service.Service) error
	GetServiceEndpoints(tenantID, serviceID string, endpoints *[]applicationendpoint.ApplicationEndpoint) error
}

func GetFacadeZZK(f *Facade) ZZK {
	return &zkf{f: f}
}
