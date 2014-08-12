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
	coordclient "github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"

	"path"
)

type leaderFunc func(*facade.Facade, dao.ControlPlane, coordclient.Connection, string, <-chan interface{})

type scheduler struct {
	cpDao        dao.ControlPlane // ControlPlane interface
	cluster_path string           // path to the cluster node
	instance_id  string           // unique id for this node instance
	shutdown     chan interface{} // Shuts down all the pools
	started      bool             // is the loop running
	zkleaderFunc leaderFunc       // multiple implementations of leader function possible
	facade       *facade.Facade
	stopped      chan interface{}

	conn    coordclient.Connection
	leader  coordclient.Leader
	zkEvent <-chan coordclient.Event
}

func NewScheduler(cluster_path string, instance_id string, cpDao dao.ControlPlane, facade *facade.Facade) (*scheduler, error) {

	conn, err := zzk.GetBasePathConnection("/")
	if err != nil {
		glog.Error(err)
		return nil, err
	}

	s := &scheduler{
		cpDao:        cpDao,
		cluster_path: cluster_path,
		instance_id:  instance_id,
		shutdown:     make(chan interface{}),
		stopped:      make(chan interface{}),
		zkleaderFunc: Lead, // random scheduler implementation
		facade:       facade,
		conn:         conn,
	}
	return s, nil
}

func (s *scheduler) Start() {
	if !s.started {
		s.started = true
		go func() {
			defer close(s.stopped)
			zzk.Listen(s.shutdown, make(chan error, 1), s)
		}()
	}
}

// Shut down node
func (s *scheduler) Stop() {
	if !s.started {
		return
	}
	defer func() { s.started = false }()
	close(s.shutdown)
	<-s.stopped
}

func (s *scheduler) GetConnection() coordclient.Connection { return s.conn }

func (s *scheduler) GetPath(nodes ...string) string {
	p := append([]string{"/pools"}, nodes...)
	return path.Join(p...)
}

func (s *scheduler) Ready() (err error) {
	registry.CreateEndpointRegistry(s.conn)

	s.leader = zzk.NewHostLeader(s.conn, s.instance_id, "/scheduler")
	if s.zkEvent, err = s.leader.TakeLead(); err != nil {
		return err
	}

	// synchronize pools, hosts, services, virtualIPs
	synchronizer := NewSynchronizer(s.facade, datastore.Get())
	go synchronizer.SyncLoop(s.shutdown)

	return nil
}

func (s *scheduler) Done() {
	s.leader.ReleaseLead()
}

func (s *scheduler) Spawn(shutdown <-chan interface{}, poolID string) {
	conn, err := zzk.GetBasePathConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		glog.Error(err)
		return
	}

	_shutdown := make(chan interface{})
	done := make(chan interface{})
	defer func() {
		close(_shutdown)
		<-done
	}()

	for {
		go func() {
			defer close(done)
			s.zkleaderFunc(s.facade, s.cpDao, conn, poolID, _shutdown)
		}()

		select {
		case <-done:
			// restart
			done = make(chan interface{})
		case <-s.zkEvent:
			return
		case <-shutdown:
			return
		}
	}
}
