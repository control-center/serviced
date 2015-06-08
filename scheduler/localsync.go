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
	zkservice "github.com/control-center/serviced/zzk/service"
	zkvirtualips "github.com/control-center/serviced/zzk/virtualips"
	"github.com/zenoss/glog"
)

const (
	minWait = 30 * time.Second
	maxWait = 3 * time.Hour
)

func (s *scheduler) localSync(shutdown <-chan interface{}, rootConn client.Connection) {
	wait := time.After(0)

retry:
	for {
		select {
		case <-wait:
		case <-shutdown:
			return
		}

		pools, err := s.GetResourcePools()
		if err != nil {
			glog.Errorf("Could not get resource pools: %s", err)
			wait = time.After(minWait)
			continue
		} else if err := zkservice.SyncPools(rootConn, pools); err != nil {
			glog.Errorf("Could not do a local sync of resource pools: %s", err)
			wait = time.After(minWait)
			continue
		}

		for _, pool := range pools {
			// Update the hosts
			if hosts, err := s.GetHostsByPool(pool.ID); err != nil {
				glog.Errorf("Could not get hosts in pool %s: %s", pool.ID, err)
				wait = time.After(minWait)
				continue retry
			} else if err := zkservice.SyncHosts(rootConn, pool.ID, hosts); err != nil {
				glog.Errorf("Could not do a local sync of hosts: %s", err)
				wait = time.After(minWait)
				continue retry
			}

			// Update the services
			if svcs, err := s.GetServicesByPool(pool.ID); err != nil {
				glog.Errorf("Could not get services: %s", err)
				wait = time.After(minWait)
				continue retry
			} else if zkservice.SyncServices(rootConn, pool.ID, svcs); err != nil {
				glog.Errorf("Could not do a local sync of services: %s", err)
				wait = time.After(minWait)
				continue retry
			}

			// Update Virtual IPs
			if err := zkvirtualips.SyncVirtualIPs(rootConn, pool.ID, pool.VirtualIPs); err != nil {
				glog.Errorf("Could not sync virtual ips for %s: %s", pool.ID, err)
				wait = time.After(minWait)
				continue retry
			}
		}

		wait = time.After(maxWait)
	}
}
