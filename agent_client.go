// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package serviced

import (
	"net/rpc"
)

// AgentClient is an interface that the serviced agent implements to provide
// information about the host it is running on.
type AgentClient struct {
	addr      string
	rpcClient *rpc.Client
}

// Create a new AgentClient.
func NewAgentClient(addr string) (s *AgentClient, err error) {
	s = new(AgentClient)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}
