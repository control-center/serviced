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
	"github.com/control-center/serviced/domain/servicedefinition"
)

// Add a new port public endpoint.
func (a *api) AddPublicEndpointPort(serviceid, endpointName, portAddr string, usetls bool,
	protocol string, isEnabled bool, restart bool) (*servicedefinition.Port, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.AddPublicEndpointPort(serviceid, endpointName, portAddr, usetls, protocol, isEnabled, restart)
}

// Remove a port public endpoint.
func (a *api) RemovePublicEndpointPort(serviceid, endpointName, portAddr string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemovePublicEndpointPort(serviceid, endpointName, portAddr)
}

// Enable/Disable a port public endpoint.
func (a *api) EnablePublicEndpointPort(serviceid, endpointName, portAddr string, isEnabled bool) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.EnablePublicEndpointPort(serviceid, endpointName, portAddr, isEnabled)
}

func (a *api) AddPublicEndpointVHost(serviceid, endpointName, vhost string, isEnabled, restart bool) (*servicedefinition.VHost, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.AddPublicEndpointVHost(serviceid, endpointName, vhost, isEnabled, restart)
}
