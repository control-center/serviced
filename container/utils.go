package container

import (
	"os"
	"strings"
)

var hostIPs map[string]struct{}

func init() {
	// Populate the host IPs map
	hostIPs = make(map[string]struct{})
	rawval := os.Getenv("CONTROLPLANE_HOST_IPS")
	// Strip off any quotes, if there
	trimmed := strings.Trim(rawval, `'"`)
	// Fill the set
	for _, ip := range strings.Fields(trimmed) {
		hostIPs[ip] = struct{}{}
	}
}

// isLocalAddress() simply checks the given IP against those passed in as the
// IPs of the host on which this container is running
func isLocalAddress(ip string) bool {
	_, ok := hostIPs[ip]
	return ok
}