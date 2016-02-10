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

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/utils"
)

// HostConfig is the deserialized object from the command-line
type HostConfig struct {
	Address *utils.URL
	PoolID  string
	Memory  string
	IPs     []string
}

type HostUpdateConfig struct {
	HostID string
	Memory string
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
func (a *api) AddHost(config HostConfig) (*host.Host, error) {
	agentClient, err := a.connectAgent(config.Address.String())
	if err != nil {
		return nil, err
	}

	req := agent.BuildHostRequest{
		IP:     config.Address.Host,
		Port:   config.Address.Port,
		PoolID: config.PoolID,
		Memory: config.Memory,
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
