// Copyright 2017 The Serviced Authors.
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

package service

import (
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
)

// PoolListener implements zzk.Listener.  The PoolListener will watch
// pool nodes (/pools/poolid) for changes and then sync virtual IP
// assignments.
type PoolListener struct {
	synchronizer VirtualIPSynchronizer
	connection   client.Connection
	logger       *log.Entry
}

// NewPoolListener instantiates a new PoolListener
func NewPoolListener(synchronizer VirtualIPSynchronizer) *PoolListener {
	return &PoolListener{synchronizer: synchronizer}
}

// SetConnection sets the ZooKeeper connection.  It is part of the zzk.Listener
// interface.
func (l *PoolListener) SetConnection(connection client.Connection) {
	l.connection = connection
}

// GetPath returns the path for the pool being watched, /pools/poolid.  It is
// part of the zzk.Listener interface.
func (l *PoolListener) GetPath(nodes ...string) string {
	return Base().Pools().Path()
}

// Ready is part of the zzk.Listener interface.
func (l *PoolListener) Ready() (err error) { return nil }

// Done is part of the zzk.Listener interface.
func (l *PoolListener) Done() {}

// PostProcess is part of the zzk.Listener interface.
func (l *PoolListener) PostProcess(p map[string]struct{}) {}

// Spawn watches a pool and syncs its virtual IPs.
func (l *PoolListener) Spawn(shutdown <-chan interface{}, poolID string) {
	l.logger = plog.WithField("poolid", poolID)

	l.logger.Debug("Spawning pool listener")

	stop := make(chan struct{})
	defer func() { close(stop) }()

	for {
		poolPath := Base().Pools().ID(poolID)
		node := &PoolNode{ResourcePool: &pool.ResourcePool{}}

		var poolEvent, ipsEvent, poolExistsEvent, ipsExistsEvent <-chan client.Event

		poolExists, poolExistsEvent, err := l.connection.ExistsW(poolPath.Path(), stop)
		if poolExists && err == nil {
			poolEvent, err = l.connection.GetW(poolPath.Path(), node, stop)
			if err != nil {
				l.processError(err, "Unable to watch pool")
				return
			}
		} else if err != nil {
			l.processError(err, "Unable to check if pool exists")
			return
		}

		children := []string{}
		if poolExists {
			var ipsExists bool
			ipsExists, ipsExistsEvent, err = l.connection.ExistsW(poolPath.IPs().Path(), stop)
			if ipsExists && err == nil {
				children, ipsEvent, err = l.connection.ChildrenW(poolPath.IPs().Path(), stop)
				if err != nil {
					l.processError(err, "Unable to watch IPs")
					return
				}
			} else if err != nil {
				l.processError(err, "Unable to watch IPs node")
				return
			}

			assignments, err := l.getAssignmentMap(children)
			if err != nil {
				l.processError(err, "Unable to get assignments")
				return
			}

			err = l.synchronizer.Sync(*node.ResourcePool, assignments, shutdown)
			if err != nil {
				l.processError(err, "Unable to sync virtual IPs")
				return
			}

		}

		select {
		case <-ipsEvent:
		case <-poolEvent:
		case <-poolExistsEvent:
		case <-ipsExistsEvent:
		case <-shutdown:
			return
		}

		close(stop)
		stop = make(chan struct{})
	}
}

func (l *PoolListener) getAssignmentMap(hostIPs []string) (map[string]string, error) {
	assignments := make(map[string]string)
	for _, hostIP := range hostIPs {
		host, ip, err := ParseIPID(hostIP)
		if err != nil {
			return nil, err
		}
		assignments[ip] = host
	}
	return assignments, nil
}

func (l *PoolListener) processError(err error, message string) {
	l.logger.WithError(err).Error(message)
	// if an error occured the listener will get spawned again so wait a little bit before exiting
	// in case the problem has been addressed.
	time.Sleep(5 * time.Second)
	return
}
