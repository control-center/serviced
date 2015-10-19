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

package dao

import (
	"fmt"
	"strconv"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/zenoss/glog"
)

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceID      string
	Application    string
	ContainerPort  uint16
	HostPort       uint16
	HostIP         string
	ContainerIP    string
	Protocol       string
	VirtualAddress string
	InstanceID     int
	ProxyPort      uint16
}

// BuildApplicationEndpoint converts a ServiceEndpoint to an ApplicationEndpoint
func BuildApplicationEndpoint(state *servicestate.ServiceState, endpoint *service.ServiceEndpoint) (ApplicationEndpoint, error) {
	var ae ApplicationEndpoint

	ae.ServiceID = state.ServiceID
	ae.Application = endpoint.Application
	ae.Protocol = endpoint.Protocol
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
