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

package snapshot

import (
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

const (
	zkSnapshot = "/snapshots"
)

func snapshotPath(nodes ...string) string {
	p := []string{zkSnapshot}
	p = append(p, nodes...)
	return path.Join(p...)
}

// Snapshot is the snapshot request object
type Snapshot struct {
	ServiceID string
	Label     string
	Err       string
	version   interface{}
}

// Version implements client.Node
func (s *Snapshot) Version() interface{} { return s.version }

// SetVersion implements client.Node
func (s *Snapshot) SetVersion(version interface{}) { s.version = version }

func (s *Snapshot) done() bool { return s.Label != "" || s.Err != "" }

// SnapshotHandler is the handler interface for running a snapshot listener
type SnapshotHandler interface {
	TakeSnapshot(serviceID string) (label string, err error)
}

// SnapshotListener is the zk listener for snapshots
type SnapshotListener struct {
	conn    client.Connection
	handler SnapshotHandler
}

// NewSnapshotListener instantiates a new listener for snapshots
func NewSnapshotListener(handler SnapshotHandler) *SnapshotListener {
	return &SnapshotListener{handler: handler}
}

// SetConnection implements zzk.Listener
func (l *SnapshotListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *SnapshotListener) GetPath(nodes ...string) string { return snapshotPath(nodes...) }

// Ready implements zzk.Listener
func (l *SnapshotListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *SnapshotListener) Done() { return }

// Spawn takes a snapshot of a service and waits for the node to be deleted.  If
// the node is not removed, then no action is performed.
func (l *SnapshotListener) Spawn(shutdown <-chan interface{}, nodeID string) {
	for {
		var snapshot Snapshot
		event, err := l.conn.GetW(l.GetPath(nodeID), &snapshot)
		if err != nil {
			glog.Errorf("Could not get snapshot %s: %s", nodeID, err)
			return
		}

		if !snapshot.done() {
			glog.V(1).Infof("Taking snapshot for service: %s", snapshot.ServiceID)
			snapshot.Label, err = l.handler.TakeSnapshot(snapshot.ServiceID)
			if err != nil {
				glog.Warningf("Snapshot failed for service: %s", snapshot.ServiceID)
				snapshot.Err = err.Error()
			}

			// Update request
			if err := l.conn.Set(l.GetPath(nodeID), &snapshot); err != nil {
				glog.Errorf("Could not update snapshot for service %s: %s", snapshot.ServiceID, err)
			}
			glog.V(1).Infof("Finished taking snapshot for service %s", snapshot.ServiceID)
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				return
			}
			glog.V(4).Infof("snapshot %s received event: %v", snapshot.ServiceID, e)
		case <-shutdown:
			return
		}
	}
}

// Send sends a new snapshot request to the queue
func Send(conn client.Connection, serviceID string) (string, error) {
	var snapshot Snapshot
	snapshot.ServiceID = serviceID

	p, err := conn.CreateEphemeral(snapshotPath(serviceID), &snapshot)
	if err != nil {
		return "", err
	}

	snapshotID := path.Base(p)
	if err := conn.Set(snapshotPath(snapshotID), &snapshot); err != nil {
		return "", err
	}

	return snapshotID, nil
}

// Recv waits for a snapshot to be complete
func Recv(conn client.Connection, nodeID string, snapshot *Snapshot) error {
	node := snapshotPath(nodeID)

	for {
		event, err := conn.GetW(node, snapshot)
		if err != nil {
			return err
		}
		if snapshot.done() {
			// Delete the request
			if err := conn.Delete(node); err != nil {
				glog.Warningf("Could not delete snapshot request %s: %s", node, err)
			}
			return nil
		}
		// Wait for something to happen
		<-event
	}
}
