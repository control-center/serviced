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

package api

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
)

// URL parses and handles URL typed options
type URL struct {
	Host string
	Port int
}

// Set converts a URL string to a URL object
func (u *URL) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad format: %s; must be formatted as HOST:PORT", value)
	}

	u.Host = parts[0]
	if port, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("port does not parse as an integer")
	} else {
		u.Port = port
	}
	return nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%d", u.Host, u.Port)
}

// ImageMap parses docker image data
type ImageMap map[string]string

// Set converts a docker image mapping into an ImageMap
func (m *ImageMap) Set(value string) error {
	parts := strings.Split(value, ",")
	if len(parts) != 2 {
		return fmt.Errorf("bad format")
	}

	(*m)[parts[0]] = parts[1]
	return nil
}

func (m *ImageMap) String() string {
	var mapping []string
	for k, v := range *m {
		mapping = append(mapping, k+","+v)
	}

	return strings.Join(mapping, " ")
}

// PortMap parses remote and local port data from the command line
type PortMap map[string]servicedefinition.EndpointDefinition

// Set converts a port mapping string from the command line to a PortMap object
func (m *PortMap) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return fmt.Errorf("bad format: %s; must be PROTOCOL:PORTNUM:PORTNAME", value)
	}
	protocol := parts[0]
	switch protocol {
	case "tcp", "udp":
	default:
		return fmt.Errorf("unsupported protocol: %s (udp|tcp)", protocol)
	}
	portnum, err := strconv.ParseUint(parts[1], 10, 16)
	if err != nil {
		return fmt.Errorf("invalid port number: %s", parts[1])
	}
	portname := parts[2]
	if portname == "" {
		return fmt.Errorf("port name cannot be empty")
	}
	port := fmt.Sprintf("%s:%d", protocol, portnum)
	(*m)[port] = servicedefinition.EndpointDefinition{Protocol: protocol, PortNumber: uint16(portnum), Application: portname}
	return nil
}

func (m *PortMap) String() string {
	var mapping []string
	for _, v := range *m {
		mapping = append(mapping, fmt.Sprintf("%s:%d:%s", v.Protocol, v.PortNumber, v.Application))
	}
	return strings.Join(mapping, " ")
}

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
