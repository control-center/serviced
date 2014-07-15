// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package api

import (
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/utils"
)

const (
	minTimeout     = 30
	defaultTimeout = 600
)

var (
	empty     interface{}
	unusedInt int
)

// GetAgentIP returns the agent ip address
func GetAgentIP() string {
	if options.Endpoint != "" {
		return options.Endpoint
	}
	agentIP, err := utils.GetIPAddress()
	if err != nil {
		panic(err)
	}
	return agentIP + ":4979"
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

// GetESStartupTimeout returns the Elastic Search Startup Timeout
func GetESStartupTimeout() int {
	var timeout int

	if t := options.ESStartupTimeout; t > 0 {
		timeout = options.ESStartupTimeout
	} else if t := os.Getenv("ES_STARTUP_TIMEOUT"); t != "" {
		if res, err := strconv.Atoi(t); err != nil {
			timeout = res
		}
	}

	if timeout == 0 {
		timeout = defaultTimeout
	} else if timeout < minTimeout {
		timeout = minTimeout
	}

	return timeout
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
