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
	"github.com/control-center/serviced/coordinator/storage"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	imgreg "github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"

	"path"
)

type leaderFunc func(<-chan interface{}, coordclient.Connection, dao.ControlPlane, *facade.Facade, string, int)

type scheduler struct {
	sync.Mutex                     // only one process can stop and start the scheduler at a time
	cpDao         dao.ControlPlane // ControlPlane interface
	poolID        string           // pool where the master resides
	realm         string           // realm for which the scheduler will run
	instance_id   string           // unique id for this node instance
	shutdown      chan interface{} // Shuts down all the pools
	started       bool             // is the loop running
	zkleaderFunc  leaderFunc       // multiple implementations of leader function possible
	snapshotTTL   int
	facade        *facade.Facade
	stopped       chan interface{}
	registry      *registry.EndpointRegistry
	storageServer *storage.Server
	pushreg       *imgreg.RegistryListener

	conn coordclient.Connection
}

// NewScheduler creates a new scheduler master
func NewScheduler(poolID string, instance_id string, storageServer *storage.Server, cpDao dao.ControlPlane, facade *facade.Facade, pushreg *imgreg.RegistryListener, snapshotTTL int) (*scheduler, error) {
	s := &scheduler{
		cpDao:         cpDao,
		poolID:        poolID,
		instance_id:   instance_id,
		shutdown:      make(chan interface{}),
		stopped:       make(chan interface{}),
		zkleaderFunc:  Lead, // random scheduler implementation
		facade:        facade,
		snapshotTTL:   snapshotTTL,
		storageServer: storageServer,
		pushreg:       pushreg,
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
	leader, err := conn.NewLeader("/scheduler")
	if err != nil {
		glog.Errorf("Could not initialize leader node for scheduler: %s", err)
		return
	}
	leaderDone := make(chan struct{})
	defer close(leaderDone)
	event, err := leader.TakeLead(&zzk.HostLeader{HostID: s.instance_id, Realm: s.realm}, leaderDone)
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

	var wg sync.WaitGroup
	defer wg.Wait()

	_shutdown := make(chan interface{})
	defer close(_shutdown)

	stopped := make(chan struct{}, 2)

	// monitor the resource pool
	monitor := zkservice.MonitorResourcePool(_shutdown, conn, s.poolID)

	// ensure all the services are unlocked
	glog.Infof("Resetting service locks")
	locker := s.facade.DFSLock(datastore.Get())
	locker.Lock("reset service locks")
	if err := s.facade.ResetLocks(datastore.Get()); err != nil {
		glog.Errorf("Could not reset dfs locks: %s", err)
		return
	}
	locker.Unlock()
	glog.Infof("DFS locks are all reset")

	// start the storage server
	wg.Add(1)
	go func() {
		defer glog.Infof("Stopping storage sync")
		defer wg.Done()
		if err := s.storageServer.Run(_shutdown, conn); err != nil {
			glog.Errorf("Could not maintain storage lead: %s", err)
			stopped <- struct{}{}
		}
	}()

	// synchronize locally
	wg.Add(1)
	go func() {
		defer glog.Infof("Stopping local sync")
		defer wg.Done()
		s.localSync(_shutdown, conn)
	}()

	wg.Add(1)
	go func() {
		defer glog.Infof("Stopping pool listeners")
		defer wg.Done()
		zzk.Start(_shutdown, conn, s, s.pushreg)
		stopped <- struct{}{}
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

	// Get a pool-based connection
	var conn coordclient.Connection
	select {
	case conn = <-zzk.Connect(zzk.GeneratePoolPath(poolID), zzk.GetLocalConnection):
		if conn == nil {
			return
		}
	case <-shutdown:
		return
	}

	var cancel chan interface{}
	var done chan struct{}

	doneW := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&doneW)
	for {
		var node zkservice.PoolNode
		event, err := s.conn.GetW(zzk.GeneratePoolPath(poolID), &node, doneW)
		if err != nil && err != coordclient.ErrEmptyNode {
			glog.Errorf("Error while monitoring pool %s: %s", poolID, err)
			return
		}

		// CC-2020: workaround to prevent churn on empty pool node
		if node.ResourcePool != nil && node.Realm == s.realm {
			if done == nil {
				cancel = make(chan interface{})
				done = make(chan struct{})

				go func() {
					defer close(done)
					s.zkleaderFunc(cancel, conn, s.cpDao, s.facade, poolID, s.snapshotTTL)
				}()
			}
		} else {
			if done != nil {
				close(cancel)
				<-done
				done = nil
			}
		}

		select {
		case <-event:
		case <-done:
			return
		case <-shutdown:
			if done != nil {
				close(cancel)
				<-done
			}
			return
		}

		close(doneW)
		doneW = make(chan struct{})
	}
}
