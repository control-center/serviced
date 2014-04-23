package api

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/rpc/agent"
)

const ()

var ()

// HostConfig is the deserialized object from the command-line
type HostConfig struct {
	Address *URL
	PoolID  string
	IPs     []string
}

// Returns a list of all hosts
func (a *api) GetHosts() ([]*host.Host, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetHosts()
}

// Get host information by its id
func (a *api) GetHost(id string) (*host.Host, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	return client.GetHost(id)
}

// Adds a new host
func (a *api) AddHost(config HostConfig) (*host.Host, error) {
	agentClient, err := a.connectAgent()
	if err != nil {
		return nil, err
	}

	req := agent.BuildHostRequest{
		IP:          config.Address.String(),
		PoolID:      config.PoolID,
		IPResources: config.IPs,
	}

	h, err := agentClient.BuildHost(req)
	if err != nil {
		return nil, err
	}

	masterClient, err := a.connectMaster()
	if err != nil {
		return nil, err
	}

	if err := masterClient.AddHost(*h); err != nil {
		return nil, err
	}

	return a.GetHost(h.ID)
}

// Removes an existing host by its id
func (a *api) RemoveHost(id string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveHost(id)
}
