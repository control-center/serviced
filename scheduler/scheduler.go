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
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkscheduler "github.com/control-center/serviced/zzk/scheduler"
	"github.com/zenoss/glog"

	"path"
)

type leaderFunc func(<-chan interface{}, coordclient.Connection, dao.ControlPlane, string)

type scheduler struct {
	sync.Mutex                    // only one process can stop and start the scheduler at a time
	cpDao        dao.ControlPlane // ControlPlane interface
	realm        string           // realm for which the scheduler will run
	instance_id  string           // unique id for this node instance
	shutdown     chan interface{} // Shuts down all the pools
	started      bool             // is the loop running
	zkleaderFunc leaderFunc       // multiple implementations of leader function possible
	facade       *facade.Facade
	stopped      chan interface{}

	conn   coordclient.Connection
	leader coordclient.Leader
}

// NewScheduler creates a new scheduler master
func NewScheduler(realm string, instance_id string, cpDao dao.ControlPlane, facade *facade.Facade) (*scheduler, error) {
	s := &scheduler{
		cpDao:        cpDao,
		realm:        realm,
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

	go func() {

		defer func() {
			close(s.stopped)
			s.started = false
		}()

	restart:
		for {
			select {
			case s.conn = <-zzk.Connect("/", zzk.GetLocalConnection):
				if s.conn == nil {
					// wait a second and try again
					<-time.After(time.Second)
					continue restart
				}
			case <-s.shutdown:
				return
			}

			s.mainloop()

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
func (s *scheduler) mainloop() {
	// become the leader
	leader := zzk.NewHostLeader(s.conn, s.instance_id, "/scheduler")
	event, err := leader.TakeLead()
	if err != nil {
		glog.Errorf("Could not become the leader: %s", err)
		return
	}
	defer leader.ReleaseLead()

	// did I shut down before I became the leader?
	select {
	case <-s.shutdown:
		return
	default:
	}

	var (
		_shutdown = make(chan interface{})
		stopped   = make(chan interface{})
	)

	er, err := registry.CreateEndpointRegistry(s.conn)
	if err != nil {
		glog.Errorf("Could not initialize endpoint registry: %s", err)
		return
	}

	// remote synchronization
	go doRemoteSync(_shutdown, s.facade, er)

	// local synchronization
	go doLocalSync(_shutdown, s.facade)

	go func() {
		defer close(stopped)
		zzk.Listen(_shutdown, make(chan error, 1), s)
	}()

	// wait for something to happen
	select {
	case <-event:
		glog.Warningf("Coup d'etat, re-electing...")
	case <-stopped:
		glog.Warningf("Leader died, re-electing...")
	case <-s.shutdown:
	}

	close(_shutdown)
	<-stopped
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

// GetConnection implements zzk.Listener
func (s *scheduler) GetConnection() coordclient.Connection { return s.conn }

// GetPath implements zzk.Listener
func (s *scheduler) GetPath(nodes ...string) string {
	return path.Join(append([]string{"/pools"}, nodes...)...)
}

// Ready implements zzk.Listener
func (s *scheduler) Ready() error {
	glog.Infof("Entering lead for realm %s!", s.realm)
	return nil
}

// Done implements zzk.Listener
func (s *scheduler) Done() {
	glog.Infof("Exiting lead for realm %s!", s.realm)
	return
}

// PostProcess implements zzk.Listener
func (s *scheduler) PostProcess(processing map[string]struct{}) {}

// Spawn implements zzk.Listener
func (s *scheduler) Spawn(shutdown <-chan interface{}, poolID string) {
	// is this pool in my realm?
	monitor := zkscheduler.MonitorResourcePool(shutdown, s.conn, poolID)

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
	go func() {
		wg.Add(1)
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
