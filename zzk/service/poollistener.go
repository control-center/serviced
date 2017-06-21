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

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/zzk"
)

// PoolListener implements zzk.Listener.  The PoolListener will watch
// pool nodes (/pools/poolid) for changes and then sync virtual IP
// assignments.
type PoolListener struct {
	Timeout      time.Duration
	synchronizer VirtualIPSynchronizer
	connection   client.Connection
}

// NewPoolListener instantiates a new PoolListener
func NewPoolListener(synchronizer VirtualIPSynchronizer) *PoolListener {
	return &PoolListener{synchronizer: synchronizer, Timeout: time.Second * 5}
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
	logger := plog.WithField("poolid", poolID)

	logger.Debug("Spawning pool listener")

	stop := make(chan struct{})
	defer func() { close(stop) }()

	timeout := time.NewTimer(1 * time.Second)
	timeout.Stop()

	for {
		poolPath := Base().Pools().ID(poolID)
		node := &PoolNode{ResourcePool: &pool.ResourcePool{}}

		var poolEvent, ipsEvent, poolExistsEvent, ipsExistsEvent <-chan client.Event

		poolExists, poolExistsEvent, err := l.connection.ExistsW(poolPath.Path(), stop)
		if poolExists && err == nil {
			poolEvent, err = l.connection.GetW(poolPath.Path(), node, stop)
			if err == client.ErrNoNode {
				close(stop)
				stop = make(chan struct{})
				continue
			} else if err != nil {
				logger.WithError(err).Error("Unable to watch pool")
				return
			}
		} else if err != nil {
			logger.WithError(err).Error("Unable to check if pool exists")
			return
		}

		children := []string{}
		if poolExists {
			var ipsExists bool
			ipsExists, ipsExistsEvent, err = l.connection.ExistsW(poolPath.IPs().Path(), stop)
			if ipsExists && err == nil {
				children, ipsEvent, err = l.connection.ChildrenW(poolPath.IPs().Path(), stop)
				if err != nil {
					logger.WithError(err).Error("Unable to watch IPs")
					return
				}
			} else if err != nil {
				logger.WithError(err).Error("Unable to watch IPs node")
				return
			}

			assignments, err := l.getAssignmentMap(children)
			if err != nil {
				logger.WithError(err).Error("Unable to get assignments")
				return
			}

			// The sync will add nodes to the ips path which will trigger an ipsEvent
			// causing the loop to occur twice.
			err = l.synchronizer.Sync(*node.ResourcePool, assignments)
			if syncError, ok := err.(SyncError); ok {
				logger.WithError(syncError).WithField("count", len(syncError)).
					Warn("Errors encountered while syncing virtual IPs")

				for _, e := range syncError {
					logger.WithError(e).Debug("Sync error")
				}

				timeout.Reset(l.Timeout)
			} else if err != nil {
				logger.WithError(err).Warn("Error Syncing")
				timeout.Reset(l.Timeout)
			}
		}

		select {
		case <-ipsEvent:
		case <-poolEvent:
		case <-poolExistsEvent:
		case <-ipsExistsEvent:
		case <-timeout.C:
		case <-shutdown:
			return
		}

		if !timeout.Stop() {
			select {
			case <-timeout.C:
			default:
			}
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

func StartPoolListener(shutdown <-chan interface{}, connection client.Connection) {
	assignmentHandler := NewZKAssignmentHandler(
		&RandomHostSelectionStrategy{},
		NewRegisteredHostHandler(connection),
		connection)

	synchronizer := NewZKVirtualIPSynchronizer(assignmentHandler)
	poolListener := NewPoolListener(synchronizer)

	zzk.Start(shutdown, connection, poolListener)
}
