package api

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"strconv"
	"strings"

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
	if options.Port != "" {
		return options.Port
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