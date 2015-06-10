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
	"fmt"
	"path"
	"sync"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

// ResourceManager is the top-level manager for service scheduling services on
// all resource pools.
type ResourceManager struct {
	cpclient dao.ControlPlane
	conn     client.Connection
}

// NewResourceManager instantiates a new resource manager
func NewResourceManager(cpclient dao.ControlPlane) *ResourceManager {
	return &ResourceManager{cpclient: cpclient}
}

// Leader initializes the leader node
func (m *ResourceManager) Leader(conn client.Connection, host *host.Host) client.Leader {
	m.conn = conn
	node := &zkservice.HostNode{Host: host}
	return conn.NewLeader("/resource/leader", node)
}

// Run starts the resource manager
func (m *ResourceManager) Run(cancel <-chan interface{}) error {
	glog.Infof("Starting resource manager")
	var wg sync.WaitGroup
	errch := make(chan error)

	// start the resource listener
	wg.Add(1)
	go func() {
		defer wg.Done()
		zzk.Listen(cancel, errch, m.conn, &ResourceListener{cpclient: m.cpclient})
	}()
	if err := <-errch; err != nil {
		glog.Errorf("Could not start resource manager: %s", err)
		return err
	}
	wg.Wait()

	glog.Infof("Resource manager stopped")
	return nil
}

// ResourceListener monitors all resource pools for changes in services
type ResourceListener struct {
	cpclient dao.ControlPlane
	conn     client.Connection
}

// SetConnection sets the coordinator connection.
func (l *ResourceListener) SetConnection(conn client.Connection) {
	l.conn = conn
}

// GetPath returns the path to the node
func (l *ResourceListener) GetPath(nodes ...string) string {
	return path.Join(append([]string{"/pools"}, nodes...)...)
}

// Ready is a passthrough
func (l *ResourceListener) Ready() error { return nil }

// Done is a passthrough
func (l *ResourceListener) Done() {}

// Spawn sets up a local pool connection and runs the resource scheduler.=
func (l *ResourceListener) Spawn(cancel <-chan interface{}, poolID string) {
	select {
	case conn := <-zzk.Connect(l.GetPath(poolID), zzk.GetLocalConnection):
		if conn != nil {
			scheduler := &ResourceScheduler{
				cpclient: l.cpclient,
				conn:     conn,
				poolID:   poolID,
			}
			scheduler.Run(cancel)
		}
	case <-cancel:
	}
}

// ResourceScheduler manages services and hosts for a single resource pool
type ResourceScheduler struct {
	cpclient dao.ControlPlane
	conn     client.Connection
	poolID   string

	registry *zkservice.HostRegistryListener
}

// NewResourceScheduler instantiates a new ResourceScheduler
func NewResourceScheduler(cpclient dao.ControlPlane, poolID string) *ResourceScheduler {
	return &ResourceScheduler{cpclient: cpclient, poolID: poolID}
}

// Leader initializes the leader node
func (m *ResourceScheduler) Leader(conn client.Connection, host *host.Host) client.Leader {
	m.conn = conn
	return nil
}

// Run starts the scheduler
func (m *ResourceScheduler) Run(cancel <-chan interface{}) error {
	glog.Infof("Starting resource scheduler for pool %s", m.poolID)
	// set up the host registry
	if err := zkservice.InitHostRegistry(m.conn); err != nil {
		glog.Errorf("Could not initialize host registry for pool %s: %s", m.poolID, err)
		return err
	}
	m.registry = zkservice.NewHostRegistryListener()
	// set up the service listener
	serviceListener := zkservice.NewServiceListener(m)
	// start the listeners
	zzk.Start(cancel, m.conn, serviceListener, m.registry)
	glog.Infof("Resource scheduler for pool %s stopped", m.poolID)
	return nil
}

// SelectHost selects a host for a service to run
func (m *ResourceScheduler) SelectHost(svc *service.Service) (*host.Host, error) {
	glog.Infof("Looking for available hosts in pool %s", m.poolID)
	hosts, err := m.registry.GetHosts()
	if err != nil {
		glog.Errorf("Could not get available hosts for pool %s: %s", m.poolID, err)
		return nil, err
	}

	if address, err := getAddress(svc); err != nil {
		glog.Errorf("Could not get address for service %s (%s): %s", svc.Name, svc.ID, err)
		return nil, err
	} else if address != nil {
		glog.Infof("Found an address assignment for %s (%s) at %s, checking host availability", svc.Name, svc.ID, address.IPAddr)
		// Get the hostID from the adress assignment
		hostID, err := getHostIDFromAddress(m.conn, address)
		if err != nil {
			glog.Errorf("Host not available at address %s: %s", address.IPAddr, err)
			return nil, err
		}
		// Checking host availability
		for i, host := range hosts {
			if host.ID == hostID {
				return hosts[i], nil
			}
		}
		glog.Errorf("Host %s not available in pool %s. Check to see if the host is running or reassign ips for service %s (%s)", hostID, m.poolID, svc.Name, svc.ID)
		return nil, fmt.Errorf("host %s not available in pool %s", hostID, m.poolID)
	}
	return NewServiceHostPolicy(svc, m.cpclient).SelectHost(hosts)
}

// getAddress returns the address assignment for a service if it exists
func getAddress(svc *service.Service) (address *addressassignment.AddressAssignment, err error) {
	for i, ep := range svc.Endpoints {
		if ep.IsConfigurable() {
			if ep.AddressAssignment.IPAddr != "" {
				address = &svc.Endpoints[i].AddressAssignment
			} else {
				return nil, fmt.Errorf("missing address assignment")
			}
		}
	}
	return
}

// getHostIDFromAddress returns the Host ID given a particular static or
// virtual ip address
func getHostIDFromAddress(conn client.Connection, address *addressassignment.AddressAssignment) (hostID string, err error) {
	if address.AssignmentType == commons.VIRTUAL {
		return virtualips.GetHostID(conn, address.IPAddr)
	}
	return address.HostID, nil
}