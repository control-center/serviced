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
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/zenoss/glog"

	"fmt"
)

// GetMuxConnectionInfoForHost returns mux connection info for the given hostID or nil
func (c *Client) GetMuxConnectionInfoForHost(hostID string) (map[string]proxy.TCPMuxConnectionInfo, error) {
	host, err := c.GetHost(hostID)
	if err != nil {
		glog.Errorf("unable to GetHost for: %s", hostID)
		return nil, err
	}

	address := fmt.Sprintf("%s:%s", host.IPAddr, "4979") // TODO: use host.RPCPort
	client, err := agent.NewClient(address)
	if err != nil {
		return nil, err
	}

	return client.GetMuxConnectionInfo()
}

// GetMuxConnectionInfo returns mux connection info for all hosts
func (c *Client) GetMuxConnectionInfo() (map[string]proxy.TCPMuxConnectionInfo, error) {
	hostids, err := c.GetActiveHostIDs()
	if err != nil {
		glog.Errorf("unable to GetActiveHostIDs")
		return nil, err
	}

	response := make(map[string]proxy.TCPMuxConnectionInfo)
	for _, hostid := range hostids {
		if len(hostid) == 0 {
			continue
		}
		if info, err := c.GetMuxConnectionInfoForHost(hostid); err != nil {
			return nil, err
		} else {
			for k, v := range info {
				response[k] = v
			}
		}
	}

	return response, nil
}
