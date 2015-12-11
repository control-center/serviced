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
	"github.com/zenoss/glog"

	"fmt"
	"os/exec"
)

// NewServer returns a new AgentServer
func NewServer(staticIPs []string) *AgentServer {
	// make our own copy of the slice of ips
	ips := make([]string, len(staticIPs))
	copy(ips, staticIPs)
	return &AgentServer{
		staticIPs: ips,
	}
}

// AgentServer The type is the API for a serviced agent. Get the host information from an agent.
type AgentServer struct {
	staticIPs []string
}

//BuildHostRequest request to build a new host. IP and IPResources will be validated to ensure they exist
//on the host. If IPResources is not set and IPResource using the IP parameter will be used
type BuildHostRequest struct {
	IP     string // IP for the host
	Port   int    // Port to contact the host on
	PoolID string // Pool to set on host
	Memory string // Memory allotted to this host
}

// BuildHost creates a Host object from the current host.
func (a *AgentServer) BuildHost(request BuildHostRequest, hostResponse *host.Host) error {
	*hostResponse = host.Host{}

	glog.Infof("local static ips %v [%d]", a.staticIPs, len(a.staticIPs))
	h, err := host.Build(request.IP, fmt.Sprintf("%d", request.Port), request.PoolID, request.Memory, a.staticIPs...)
	if err != nil {
		return err
	}
	if h != nil {
		*hostResponse = *h
	}
	return nil
}

// GetDockerLogs returns the last 2000 lines of logs from the docker container
func (a *AgentServer) GetDockerLogs(dockerID string, logs *string) error {
	cmd := exec.Command("docker", "logs", "--tail=2000", dockerID)
	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Unable to return logs because: %v", err)
		return err
	}
	*logs = string(output)
	return nil
}
