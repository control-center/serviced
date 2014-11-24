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
	"sync"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"

	"path"
)

type leaderFunc func(<-chan interface{}, coordclient.Connection, dao.ControlPlane, string)

type scheduler struct {
	sync.Mutex                    // only one process can stop and start the scheduler at a time
	cpDao        dao.ControlPlane // ControlPlane interface
	poolID       string           // pool where the master resides
	realm        string           // realm for which the scheduler will run
	instance_id  string           // unique id for this node instance
	shutdown     chan interface{} // Shuts down all the pools
	started      bool             // is the loop running
	zkleaderFunc leaderFunc       // multiple implementations of leader function possible
	facade       *facade.Facade
	stopped      chan interface{}
	registry     *registry.EndpointRegistry

	conn coordclient.Connection
}

// NewScheduler creates a new scheduler master
func NewScheduler(poolID string, instance_id string, cpDao dao.ControlPlane, facade *facade.Facade) (*scheduler, error) {
	s := &scheduler{
		cpDao:        cpDao,
		poolID:       poolID,
		instance_id:  instance_id,
		shutdown:     make(chan interface{}),
		stopped:      make(chan interface{}),
		zkleaderFunc: Lead, // random scheduler implementation
		facade:       facade,
	}
	return s, nil
}

// Start starts the scheduler
func (s *scheduler) Start() {
	s.Lock()
	defer s.Unlock()

	if s.started {
		return
	}
	s.started = true

	pool, err := s.facade.GetResourcePool(datastore.Get(), s.poolID)
	if err != nil {
		glog.Errorf("Could not acquire resource pool %s: %s", s.poolID, err)
		return
	}
	s.realm = pool.Realm

	go func() {

		defer func() {
			close(s.stopped)
			s.started = false
		}()

		for {
			var conn coordclient.Connection
			select {
			case conn = <-zzk.Connect("/", zzk.GetLocalConnection):
				if conn != nil {
					s.mainloop(conn)
				}
			case <-s.shutdown:
				return
			}

			select {
			case <-s.shutdown:
				return
			default:
				// restart
			}
		}
	}()
}

// mainloop acquires the leader lock and initializes the listener
func (s *scheduler) mainloop(conn coordclient.Connection) {
	// become the leader
	leader := zzk.NewHostLeader(conn, s.instance_id, s.realm, "/scheduler")
	event, err := leader.TakeLead()
	if err != nil {
		glog.Errorf("Could not become the leader: %s", err)
		return
	}
	defer leader.ReleaseLead()

	s.registry, err = registry.CreateEndpointRegistry(conn)
	if err != nil {
		glog.Errorf("Error initializing endpoint registry: %s", err)
		return
	}

	// did I shut down before I became the leader?
	select {
	case <-s.shutdown:
		return
	default:
	}

	var (
		wg        sync.WaitGroup
		stopped   = make(chan interface{})
		_shutdown = make(chan interface{})
	)
	defer func() {
		close(_shutdown)
		wg.Wait()
	}()

	// monitor the resource pool
	monitor := zkservice.MonitorResourcePool(_shutdown, conn, s.poolID)

	// synchronize with the remote
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.remoteSync(_shutdown, conn)
	}()

	// synchronize locally
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.localSync(_shutdown, conn)
	}()

	wg.Add(1)
	go func() {
		defer wg.Done()
		defer close(stopped)
		zzk.Start(_shutdown, conn, s, zkservice.NewServiceLockListener())
	}()

	// wait for something to happen
	for {
		select {
		case <-event:
			glog.Warningf("Coup d'etat, re-electing...")
			return
		case <-stopped:
			glog.Warningf("Leader died, re-electing...")
			return
		case pool := <-monitor:
			if pool == nil || pool.Realm != s.realm {
				glog.Warningf("Realm changed, re-electing...")
				return
			}
		case <-s.shutdown:
			return
		}
	}
}

// Stop stops all scheduler processes for the master
func (s *scheduler) Stop() {
	s.Lock()
	defer s.Unlock()

	if !s.started {
		return
	}
	close(s.shutdown)
	<-s.stopped
}

// SetConnection implements zzk.Listener
func (s *scheduler) SetConnection(conn coordclient.Connection) { s.conn = conn }

// PostProcess implements zzk.Listener
func (s *scheduler) PostProcess(p map[string]struct{}) {}

// GetPath implements zzk.Listener
func (s *scheduler) GetPath(nodes ...string) string {
	return path.Join(append([]string{"/pools"}, nodes...)...)
}

// Ready implements zzk.Listener
func (s *scheduler) Ready() error {
	glog.Infof("Entering lead for realm %s!", s.realm)
	glog.Infof("Host Master successfully started")
	return nil
}

// Done implements zzk.Listener
func (s *scheduler) Done() {
	glog.Infof("Exiting lead for realm %s!", s.realm)
	glog.Infof("Host Master shutting down")
	return
}

// Spawn implements zzk.Listener
func (s *scheduler) Spawn(shutdown <-chan interface{}, poolID string) {
	// is this pool in my realm?
	monitor := zkservice.MonitorResourcePool(shutdown, s.conn, poolID)

	// wait for my pool to join the realm (or shutdown)
	done := false
	for !done {
		select {
		case pool := <-monitor:
			if pool != nil && pool.Realm == s.realm {
				done = true
			}
		case <-shutdown:
			return
		}
	}

	// acquire a pool-based connection
	var conn coordclient.Connection
	select {
	case conn = <-zzk.Connect(zzk.GeneratePoolPath(poolID), zzk.GetLocalConnection):
		if conn == nil {
			return
		}
	case pool := <-monitor:
		if pool == nil || pool.Realm != s.realm {
			return
		}
	case <-shutdown:
		return
	}

	// manage the pool
	_shutdown := make(chan interface{})
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		s.zkleaderFunc(_shutdown, conn, s.cpDao, poolID)
	}()

	// wait for shutdown or if the pool's realm changes or the pool gets deleted
	select {
	case pool := <-monitor:
		if pool == nil || pool.Realm != s.realm {
			close(_shutdown)
		}
	case <-shutdown:
		close(_shutdown)
	}

	wg.Wait()
}
