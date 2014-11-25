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

	"github.com/control-center/serviced/commons"
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/snapshot"
	"github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

type leader struct {
	conn         coordclient.Connection
	dao          dao.ControlPlane
	hostRegistry *zkservice.HostRegistryListener
	poolID       string
}

// Lead is executed by the "leader" of the control center cluster to handle its management responsibilities of:
//    services
//    snapshots
//    virtual IPs
func Lead(shutdown <-chan interface{}, conn coordclient.Connection, dao dao.ControlPlane, poolID string) {
	// creates a listener for the host registry
	if err := zkservice.InitHostRegistry(conn); err != nil {
		glog.Errorf("Could not initialize host registry for pool %s: %s", err)
		return
	}
	hostRegistry := zkservice.NewHostRegistryListener()
	leader := leader{conn, dao, hostRegistry, poolID}
	glog.V(0).Info("Processing leader duties")

	// creates a listener for snapshots with a function call to take snapshots
	// and return the label and error message
	snapshotListener := snapshot.NewSnapshotListener(&leader)

	// creates a listener for services
	serviceListener := zkservice.NewServiceListener(&leader)

	// starts all of the listeners
	zzk.Start(shutdown, conn, serviceListener, hostRegistry, snapshotListener)
}

func (l *leader) TakeSnapshot(serviceID string) (string, error) {
	var label string
	err := l.dao.Snapshot(serviceID, &label)
	return label, err
}

// SelectHost chooses a host from the pool for the specified service. If the service
// has an address assignment the host will already be selected. If not the host with the least amount
// of memory committed to running containers will be chosen.
func (l *leader) SelectHost(s *service.Service) (*host.Host, error) {
	var assignmentType string
	var ipAddr string
	var hostid string

	glog.Infof("Looking for available hosts in pool %s", l.poolID)
	hosts, err := l.hostRegistry.GetHosts()
	if err != nil {
		glog.Errorf("Could not get available hosts for pool %s: %s", l.poolID, err)
		return nil, err
	}

	for _, ep := range s.Endpoints {
		if ep.AddressAssignment != (addressassignment.AddressAssignment{}) {
			assignmentType = ep.AddressAssignment.AssignmentType
			ipAddr = ep.AddressAssignment.IPAddr
			hostid = ep.AddressAssignment.HostID
			break
		}
	}

	if assignmentType == commons.VIRTUAL {
		// populate hostid
		var err error
		hostid, err = virtualips.GetHostID(l.conn, ipAddr)
		if err != nil {
			return nil, err
		}
		glog.Infof("Service: %v has been assigned virtual IP: %v which has been locked and configured on host %s", s.Name, ipAddr, hostid)
	}

	if hostid != "" {
		return poolHostFromAddressAssignments(hostid, hosts)
	}

	return NewServiceHostPolicy(s, l.dao).SelectHost(hosts)
}

// poolHostFromAddressAssignments determines the pool host for the service from its address assignment(s).
func poolHostFromAddressAssignments(hostid string, hosts []*host.Host) (*host.Host, error) {
	// ensure the assigned host is in the pool
	for _, h := range hosts {
		if h.ID == hostid {
			return h, nil
		}
	}

	return nil, fmt.Errorf("assigned host is not in pool")
}
