// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/serviced/domain/host"

	"net/rpc"
)

// AgentClient rpc client to interact with agent
type Client struct {
	addr      string
	rpcClient *rpc.Client
}

// Create a new Client.
func NewClient(addr string) (*Client, error) {
	s := new(Client)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// Close closes rpc client
func (c *Client) Close() (err error) {
	return c.rpcClient.Close()
}

// BuildHost creates a Host object from the current host.
func (a *Client) BuildHost(request BuildHostRequest) (*host.Host, error) {
	hostResponse := host.New()
	if err := a.rpcClient.Call("Master.BuildHost", request, hostResponse); err != nil {
		return nil, err
	}
	return hostResponse, nil
}
