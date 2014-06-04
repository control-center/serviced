// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/domain/host"

	"os/exec"
)

// NewServer returns a new AgentServer
func NewServer(staticIPs []string) *AgentServer {
	// make our own copy of the slice of ips
	ips := make([]string, len(staticIPs))
	copy(staticIPs, ips)
	return &AgentServer{
		staticIPs: staticIPs,
	}
}

// AgentServer The type is the API for a serviced agent. Get the host information from an agent.
type AgentServer struct {
	staticIPs []string
}

//BuildHostRequest request to build a new host. IP and IPResources will be validated to ensure they exist
//on the host. If IPResources is not set and IPResource using the IP parameter will be used
type BuildHostRequest struct {
	IP     string //IP for the host
	Port   int    //Port to contact the host on
	PoolID string //Pool to set on host
}

// BuildHost creates a Host object from the current host.
func (a *AgentServer) BuildHost(request BuildHostRequest, hostResponse *host.Host) error {

	glog.Infof("local static ips %v [%d]", a.staticIPs, len(a.staticIPs))
	h, err := host.Build(request.IP, request.PoolID, a.staticIPs...)
	if err != nil {
		return err
	}
	*hostResponse = *h
	return nil
}

func (a *AgentServer) GetDockerLogs(dockerID string, logs *string) error {
	cmd := exec.Command("docker", "logs", dockerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	if len(output) > 10000 {
		output = output[len(output)-10000:]
	}
	*logs = string(output)
	return nil
}
