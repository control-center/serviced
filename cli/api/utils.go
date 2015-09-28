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

package api

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"

	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

const (
	networkAccessDelay    = 1
	networkAccessAttempts = 90
)

var (
	empty     interface{}
	unusedInt int
)

// GetAgentIP returns the agent ip address
func GetAgentIP(defaultRPCPort int) string {
	if options.Endpoint != "" {
		return options.Endpoint
	}

	var agentIP string
	var err error
	for i := 1; i <= networkAccessAttempts; i++ {
		if agentIP, err = utils.GetIPAddress(); err == nil {
			return agentIP + fmt.Sprintf(":%d", defaultRPCPort)
		}
		glog.Info("Waiting for network initialization...")
		time.Sleep(networkAccessDelay * time.Second)
	}

	glog.Fatalf("Gave up waiting for network (to determine our outbound IP address): %s", err)
	return ""
}

// GetDockerDNS returns the docker dns address
func GetDockerDNS() []string {
	if len(options.DockerDNS) > 0 {
		return options.DockerDNS
	}

	dockerdns := os.Getenv("SERVICED_DOCKER_DNS")
	return strings.Split(dockerdns, ",")
}

// GetVarPath returns the serviced varpath
func GetVarPath() string {
	if options.VarPath != "" {
		return options.VarPath
	} else if home := os.Getenv("SERVICED_HOME"); home != "" {
		return path.Join(home, "var")
	} else if user, err := user.Current(); err == nil {
		return path.Join(os.TempDir(), "serviced-"+user.Username, "var")
	}
	return path.Join(os.TempDir(), "serviced")
}

// GetGateway returns the default gateway
func GetGateway(defaultRPCPort int) string {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	localhost := URL{"127.0.0.1", defaultRPCPort}

	if err != nil {
		glog.V(2).Info("Error checking gateway: ", err)
		glog.V(1).Info("Could not get gateway using ", localhost.Host)
		return localhost.String()
	}

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "default" {
			endpoint := URL{fields[2], defaultRPCPort}
			return endpoint.String()
		}
	}
	glog.V(1).Info("No gateway found, using ", localhost.Host)
	return localhost.String()
}

type version []int

func (a version) String() string {
	var format = make([]string, len(a))
	for idx, value := range a {
		format[idx] = fmt.Sprintf("%d", value)
	}
	return strings.Join(format, ".")
}

func (a version) Compare(b version) int {
	astr := ""
	for _, s := range a {
		astr += fmt.Sprintf("%12d", s)
	}
	bstr := ""
	for _, s := range b {
		bstr += fmt.Sprintf("%12d", s)
	}

	if astr > bstr {
		return -1
	} else if astr < bstr {
		return 1
	} else {
		return 0
	}
}
