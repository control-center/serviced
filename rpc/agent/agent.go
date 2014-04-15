// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/serviced/domain/host"
)

// NewServer returns a new AgentServer
func NewServer() AgentServer {
	return AgentServer{}
}

// AgentServer The type is the API for a serviced agent. Get the host information from an agent.
type AgentServer struct {
}

//BuildHostRequest request to build a new host. IP and IPResources will be validated to ensure they exist
//on the host. If IPResources is not set and IPResource using the IP parameter will be used
type BuildHostRequest struct {
	IP          string   //IP for the host.
	PoolID      string   //Pool to set on host
	IPResources []string //IP strings requested as a resources
}

// BuildHost creates a Host object from the current host.
func (a *AgentServer) BuildHost(request BuildHostRequest, hostResponse *host.Host) error {

	h, err := host.Build(request.IP, request.PoolID, request.IPResources...)
	if err != nil {
		return err
	}
	*hostResponse = *h
	return nil
}
