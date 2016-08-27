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

import "github.com/control-center/serviced/domain/service"

// StopServiceInstance stops a service instance.
func (c *Client) StopServiceInstance(serviceID string, instanceID int) error {
	req := ServiceInstanceRequest{
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}

	err := c.call("StopServiceInstance", req, new(string))
	return err
}

// LocateServiceInstance returns the location of a service instance
func (c *Client) LocateServiceInstance(serviceID string, instanceID int) (*service.LocationInstance, error) {
	req := ServiceInstanceRequest{
		ServiceID:  serviceID,
		InstanceID: instanceID,
	}
	resp := &service.LocationInstance{}

	err := c.call("LocateServiceInstance", req, resp)
	return resp, err
}

// SendDockerAction submits an action to a docker container
func (c *Client) SendDockerAction(serviceID string, instanceID int, action string, args []string) error {
	req := DockerActionRequest{
		ServiceID:  serviceID,
		InstanceID: instanceID,
		Action:     action,
		Args:       args,
	}

	err := c.call("SendDockerAction", req, new(string))
	return err
}
