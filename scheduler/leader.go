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

package scheduler

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/commons"
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/dfs/ttl"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/scheduler/strategy"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

type leader struct {
	shutdown <-chan interface{}
	conn     coordclient.Connection
	cpClient dao.ControlPlane
	facade   *facade.Facade
	poolID   string
}

// Lead is executed by the "leader" of the control center cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(shutdown <-chan interface{}, conn coordclient.Connection, cpClient dao.ControlPlane, facade *facade.Facade, poolID string, snapshotTTL int) {

	glog.V(0).Info("Processing leader duties")
	leader := leader{shutdown, conn, cpClient, facade, poolID}

	// creates a listener for the host registry
	hostRegistry := zkservice.NewHostRegistryListener()

	// creates a listener for services
	serviceListener := zkservice.NewServiceListener(&leader)

	// kicks off the snapshot cleaning goroutine
	if snapshotTTL > 0 {
		go ttl.RunSnapshotTTL(cpClient, shutdown, time.Minute, time.Duration(snapshotTTL)*time.Hour)
	}

	// starts all of the listeners
	zzk.Start(shutdown, conn, serviceListener, hostRegistry)
}

// SelectHost chooses a host from the pool for the specified service. If the service
// has an address assignment the host will already be selected. If not the host with the least amount
// of memory committed to running containers will be chosen.
func (l *leader) SelectHost(s *service.Service) (*host.Host, error) {
	glog.Infof("Looking for available hosts in pool %s", l.poolID)
	hosts, err := zkservice.GetRegisteredHosts(l.conn, l.shutdown)
	if err != nil {
		glog.Errorf("Could not get available hosts for pool %s: %s", l.poolID, err)
		return nil, err
	}

	// make sure all of the endpoints have address assignments
	var assignment *addressassignment.AddressAssignment
	for i, ep := range s.Endpoints {
		if ep.IsConfigurable() {
			if ep.AddressAssignment.IPAddr != "" {
				assignment = &s.Endpoints[i].AddressAssignment
			} else {
				return nil, fmt.Errorf("missing address assignment")
			}
		}
	}
	if assignment != nil {
		glog.Infof("Found an address assignment for %s (%s) at %s, checking host availability", s.Name, s.ID, assignment.IPAddr)
		var hostID string
		var err error

		// Get the hostID from the address assignment
		if assignment.AssignmentType == commons.VIRTUAL {
			if hostID, err = virtualips.GetHostID(l.conn, assignment.IPAddr); err != nil {
				glog.Errorf("Host not available for virtual ip address %s: %s", assignment.IPAddr, err)
				return nil, err
			}
		} else {
			hostID = assignment.HostID
		}

		// Checking host availability
		for i, host := range hosts {
			if host.ID == hostID {
				return hosts[i], nil
			}
		}

		glog.Errorf("Host %s not available in pool %s.  Check to see if the host is running or reassign ips for service %s (%s)", hostID, l.poolID, s.Name, s.ID)
		return nil, fmt.Errorf("host %s not available in pool %s", hostID, l.poolID)
	}

	hp := s.HostPolicy
	if hp == "" {
		hp = servicedefinition.Balance
	}
	strat, err := strategy.Get(string(hp))
	if err != nil {
		return nil, err
	}

	return StrategySelectHost(s, hosts, strat, l.facade)
}
