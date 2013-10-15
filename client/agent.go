/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package client

import (
	"net/rpc"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
)

// AgentClient is an interface that the serviced agent implements to provide
// information about the host it is running on.
type AgentClient struct {
	addr      string
	rpcClient *rpc.Client
}

// assert that this implemenents the Agent interface
var _ serviced.Agent = &AgentClient{}

// Create a new AgentClient.
func NewAgentClient(addr string) (s *AgentClient, err error) {
	s = new(AgentClient)
	s.addr = addr
	rpcClient, err := rpc.DialHTTP("tcp", s.addr)
	s.rpcClient = rpcClient
	return s, err
}

// Return the standard host information from the referenced agent.
func (a *AgentClient) GetInfo(unused int, host *dao.Host) error {
	return a.rpcClient.Call("ControlPlaneAgent.GetInfo", unused, host)
}

