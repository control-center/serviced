// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package scheduler

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

func (s *scheduler) startRemote(cancel <-chan struct{}, remote, local client.Connection) <-chan interface{} {
	var (
		shutdown = make(chan interface{})
		done     = make(chan interface{})
	)

	// wait to receieve a cancel channel or a done channel and shutdown
	go func() {
		defer close(shutdown)
		select {
		case <-cancel:
		case <-done:
		}
	}()

	// start the listeners and wait for shutdown or for something to break
	go func() {
		defer close(done)
		glog.Infof("Remote connection established; synchronizing")
		zzk.Start(shutdown, remote, nil, s.getPoolSynchronizer(), s.getEndpointSynchronizer(local))
		glog.Warningf("Running in disconnected mode")
	}()

	// indicate when the listeners a finished
	return done
}

func (s *scheduler) monitorRemote(shutdown <-chan interface{}, remote, local client.Connection) {
	var done <-chan interface{}
	_shutdown := make(chan interface{})
	cancel := make(chan struct{})

	defer func() {
		close(_shutdown)
		close(cancel)
		select {
		case <-done:
		}
	}()

	// monitor the leader's realm
	ch := zzk.MonitorRealm(_shutdown, remote, "/scheduler")
	for {
		select {
		case realm := <-ch:
			switch realm {
			case "":
				// empty realm means something bad happened; exit
				return
			case s.realm:
				// remote realm is the same as local realm; disconnect
				// from the master until this changes
				if done != nil {
					cancel <- struct{}{}
					<-done
					done = nil
				}
			default:
				// start remote synchonization if not yet started
				if done == nil {
					done = s.startRemote(cancel, remote, local)
				}
			}
		case <-done:
			// synchronization failed; shutdown
			return
		case <-shutdown:
			// receieved signal to shutdown
			return
		}
	}
}

func (s *scheduler) remoteSync(shutdown <-chan interface{}, local client.Connection) {
	for {
		select {
		case remote := <-zzk.Connect("/", zzk.GetRemoteConnection):
			if remote != nil {
				s.monitorRemote(shutdown, remote, local)
			}
		case <-shutdown:
			return
		}

		select {
		case <-shutdown:
			return
		default:
			// probably lost connection to the remote; try again
		}
	}
}

func (s *scheduler) getPoolSynchronizer() zzk.Listener {
	poolSync := zkservice.NewPoolSynchronizer(s, zzk.GetRemoteConnection)

	// Add the host listener
	poolSync.AddListener(func(id string) zzk.Listener {
		return zkservice.NewHostSynchronizer(s, id)
	})

	// Add the service listener
	poolSync.AddListener(func(id string) zzk.Listener {
		return zkservice.NewServiceSynchronizer(s, id)
	})

	return poolSync
}

func (s *scheduler) getEndpointSynchronizer(local client.Connection) zzk.Listener {
	return registry.NewEndpointSynchronizer(local, s.registry, zzk.GetRemoteConnection)
}

func (s *scheduler) GetResourcePools() ([]pool.ResourcePool, error) {
	return s.facade.GetResourcePools(datastore.Get())
}

func (s *scheduler) AddUpdateResourcePool(pool *pool.ResourcePool) error {
	if p, err := s.facade.GetResourcePool(datastore.Get(), pool.ID); err != nil {
		return err
	} else if p == nil {
		return s.facade.AddResourcePool(datastore.Get(), pool)
	}

	return s.facade.UpdateResourcePool(datastore.Get(), pool)
}

func (s *scheduler) RemoveResourcePool(id string) error {
	return s.facade.RemoveResourcePool(datastore.Get(), id)
}

func (s *scheduler) GetServicesByPool(id string) ([]service.Service, error) {
	return s.facade.GetServicesByPool(datastore.Get(), id)
}

func (s *scheduler) AddUpdateService(svc *service.Service) error {
	if sv, err := s.facade.GetService(datastore.Get(), svc.ID); err != nil {
		return err
	} else if sv == nil {
		return s.facade.AddService(datastore.Get(), *svc, false)
	}

	return s.facade.UpdateService(datastore.Get(), *svc)
}

func (s *scheduler) RemoveService(id string) error {
	return s.facade.RemoveService(datastore.Get(), id)
}

func (s *scheduler) GetHostsByPool(id string) ([]host.Host, error) {
	return s.facade.FindHostsInPool(datastore.Get(), id)
}

func (s *scheduler) AddUpdateHost(host *host.Host) error {
	if h, err := s.facade.GetHost(datastore.Get(), host.ID); err != nil {
		return err
	} else if h == nil {
		return s.facade.AddHost(datastore.Get(), host)
	}

	return s.facade.UpdateHost(datastore.Get(), host)
}

func (s *scheduler) RemoveHost(id string) error {
	return s.facade.RemoveHost(datastore.Get(), id)
}
