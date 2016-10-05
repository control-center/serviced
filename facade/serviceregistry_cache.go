// Copyright 2016 The Serviced Authors.
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
	"sync"

	zkr "github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/domain/service"
)

type serviceRegistryCache struct {
	mutex     *sync.Mutex
	registry  map[string]*serviceRegistry
}

// serviceRegistry holds cached information used to optimize the registry sync process
type serviceRegistry struct {
	ServiceID   string
	PublicPorts map[zkr.PublicPortKey]zkr.PublicPort
	VHosts      map[zkr.VHostKey]zkr.VHost
}

type ServiceRegistrySyncRequest struct {
	ServiceID       string
	PortsToDelete   []zkr.PublicPortKey
	PortsToPublish  map[zkr.PublicPortKey]zkr.PublicPort
	VHostsToDelete  []zkr.VHostKey
	VHostsToPublish map[zkr.VHostKey]zkr.VHost
}

func NewServiceRegistryCache() *serviceRegistryCache {
	return &serviceRegistryCache{
		mutex:    &sync.Mutex{},
		registry: make(map[string]*serviceRegistry, 0),
	}
}

func (sc *serviceRegistryCache) Lock() {
	sc.mutex.Lock()
}

func (sc *serviceRegistryCache) Unlock() {
	sc.mutex.Unlock()
}

// BuildCache creates a new service registry cache from the specified data
func (sc *serviceRegistryCache) BuildCache(publicPorts map[zkr.PublicPortKey]zkr.PublicPort, vhosts map[zkr.VHostKey]zkr.VHost) {
	sc.registry = make(map[string]*serviceRegistry, 0)
	for key, value := range publicPorts {
		serviceRegistry := sc.GetRegistryForService(value.ServiceID)
		serviceRegistry.PublicPorts[key] = value
	}
	for key, value := range vhosts {
		serviceRegistry := sc.GetRegistryForService(value.ServiceID)
		serviceRegistry.VHosts[key] = value
	}
	return
}

func (sc *serviceRegistryCache) BuildSyncRequest(tenantID string, svc *service.Service) zkr.ServiceRegistrySyncRequest {
	request := zkr.ServiceRegistrySyncRequest{
		ServiceID:      svc.ID,
		PortsToDelete:  []zkr.PublicPortKey{},
		PortsToPublish: make(map[zkr.PublicPortKey]zkr.PublicPort),
		VHostsToDelete: []zkr.VHostKey{},
		VHostsToPublish: make(map[zkr.VHostKey]zkr.VHost),
	}

	for _, ep := range svc.Endpoints {
		// map the public ports
		for _, p := range ep.PortList {
			if p.Enabled {
				key := zkr.PublicPortKey{
					HostID:      "master",
					PortAddress: p.PortAddr,
				}
				pub := zkr.PublicPort{
					TenantID:    tenantID,
					Application: ep.Application,
					ServiceID:   svc.ID,
					Protocol:    p.Protocol,
					UseTLS:      p.UseTLS,
				}
				request.PortsToPublish[key] = pub
			}
		}

		// map the vhosts
		for _, v := range ep.VHostList {
			if v.Enabled {
				key := zkr.VHostKey{
					HostID:    "master",
					Subdomain: v.Name,
				}
				vh := zkr.VHost{
					TenantID:    tenantID,
					Application: ep.Application,
					ServiceID:   svc.ID,
				}
				request.VHostsToPublish[key] = vh
			}
		}
	}

	// Request deletes for any cached values that are not in the maps we just built
	if serviceRegistry, ok := sc.registry[svc.ID]; ok {
		for key, _ := range serviceRegistry.PublicPorts {
			_, ok := request.PortsToPublish[key]
			if !ok {
				request.PortsToDelete = append(request.PortsToDelete, key)
			}
		}
		for key, _ := range serviceRegistry.VHosts {
			_, ok := request.VHostsToPublish[key]
			if !ok {
				request.VHostsToDelete = append(request.VHostsToDelete, key)
			}
		}
	}
	return request
}
func (sc *serviceRegistryCache) UpdateRegistry(serviceID string, publicPorts map[zkr.PublicPortKey]zkr.PublicPort, vhosts map[zkr.VHostKey]zkr.VHost) {
	serviceRegistry := sc.GetRegistryForService(serviceID)
	serviceRegistry.PublicPorts = publicPorts
	serviceRegistry.VHosts = vhosts
}

// getRegistryForService returns the registry entry for the spcified service. If the service is not already in the
// cache, then an empty entry is added to the cache and returned
func (sc *serviceRegistryCache) GetRegistryForService(serviceID string) *serviceRegistry {
	registry, ok := sc.registry[serviceID]
	if !ok {
		registry = &serviceRegistry{
			ServiceID:   serviceID,
			PublicPorts: make(map[zkr.PublicPortKey]zkr.PublicPort, 0),
			VHosts:      make(map[zkr.VHostKey]zkr.VHost, 0),
		}
		sc.registry[serviceID] = registry
	}
	return registry
}
