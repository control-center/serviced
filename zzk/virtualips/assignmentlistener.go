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

package virtualips

import (
	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/logging"
)

var (
	plog = logging.PackageLogger()
)

// AssignmentListener implements zzk.Listener.  The AssignmentListener will watch for
// virtual IP nodes (/pools/poolid/virtualIPs) and assign or unassign them.
type AssignmentListener struct {
	conn    client.Connection
	logger  *log.Entry
	poolID  string
	handler AssignmentHandler
}

// NewAssignmentListener instantiates a new ServiceListener
func NewAssignmentListener(poolID string, handler AssignmentHandler) *AssignmentListener {
	return &AssignmentListener{
		logger:  plog.WithField("poolid", poolID),
		poolID:  poolID,
		handler: handler}
}

// SetConnection sets the ZooKeeper connection.  It is part of the zzk.Listener interface.
func (l *AssignmentListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath returns the path /pools/poolid/virtualIPs.  It is part of the zzk.Listener interface.
func (l *AssignmentListener) GetPath(nodes ...string) string {
	return Base().Pools().ID(l.poolID).VirtualIPs().Path()
}

// Ready is part of the zzk.Listener interface.
func (l *AssignmentListener) Ready() (err error) { return nil }

// Done is part of the zzk.Listener interface.
func (l *AssignmentListener) Done() {}

// PostProcess is part of the zzk.Listener interface.
func (l *AssignmentListener) PostProcess(p map[string]struct{}) {}

// Spawn watches a virtual IP and assigns or unassigns it to a host.
func (l *AssignmentListener) Spawn(shutdown <-chan interface{}, virtualIP string) {
	logger := l.logger.WithField("ipAddress", virtualIP)

	logger.Debug("Spawning virtual IP listener")

	stop := make(chan struct{})
	defer func() { close(stop) }()

	for {
		vipNode := &VirtualIPNode{}
		p := Base().Pools().ID(l.poolID).VirtualIPs().ID(virtualIP).Path()
		virtualIPEvent, err := l.conn.GetW(p, vipNode, stop)
		if err != nil {
			logger.WithError(err).Error("Could not look up virtual IP")
			return
		}

		logger.Debug("Assigning virtual IP")

		err = l.handler.Assign(l.poolID, vipNode.IP, vipNode.Netmask, vipNode.BindInterface, shutdown)
		if err != nil && err != ErrAlreadyAssigned {
			logger.WithError(err).Error("Error assigning virtual IP")
			return
		}

		logger.Debug("Watching virtual IP assignment")

		assignmentEvent, err := l.handler.Watch(l.poolID, virtualIP, stop)
		if err != nil {
			logger.WithError(err).Error("Error watching assignment")
			return
		}

		select {
		case <-virtualIPEvent:
			logger.Debug("Unassigning virtual IP")

			err := l.handler.Unassign(l.poolID, virtualIP)
			if err != nil && err != ErrNoAssignedHost {
				logger.WithError(err).Error("Could not unassign virtual IP")
			}
			return
		case <-assignmentEvent:
			logger.Debug("Assignment event")
		case <-shutdown:
			logger.Debug("Shutdown event")
			return
		}

		close(stop)
		stop = make(chan struct{})
	}
}
