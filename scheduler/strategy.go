// Copyright 2015 The Serviced Authors.
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

package scheduler

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/scheduler/strategy"
	"github.com/zenoss/glog"
)

// Verify we implement all the interfaces
var (
	_ strategy.Host          = &StrategyHost{}
	_ strategy.ServiceConfig = &StrategyRunningService{}
	_ strategy.ServiceConfig = &StrategyService{}
)

type StrategyHost struct {
	host     *host.Host
	services []strategy.ServiceConfig
}

type StrategyRunningService struct {
	svc dao.RunningService
}

type StrategyService struct {
	svc *service.Service
}

func StrategySelectHost(svc *service.Service, hosts []*host.Host, strat strategy.Strategy, facade *facade.Facade) (*host.Host, error) {

	glog.V(2).Infof("Applying %s strategy for service %s", strat.Name(), svc.ID)

	hostmap := map[string]*StrategyHost{}
	hostids := []string{}

	for _, h := range hosts {
		// Create a StrategyHost for the host
		hostmap[h.ID] = &StrategyHost{h, []strategy.ServiceConfig{}}
		// Save off the hostid
		hostids = append(hostids, h.ID)
	}

	// Look up all running services for the hosts
	glog.V(2).Infof("Looking up instances for hosts: %+v", hostids)
	svcs, err := facade.GetRunningServicesForHosts(datastore.Get(), hostids...)
	if err != nil {
		return nil, err
	}
	// Assign the services to the StrategyHosts
	for _, s := range svcs {
		if h, ok := hostmap[s.HostID]; ok {
			h.services = append(h.services, &StrategyRunningService{s})
		}
	}
	shosts := []strategy.Host{}
	for _, h := range hostmap {
		glog.V(2).Infof("Host %s is running %d service instances", h.HostID(), len(h.services))
		shosts = append(shosts, h)
	}
	if result, err := strat.SelectHost(&StrategyService{svc}, shosts); result == nil || err != nil {
		return nil, err
	} else {
		h := result.(*StrategyHost).host
		glog.V(2).Infof("Deploying service %s to host %s", svc.ID, h.ID)
		return h, nil
	}
}

// Implement everything

func (h *StrategyHost) HostID() string {
	return h.host.ID
}

func (h *StrategyHost) RunningServices() []strategy.ServiceConfig {
	return h.services
}

func (h *StrategyHost) TotalCores() int {
	return h.host.Cores
}

func (h *StrategyHost) TotalMemory() uint64 {
	return h.host.TotalRAM()
}

func (s *StrategyService) GetServiceID() string {
	return s.svc.ID
}

func (s *StrategyService) RequestedCorePercent() int {
	return int(s.svc.CPUCommitment)
}

func (s *StrategyService) RequestedMemoryBytes() uint64 {
	return s.svc.RAMCommitment.Value
}

func (s *StrategyService) HostPolicy() servicedefinition.HostPolicy {
	return s.svc.HostPolicy
}

func (s *StrategyRunningService) GetServiceID() string {
	return s.svc.ServiceID
}

func (s *StrategyRunningService) RequestedCorePercent() int {
	return int(s.svc.CPUCommitment)
}

func (s *StrategyRunningService) RequestedMemoryBytes() uint64 {
	return s.svc.RAMCommitment.Value
}

func (s *StrategyRunningService) HostPolicy() servicedefinition.HostPolicy {
	return s.svc.HostPolicy
}
