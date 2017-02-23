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
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/utils"
)

// HostConfig is the deserialized object from the command-line
type HostConfig struct {
	Address *utils.URL
	Nat     *utils.URL
	PoolID  string
	Memory  string
	IPs     []string
}

type HostUpdateConfig struct {
	HostID string
	Memory string
}

type AuthHost struct {
	host.Host
	Authenticated bool
}

func getAuthInfo(client master.ClientInterface, hosts []host.Host) ([]AuthHost, error) {
	hostIDs := []string{}
	for _, h := range hosts {
		hostIDs = append(hostIDs, h.ID)
	}
	authHosts, err := client.HostsAuthenticated(hostIDs)
	if err != nil {
		return nil, err
	}
	hostsWithAuth := []AuthHost{}
	for _, h := range hosts {
		hostsWithAuth = append(hostsWithAuth, AuthHost{h, authHosts[h.ID]})
	}
	return hostsWithAuth, nil
}

// Get host information by its id
func (a *api) GetHostWithAuthInfo(id string) (*AuthHost, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	h, err := client.GetHost(id)
	if err != nil {
		return nil, err
	}
	hostsWithAuth, err := getAuthInfo(client, []host.Host{*h})
	if err != nil {
		return nil, err
	}
	return &hostsWithAuth[0], nil
}

func (a *api) GetHostsWithAuthInfo() ([]AuthHost, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	hosts, err := client.GetHosts()
	if err != nil {
		return nil, err
	}
	return getAuthInfo(client, hosts)
}

// Returns a list of all hosts
func (a *api) GetHosts() ([]host.Host, error) {
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

func (a *api) GetHostMemory(id string) (*metrics.MemoryUsageStats, error) {
	client, err := a.connectDAO()
	if err != nil {
		return nil, err
	}

	req := dao.MetricRequest{
		StartTime: time.Now().Add(-24 * time.Hour),
		HostID:    id,
	}

	var result metrics.MemoryUsageStats
	if err := client.GetHostMemoryStats(req, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

// Adds a new host
func (a *api) AddHost(config HostConfig) (*host.Host, []byte, error) {
	// if a nat is configured then we connect rpc to the nat, otherwise
	// connect to the host address.
	var rpcAddress string
	if len(config.Nat.Host) > 0 {
		rpcAddress = config.Nat.String()
	} else {
		rpcAddress = config.Address.String()
	}
	agentClient, err := a.connectAgent(rpcAddress)
	if err != nil {
		return nil, nil, err
	}

	req := agent.BuildHostRequest{
		IP:      config.Address.Host,
		Port:    config.Address.Port,
		PoolID:  config.PoolID,
		Memory:  config.Memory,
	}

	h, err := agentClient.BuildHost(req)
	if err != nil {
		return nil, nil, err
	}

	masterClient, err := a.connectMaster()
	if err != nil {
		return nil, nil, err
	}

	var privateKey []byte
	if privateKey, err = masterClient.AddHost(*h); err != nil {
		return nil, nil, err
	}

	if host_, err := a.GetHost(h.ID); err != nil {
		return nil, nil, err
	} else {
		return host_, privateKey, nil
	}
}

// Removes an existing host by its id
func (a *api) RemoveHost(id string) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}

	return client.RemoveHost(id)
}

// Sets the memory allocation for an existing host
func (a *api) SetHostMemory(config HostUpdateConfig) error {
	client, err := a.connectMaster()
	if err != nil {
		return err
	}
	h, err := client.GetHost(config.HostID)
	if err != nil {
		return err
	}
	if _, err := host.GetRAMLimit(config.Memory, h.Memory); err != nil {
		return err
	}
	h.RAMLimit = config.Memory
	return client.UpdateHost(*h)
}

func (a *api) AuthenticateHost(hostID string) (string, int64, error) {
	client, err := a.connectMaster()
	if err != nil {
		return "", 0, err
	}
	return client.AuthenticateHost(hostID)
}

// Retrieve host's public key
func (a *api) GetHostPublicKey(id string) ([]byte, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	return client.GetHostPublicKey(id)
}

// Reset a host's key
func (a *api) ResetHostKey(id string) ([]byte, error) {
	client, err := a.connectMaster()
	if err != nil {
		return nil, err
	}
	return client.ResetHostKey(id)
}

// Write delegate keys to disk
func (a *api) RegisterHost(keydata []byte) error {
	return auth.RegisterLocalHost(keydata)
}

func (a *api) RegisterRemoteHost(h *host.Host, nat utils.URL, keyData []byte, prompt bool) error {
	return auth.RegisterRemoteHost(h.ID, nat, h.IPAddr, keyData, prompt)
}

// Output a delegate key file to a given location on disk
func (a *api) WriteDelegateKey(filename string, data []byte) error {
	return auth.WriteKeyToFile(filename, data)
}
