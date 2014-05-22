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
	Error     error
	version   interface{}
}

// Version implements client.Node
func (s *Snapshot) Version() interface{} { return s.version }

// SetVersion implements client.Node
func (s *Snapshot) SetVersion(version interface{}) { s.version = version }

func (s *Snapshot) done() bool { return s.Label != "" || s.Error != nil }

// TakeSnapshot is the function call for taking the snapshot
type TakeSnapshot func(serviceID string) (label string, err error)

// Listen listens for changes on the event node and processes the snapshot
func Listen(conn client.Connection, ts TakeSnapshot) {
	// Make the path if it doesn't exist
	if exists, err := conn.Exists(snapshotPath()); err != nil && err != client.ErrNoNode {
		glog.Errorf("Error checking path %s: %s", snapshotPath(), err)
		return
	} else if !exists {
		if err := conn.CreateDir(snapshotPath()); err != nil {
			glog.Errorf("Could not create path %s: %s", snapshotPath(), err)
			return
		}
	}

	// Wait for snapshot events
	for {
		nodes, event, err := conn.ChildrenW(snapshotPath())
		if err != nil {
			glog.Errorf("Could not watch snapshots: %s", err)
			return
		}

		for _, serviceID := range nodes {
			// Get the request
			path := snapshotPath(serviceID)
			var snapshot Snapshot
			if err := conn.Get(path, &snapshot); err != nil {
				glog.V(1).Infof("Could not get snapshot %s: %s", serviceID, err)
				continue
			}

			// Snapshot action already performed, continue
			if snapshot.done() {
				continue
			}

			// Do snapshot
			glog.V(1).Infof("Taking snapshot for request: %v", snapshot)
			snapshot.Label, snapshot.Error = ts(snapshot.ServiceID)
			if snapshot.Error != nil {
				glog.V(1).Infof("Snapshot failed for request: %v", snapshot)
			}
			// Update request
			if err := conn.Set(path, &snapshot); err != nil {
				glog.V(1).Infof("Could not update snapshot request %s: %s", serviceID, err)
				continue
			}

			glog.V(1).Infof("Finished taking snapshot for request: %v", snapshot)
		}
		// Wait for an event that something changed
		<-event
	}
}

// Send sends a new snapshot request to the queue
func Send(conn client.Connection, snapshot *Snapshot) error {
	return conn.Create(snapshotPath(snapshot.ServiceID), snapshot)
}

// Recv waits for a snapshot to be complete
func Recv(conn client.Connection, serviceID string) (Snapshot, error) {
	var snapshot Snapshot
	node := snapshotPath(serviceID)

	for {
		event, err := conn.GetW(node, &snapshot)
		if err != nil {
			return snapshot, err
		}
		if snapshot.done() {
			// Delete the request
			if err := conn.Delete(node); err != nil {
				glog.Warningf("Could not delete snapshot request %s: %s", node, err)
			}
			return snapshot, nil
		}
		// Wait for something to happen
		<-event
	}
}