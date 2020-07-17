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

package service

import (
	"errors"

	"github.com/control-center/serviced/domain/addressassignment"
	svcdef "github.com/control-center/serviced/domain/servicedefinition"
)

// ServiceEndpoint endpoint exported or imported by a service
type ServiceEndpoint struct {
	Name         string // Human readable name of the endpoint. Unique per service definition
	Purpose      string
	Protocol     string
	PortNumber   uint16
	PortTemplate string // A template which, if specified, is used to calculate the port number

	// VirtualAddress is an address by which an imported endpoint may be accessed within the container.
	// e.g. "mysqlhost:1234"
	VirtualAddress string

	Application         string
	ApplicationTemplate string
	AddressConfig       svcdef.AddressResourceConfig

	// VHost is used to request named vhost for this endpoint.
	// Should be the name of a subdomain, i.e "myapplication"  not "myapplication.host.com"
	VHosts []string

	// VHostList is used to request named vhost(s) for this endpoint.
	VHostList []svcdef.VHost

	AddressAssignment addressassignment.AddressAssignment

	// PortList is the list of enabled/disabled ports to assign to this endpoint.
	PortList []svcdef.Port
}

// BuildServiceEndpoint build a ServiceEndpoint from a EndpointDefinition
func BuildServiceEndpoint(epd svcdef.EndpointDefinition) ServiceEndpoint {
	sep := ServiceEndpoint{}
	sep.Name = epd.Name
	sep.Purpose = epd.Purpose
	sep.Protocol = epd.Protocol
	sep.PortNumber = epd.PortNumber
	sep.PortTemplate = epd.PortTemplate
	sep.VirtualAddress = epd.VirtualAddress
	sep.Application = epd.Application
	sep.ApplicationTemplate = epd.ApplicationTemplate
	sep.AddressConfig = epd.AddressConfig
	sep.VHosts = epd.VHosts
	sep.VHostList = epd.VHostList
	sep.PortList = epd.PortList

	// run public ports through scrubber to allow for "almost correct" port addresses
	for index, port := range sep.PortList {
		sep.PortList[index].PortAddr = ScrubPortString(port.PortAddr)
	}
	return sep
}

// IsConfigurable returns true if the endpoint is configurable
func (e *ServiceEndpoint) IsConfigurable() bool {
	return e.AddressConfig.Port > 0 && e.AddressConfig.Protocol != ""
}

// SetAssignment sets the AddressAssignment for the endpoint
func (e *ServiceEndpoint) SetAssignment(aa addressassignment.AddressAssignment) error {
	if e.AddressConfig.Port == 0 {
		return errors.New("cannot assign address to endpoint without AddressResourceConfig")
	}
	e.AddressAssignment = aa
	return nil
}

// RemoveAssignment resets a service endpoints to nothing
func (e *ServiceEndpoint) RemoveAssignment() error {
	e.AddressAssignment = addressassignment.AddressAssignment{}
	return nil
}

// GetAssignment Returns nil if no assignment set
func (e *ServiceEndpoint) GetAssignment() *addressassignment.AddressAssignment {
	if e.AddressAssignment.ID == "" {
		return nil
	}
	//return reference to copy
	result := e.AddressAssignment
	return &result
}
