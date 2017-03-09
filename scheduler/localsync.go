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
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

const (
	minWait    = 30 * time.Second
	maxWait    = 3 * time.Hour
	lockBlock  = time.Second
	noLockWait = 5 * time.Minute
)

func (s *scheduler) localSync(shutdown <-chan interface{}, rootConn client.Connection) {
	wait := time.After(0)
	for {
		select {
		case <-wait:
		case <-shutdown:
			return
		}
		wait = s.doSync(rootConn)
	}
}

func (s *scheduler) doSync(rootConn client.Connection) <-chan time.Time {
	ctx := datastore.Get()
	defer ctx.Metrics().Stop(ctx.Metrics().Start("scheduler.doSync"))
	// SyncRegistryImages performs its own DFSLock, so run it before locking in here
	if err := s.facade.SyncRegistryImages(ctx, false); err != nil {
		glog.Errorf("%s", err)
		return time.After(minWait)
	}

	if err := s.facade.DFSLock(ctx).LockWithTimeout("zookeeper sync", lockBlock); err != nil {
		glog.Infof("Could not lock DFS (%s), will retry later", err)
		return time.After(noLockWait)
	}
	defer s.facade.DFSLock(ctx).Unlock()

	pools, err := s.GetResourcePools()
	if err != nil {
		glog.Errorf("Could not get resource pools: %s", err)
		return time.After(minWait)
	} else if err := zkservice.SyncResourcePools(rootConn, pools); err != nil {
		glog.Errorf("Could not do a local sync of resource pools: %s", err)
		return time.After(minWait)
	}

	for _, pool := range pools {
		conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(pool.ID))
		if err != nil {
			glog.Errorf("Could not get a pool-based connection for %s to zookeeper: %s", pool.ID, err)
			return time.After(minWait)
		}

		// Update the hosts
		if hosts, err := s.GetHostsByPool(pool.ID); err != nil {
			glog.Errorf("Could not get hosts in pool %s: %s", pool.ID, err)
			return time.After(minWait)
		} else if err := zkservice.SyncHosts(conn, hosts); err != nil {
			glog.Errorf("Could not do a local sync of hosts: %s", err)
			return time.After(minWait)
		}

		// Update the services
		if svcs, err := s.GetServicesByPool(pool.ID); err != nil {
			glog.Errorf("Could not get services: %s", err)
			return time.After(minWait)
		} else if err = zkservice.SyncServices(conn, svcs); err != nil {
			glog.Errorf("Could not do a local sync of services: %s", err)
			return time.After(minWait)
		} else {
			for _, svc := range svcs {
				if err := s.facade.SyncServiceRegistry(ctx, &svc); err != nil {
					glog.Errorf("Could not sync public endpoints for service %s (%s): %s", svc.Name, svc.ID, err)
					return time.After(minWait)
				}
			}
		}
	}

	return time.After(maxWait)
}
