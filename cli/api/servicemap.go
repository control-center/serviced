// Copyright 2016 The Serviced Authors.
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

package api

import (
	"fmt"

	"github.com/control-center/serviced/domain/service"
)

// ServiceMap maps services by its service id
type ServiceMap map[string]service.Service

// NewServiceMap creates a new service map from a slice of services
func NewServiceMap(services []service.Service) ServiceMap {
	var smap = make(ServiceMap)
	for _, s := range services {
		smap.Add(s)
	}
	return smap
}

// Get gets a service from the service map identified by its service id
func (m ServiceMap) Get(serviceID string) service.Service { return m[serviceID] }

// Add appends a service to the service map
func (m ServiceMap) Add(service service.Service) error {
	if _, ok := m[service.ID]; ok {
		return fmt.Errorf("service already exists")
	}
	m[service.ID] = service
	return nil
}

// Update updates an existing service within the ServiceMap.  If the service
// not exist, it gets created.
func (m ServiceMap) Update(service service.Service) {
	m[service.ID] = service
}

// Remove removes a service from the service map
func (m ServiceMap) Remove(serviceID string) error {
	if _, ok := m[serviceID]; !ok {
		return fmt.Errorf("service not found")
	}
	delete(m, serviceID)
	return nil
}

// Tree returns a map of parent services and its list of children
func (m ServiceMap) Tree() map[string][]string {
	tree := make(map[string][]string)
	for id, svc := range m {
		children := tree[svc.ParentServiceID]
		tree[svc.ParentServiceID] = append(children, id)
	}
	return tree
}
