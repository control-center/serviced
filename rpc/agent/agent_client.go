// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/serviced/domain/host"

	"net/rpc"
)

// AgentClient is an interface that the serviced agent implements to provide
// information about the host it is running on.
type AgentClient struct {
	addr      string
	rpcClient *rpc.Client
}

// Create a new AgentClient.
func NewClient(addr string) (*AgentClient, error) {
	s := new(AgentClient)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// BuildHost creates a Host object from the current host.
func (a *AgentClient) BuildHost(request BuildHostRequest) (*host.Host, error) {
	hostResponse := host.New()
	if err := a.rpcClient.Call("Master.BuildHost", request, hostResponse); err != nil {
		return nil, err
	}
	return hostResponse, nil
}
