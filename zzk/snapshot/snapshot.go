// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package snapshot

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
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
func NewSnapshotListener(conn client.Connection, handler SnapshotHandler) *SnapshotListener {
	return &SnapshotListener{conn, handler}
}

// Listen is the listener call for snapshots
func (l *SnapshotListener) Listen(shutdown <-chan interface{}) {
	// Make the path if it doesn't exist
	if exists, err := l.conn.Exists(snapshotPath()); err != nil && err != client.ErrNoNode {
		glog.Errorf("Error checking path %s: %s", snapshotPath(), err)
		return
	} else if !exists {
		if err := l.conn.CreateDir(snapshotPath()); err != nil {
			glog.Errorf("Could not create path %s: %s", snapshotPath(), err)
			return
		}
	}

	// Wait for snapshot events
	for {
		nodes, event, err := l.conn.ChildrenW(snapshotPath())
		if err != nil {
			glog.Errorf("Could not watch snapshots: %s", err)
			return
		}

		for _, serviceID := range nodes {
			// Get the request
			path := snapshotPath(serviceID)
			var snapshot Snapshot
			if err := l.conn.Get(path, &snapshot); err != nil {
				glog.V(1).Infof("Could not get snapshot %s: %s", serviceID, err)
				continue
			}

			// Snapshot action already performed, continue
			if snapshot.done() {
				continue
			}

			// Do snapshot
			glog.V(1).Infof("Taking snapshot for request: %v", snapshot)
			snapshot.Label, err = l.handler.TakeSnapshot(snapshot.ServiceID)
			if err != nil {
				glog.Warning("Snapshot failed for request: ", snapshot)
				snapshot.Err = err.Error()
			}

			// Update request
			if err := l.conn.Set(path, &snapshot); err != nil {
				glog.V(1).Infof("Could not update snapshot request %s: %s", serviceID, err)
				continue
			}

			glog.V(1).Infof("Finished taking snapshot for request: %v", snapshot)
		}
		// Wait for an event that something changed
		select {
		case e := <-event:
			glog.V(2).Info("Receieved snapshot event: ", e)
		case <-shutdown:
			return
		}
	}
}

// Send sends a new snapshot request to the queue
func Send(conn client.Connection, serviceID string) error {
	return conn.Create(snapshotPath(serviceID), &Snapshot{ServiceID: serviceID})
}

// Recv waits for a snapshot to be complete
func Recv(conn client.Connection, serviceID string, snapshot *Snapshot) error {
	node := snapshotPath(serviceID)

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