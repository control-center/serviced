package api

import (
	host "github.com/zenoss/serviced/dao"
)

const ()

var ()

// HostConfig is the deserialized object from the command-line
type HostConfig struct {
	Address URL
	PoolID  string
	IPs     []string
}

// ListHosts returns a list of all hosts
func (a *api) ListHosts() ([]host.Host, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	hostmap := make(map[string]*host.Host)
	if err := client.GetHosts(&empty, &hostmap); err != nil {
		return nil, fmt.Errorf("could not get hosts: %s", err)
	}
	hosts := make([]host.Host, len(hostmap))
	i := 0
	for _, h := range hostmap {
		hosts[i] = *h
		i++
	}
	return hosts, nil
}

// GetHost looks up a host by its id
func (a *api) GetHost(id string) (*host.Host, error) {
	client, err := connect()
	if err != nil {
		return nil, err
	}

	hostmap := make(map[string]*host.Host)
	if err := client.GetHosts(&empty, &hostmap); err != nil {
		return nil, fmt.Errorf("could not get hosts: %s", err)
	}
	return hostmap[id], nil
}

// AddHost adds a new host
func (a *api) AddHost(config HostConfig) (*host.Host, error) {
	// Add the IP used to connect
	agentClient, err := serviced.NewAgentClient(string(config.Address))
	if err != nil {
		return nil, fmt.Errorf("could not create host connection: %s", err)
	}
	var remoteHost host.Host
	if err := agentClient.GetInfo(config.IPs, &remoteHost); err != nil {
		return nil, fmt.Errorf("could not get remote host info: %s", err)
	}
	remoteHost.IpAddr = config.Address.Host
	remoteHost.PoolId = config.PoolID

	// Add the host
	glog.V(0).Infof("Got info for host: %v", remoteHost)
	client, err := connect()
	if err != nil {
		return nil, err
	}
	var id string
	if err := client.AddHost(remoteHost, &id); err != nil {
		return nil, fmt.Errorf("could not add host: %s", err)
	}

	// Get information about the host that was added
	var hostmap map[string]*host.Host
	if err := client.GetHosts(&empty, hostmap); err != nil {
		return nil, fmt.Errorf("could not get hosts: %s", err)
	}

	return hostmap[id], nil
}

// RemoveHost removes an existing host by its id
func (a *api) RemoveHost(id string) error {
	client, err := connect()
	if err != nil {
		return err
	}

	if err := client.RemoveHost(id, &unusedInt); err != nil {
		return nil, fmt.Errorf("could not remove host: %s", err)
	}

	return nil
}
