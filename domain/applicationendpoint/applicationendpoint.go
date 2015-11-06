// Copyright 2015 The Serviced Authors.
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

package applicationendpoint

import (
	"fmt"
	"strconv"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
	"strings"
)

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceID      string
	InstanceID     int
	Application    string
	Purpose        string
	HostID         string
	HostIP         string
	HostPort       uint16
	ContainerID    string
	ContainerIP    string
	ContainerPort  uint16
	Protocol       string
	VirtualAddress string
	ProxyPort      uint16
}

type EndpointReport struct {
	Endpoint ApplicationEndpoint

	// FIXME: Refactor into some kind of array of typed messages (e.g. info, warn and error)
	Messages []string
}

// BuildApplicationEndpoint converts a ServiceEndpoint to an ApplicationEndpoint
func BuildApplicationEndpoint(state *servicestate.ServiceState, endpoint *service.ServiceEndpoint) (ApplicationEndpoint, error) {
	var ae ApplicationEndpoint

	ae.ServiceID = state.ServiceID
	ae.Application = endpoint.Application
	ae.Protocol = endpoint.Protocol
	ae.Purpose = endpoint.Purpose
	ae.ContainerID = state.DockerID
	ae.ContainerIP = state.PrivateIP
	if endpoint.PortTemplate != "" {
		port, err := state.EvalPortTemplate(endpoint.PortTemplate)
		if err != nil {
			glog.Errorf("%s", err)
		} else {
			ae.ContainerPort = port
		}
	} else {
		// No dynamic port, just use the specified PortNumber
		ae.ContainerPort = endpoint.PortNumber
	}
	ae.HostID = state.HostID
	ae.HostIP = state.HostIP
	if len(state.PortMapping) > 0 {
		pmKey := fmt.Sprintf("%d/%s", ae.ContainerPort, ae.Protocol)
		pm := state.PortMapping[pmKey]
		if len(pm) > 0 {
			port, err := strconv.Atoi(pm[0].HostPort)
			if err != nil {
				glog.Errorf("Unable to interpret HostPort: %s: %s", pm[0].HostPort, err)
				return ae, err
			}
			ae.HostPort = uint16(port)
		}
	}
	ae.VirtualAddress = endpoint.VirtualAddress
	ae.InstanceID = state.InstanceID

	glog.V(2).Infof("  built ApplicationEndpoint: %+v", ae)

	return ae, nil
}

// BuildEndpointReports converts an array of ApplicationEndpoints to an array of EndpointReports
func BuildEndpointReports(appEndpoints []ApplicationEndpoint) []EndpointReport {
	endpoints := make([]EndpointReport, 0)
	for _, appEndpoint := range appEndpoints {
		endpoints = append(endpoints, EndpointReport{Endpoint: appEndpoint, Messages: []string{}})
	}
	return endpoints
}

// Returns a string which uniquely identifies an endpoint instance
func (endpoint *ApplicationEndpoint) GetID() string {
	return strings.ToLower(fmt.Sprintf("%s/%d %s %s", endpoint.ServiceID, endpoint.InstanceID, endpoint.Purpose, endpoint.Application))
}

// Find the entry in endpoints which matches the specified endpoint
func (endpoint *ApplicationEndpoint) Find(endpoints []ApplicationEndpoint) *ApplicationEndpoint {
	// Yes, this is brute-force linear search, but in practice the lists should be small, few 10s at most
	endpointID := endpoint.GetID()
	for _, entry := range endpoints {
		if entry.GetID() == endpointID {
			return &entry
		}
	}
	return nil
}

// Equals verifies whether two endpoint objects are equal
func (a *ApplicationEndpoint) Equals(b *ApplicationEndpoint) bool {
	if a.ServiceID != b.ServiceID {
		return false
	}
	if a.InstanceID != b.InstanceID {
		return false
	}
	if a.Application != b.Application {
		return false
	}
	if a.Purpose != b.Purpose {
		return false
	}
	if a.HostID != b.HostID {
		return false
	}
	if a.HostIP != b.HostIP {
		return false
	}
	if a.HostPort != b.HostPort {
		return false
	}
	if a.ContainerID != b.ContainerID {
		return false
	}
	if a.ContainerIP != b.ContainerIP {
		return false
	}
	if a.ContainerPort != b.ContainerPort {
		return false
	}
	if a.Protocol != b.Protocol {
		return false
	}
	if a.VirtualAddress != b.VirtualAddress {
		return false
	}
	return true
}

// ApplicationEndpointSlice is an ApplicationEndpoint array sortable by ServiceID, InstanceID, and Application
type ApplicationEndpointSlice []ApplicationEndpoint

func (s ApplicationEndpointSlice) Len() int {
	return len(s)
}

func (s ApplicationEndpointSlice) Less(i, j int) bool {
	return s[i].GetID() < s[j].GetID()
}

func (s ApplicationEndpointSlice) Swap(i, j int) {
	s[i], s[j] = s[j], s[i]
}
