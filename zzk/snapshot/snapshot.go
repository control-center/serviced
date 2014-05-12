package zzk

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
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

type Handler struct {
	conn client.Connection
	dao  dao.ControlPlane
}

// New starts a new event listener
func New(conn client.Connection, dao dao.ControlPlane) *Handler {
	return &Handler{
		conn: conn,
		dao:  dao,
	}
}

// Listen listens for changes on the event node and processes the snapshot
func (h *Handler) Listen() {

	// Make the path if it doesn't exist
	if exists, err := h.conn.Exists(snapshotPath()); err != nil && err != client.ErrNoNode {
		glog.Errorf("Error checking path %s: %s", snapshotPath(), err)
		return
	} else if !exists {
		if err := h.conn.CreateDir(snapshotPath()); err != nil {
			glog.Errorf("Could not create path %s: %s", snapshotPath(), err)
			return
		}
	}

	// Wait for snapshot events
	for {
		nodes, event, err := h.conn.ChildrenW(snapshotPath())
		if err != nil {
			glog.Errorf("Could not watch snapshots: %s", err)
			return
		}

		for _, serviceID := range nodes {
			// Get the request
			path := snapshotPath(serviceID)
			var snapshot Snapshot
			if err := h.conn.Get(path, &snapshot); err != nil {
				glog.V(1).Infof("Could not get snapshot %s: %s", serviceID, err)
				continue
			}

			// Snapshot action already performed, continue
			if snapshot.done() {
				continue
			}

			// Do snapshot
			glog.V(1).Infof("Taking snapshot for request: %v", snapshot)
			var label string
			err := h.dao.LocalSnapshot(snapshot.ServiceID, &label)
			if err != nil {
				glog.V(1).Infof("Snapshot failed for request: %v", snapshot)
			}

			// Update request
			snapshot.Label = label
			snapshot.Error = err
			if err := h.conn.Set(path, &snapshot); err != nil {
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
func (h *Handler) Send(snapshot *Snapshot) error {
	return h.conn.Create(snapshotPath(snapshot.ServiceID), snapshot)
}

// Recv waits for a snapshot to be complete
func (h *Handler) Recv(snapshot *Snapshot, serviceID string) error {
	for {
		p := snapshotPath(serviceID)
		event, err := h.conn.GetW(p, snapshot)
		if err != nil {
			return err
		}
		if snapshot.done() {
			// Delete the request
			if err := h.conn.Delete(p); err != nil {
				glog.Warningf("Could not delete snapshot request %s: %s", p, err)
			}
			return nil
		}
		// Wait for something to happen
		<-event
	}
}