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
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"

	"path"
)

type leaderFunc func(<-chan interface{}, coordclient.Connection, dao.ControlPlane, string)

type scheduler struct {
	sync.Mutex                    // only one process can stop and start the scheduler at a time
	cpDao        dao.ControlPlane // ControlPlane interface
	cluster_path string           // path to the cluster node
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
func NewScheduler(cluster_path string, instance_id string, cpDao dao.ControlPlane, facade *facade.Facade) (*scheduler, error) {
	s := &scheduler{
		cpDao:        cpDao,
		cluster_path: cluster_path,
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
	restart:
		for {
			connc := connectZK("/")
			select {
			case s.conn = <-connc:
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

	registry.CreateEndpointRegistry(s.conn)

	// synchronize elastic with zookeeper
	go NewSynchronizer(s.facade, datastore.Get()).SyncLoop(_shutdown)
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
	s.started = false
}

// GetConnection implements zzk.Listener
func (s *scheduler) GetConnection() coordclient.Connection { return s.conn }

// GetPath implements zzk.Listener
func (s *scheduler) GetPath(nodes ...string) string {
	return path.Join(append([]string{"/pools"}, nodes...)...)
}

// Ready implements zzk.Listener
func (s *scheduler) Ready() error {
	glog.Infof("Entering lead!")
	return nil
}

// Done implements zzk.Listener
func (s *scheduler) Done() {
	glog.Infof("Exiting lead!")
	return
}

// Spawn implements zzk.Listener
func (s *scheduler) Spawn(shutdown <-chan interface{}, poolID string) {
	// acquire a pool-based connection
	connc := connectZK(zzk.GeneratePoolPath(poolID))
	var conn coordclient.Connection
	select {
	case conn = <-connc:
		if conn == nil {
			return
		}
	case <-shutdown:
		return
	}

	// manage the pool
	s.zkleaderFunc(shutdown, conn, s.cpDao, poolID)
}

// connectZK creates an asynchronous pool-based connection
func connectZK(path string) <-chan coordclient.Connection {
	connc := make(chan coordclient.Connection)

	go func() {
		for {
			c, err := zzk.GetBasePathConnection(path)
			if err != nil {
				glog.Errorf("Could not acquire connection to %s: %s", path, err)
				close(connc)
			} else {
				connc <- c
			}
		}
	}()

	return connc
}
