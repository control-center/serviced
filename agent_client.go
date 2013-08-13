/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"net/rpc"
)

type AgentClient struct {
	addr      string
	rpcClient *rpc.Client
}

// assert that this implemenents the Agent interface
var _ Agent = &AgentClient{}

// Create a new AgentClient.
func NewAgentClient(addr string) (s *AgentClient, err error) {
	s = new(AgentClient)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

func (a *AgentClient) GetInfo(unused int, host *Host) error {
	return a.rpcClient.Call("ControlPlaneAgent.GetInfo", unused, host)
}
