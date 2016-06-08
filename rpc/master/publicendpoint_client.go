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

package master

import (
	"github.com/control-center/serviced/domain/servicedefinition"
)

// Defines a request to add a port public endpoints to a service
type PublicEndpointPortRequest struct {
	Serviceid    string
	EndpointName string
	PortAddr     string
	UseTLS       bool
	Protocol     string
	IsEnabled    bool
	Restart      bool
}

// Adds a port public endpoint to a service.
func (c *Client) AddPublicEndpointPort(serviceid, endpointName, portAddr string, usetls bool,
	protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
	request := &PublicEndpointPortRequest{
		Serviceid:    serviceid,
		EndpointName: endpointName,
		PortAddr:     portAddr,
		UseTLS:       usetls,
		Protocol:     protocol,
		IsEnabled:    isEnabled,
		Restart:      restart,
	}
	var result servicedefinition.Port
	err := c.call("AddPublicEndpointPort", request, &result)
	return &result, err
}
