package utils

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/zenoss/glog"
)

// URL parses and handles URL typed options
type URL struct {
	Host string
	Port int
}

// Set converts a URL string to a URL object
func (u *URL) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad format: %s; must be formatted as HOST:PORT", value)
	}

	u.Host = parts[0]
	if port, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("port does not parse as an integer")
	} else {
		u.Port = port
	}
	return nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%d", u.Host, u.Port)
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
