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

package agent

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/rpc/rpcutils"
)

// Client rpc client to interact with agent
type Client struct {
	addr      string
	rpcClient rpcutils.Client
}

// NewClient Create a new Client.
func NewClient(addr string) (*Client, error) {
	client, err := rpcutils.GetCachedClient(addr)
	if err != nil {
		return nil, err
	}
	s := new(Client)
	s.addr = addr
	s.rpcClient = client
	return s, nil
}

// Close closes rpc client
func (c *Client) Close() (err error) {
	return c.rpcClient.Close()
}

// BuildHost creates a Host object from the current host.
func (c *Client) BuildHost(request BuildHostRequest) (*host.Host, error) {
	hostResponse := host.New()
	if err := c.rpcClient.Call("Agent.BuildHost", request, hostResponse, 0); err != nil {
		return nil, err
	}
	return hostResponse, nil
}

// GetDockerLogs returns the last 10k worth of logs from the docker container
func (c *Client) GetDockerLogs(dockerID string) (string, error) {
	var logs string
	err := c.rpcClient.Call("Agent.GetDockerLogs", dockerID, &logs, 0)
	return logs, err
}
