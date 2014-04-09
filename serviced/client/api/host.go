package api

import (
	"github.com/zenoss/serviced/domain/host"
)

const ()

var ()

// HostConfig is the deserialized object from the command-line
type HostConfig struct {
	IPAddr string
	PoolID string
	IPs    []string
}

// ListHosts returns a list of all hosts
func (a *api) ListHosts() ([]host.Host, error) {
	return nil, nil
}

// GetHost looks up a host by its id
func (a *api) GetHost(id string) (*host.Host, error) {
	return nil, nil
}

// AddHost adds a new host
func (a *api) AddHost(config HostConfig) (*host.Host, error) {
	return nil, nil
}

// RemoveHost removes an existing host by its id
func (a *api) RemoveHost(id string) error {
	return nil
}
