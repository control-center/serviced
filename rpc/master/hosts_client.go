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

package master

import (
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/domain/host"
)

//GetHost gets the host for the given hostID or nil
func (c *Client) GetHost(hostID string) (*host.Host, error) {
	response := host.New()
	if err := c.call("GetHost", hostID, response); err != nil {
		return nil, err
	}
	return response, nil
}

//GetHosts returns all hosts or empty array
func (c *Client) GetHosts() ([]host.Host, error) {
	response := make([]host.Host, 0)
	if err := c.call("GetHosts", empty, &response); err != nil {
		return []host.Host{}, err
	}
	return response, nil
}

//GetActiveHosts returns all active host ids or empty array
func (c *Client) GetActiveHostIDs() ([]string, error) {
	response := []string{}
	if err := c.call("GetActiveHostIDs", empty, &response); err != nil {
		return []string{}, err
	}
	return response, nil
}

//AddHost adds a Host
func (c *Client) AddHost(host host.Host) ([]byte, error) {
	response := []byte{}
	if err := c.call("AddHost", host, &response); err != nil {
		return []byte{}, err
	}
	return response, nil
}

//UpdateHost updates a host
func (c *Client) UpdateHost(host host.Host) error {
	return c.call("UpdateHost", host, nil)
}

//RemoveHost removes a host
func (c *Client) RemoveHost(hostID string) error {
	return c.call("RemoveHost", hostID, nil)
}

//FindHostsInPool returns all hosts in a pool
func (c *Client) FindHostsInPool(poolID string) ([]host.Host, error) {
	response := make([]host.Host, 0)
	if err := c.call("FindHostsInPool", poolID, &response); err != nil {
		return []host.Host{}, err
	}
	return response, nil
}

// AuthenticateHost authenticates a host
func (c *Client) AuthenticateHost(hostID string) (string, int64, error) {
	req := HostAuthenticationRequest{
		HostID:  hostID,
		Expires: time.Now().Add(time.Duration(1 * time.Minute)).UTC().Unix(),
	}
	sig, err := auth.SignAsDelegate(req.toMessage())
	if err != nil {
		return "", 0, err
	}
	req.Signature = sig
	var response HostAuthenticationResponse
	if err := c.call("AuthenticateHost", req, &response); err != nil {
		return "", 0, err
	}
	return response.Token, response.Expires, nil
}
