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
	"github.com/docker/docker/pkg/jsonlog"
	"github.com/zenoss/glog"

	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"
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

// getLastDockerLogs gets the last N bytes from the docker logs
func getLastDockerLogs(logfile string, size int64) (output []string, err error) {
	fi, err := os.Open(logfile)
	if err != nil {
		return output, err
	}
	stat, err := fi.Stat()
	if err != nil {
		return output, err
	}

	offset := stat.Size() - size
	if offset < 0 {
		offset = 0
	}
	if _, err := fi.Seek(offset, 0); err != nil {
		return output, err
	}

	output = make([]string, 0)
	reader := bufio.NewReader(fi)
	var dec *json.Decoder
	for {
		l := jsonlog.JSONLog{}
		if dec == nil {
			// keep trying to decode a message on new line boundaries
			// until we are successful
			line, err := reader.ReadBytes('\n')
			if err != nil {
				if err == io.EOF {
					return output, nil
				}
				return output, err
			}
			if err := json.Unmarshal(line, &l); err != nil {
				continue
			}
			dec = json.NewDecoder(reader)
		}
		if err := dec.Decode(&l); err != nil {
			break
		}
		output = append(output, fmt.Sprintf("%s", l.Log))
	}
	return output, nil
}

// GetDockerLogs returns the last 10k worth of logs from the docker container
func (a *AgentServer) GetDockerLogs(dockerID string, logs *string) error {

	//  TODO: revisit this after docker supports truncating logs
	filename := fmt.Sprintf("/var/lib/docker/containers/%s/%s-json.log", dockerID, dockerID)

	output, err := getLastDockerLogs(filename, 20000)
	if err != nil {
		return err
	}
	*logs = strings.Join(output, "")
	return nil
}
