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

func (l *SnapshotListener) GetConnection() client.Connection { return l.conn }

func (l *SnapshotListener) GetPath(nodes ...string) string { return snapshotPath(nodes...) }

func (l *SnapshotListener) Ready() (err error) { return }

func (l *SnapshotListener) Done() { return }

func (l *SnapshotListener) Spawn(shutdown <-chan interface{}, serviceID string) {
	for {
		var snapshot Snapshot
		event, err := l.conn.GetW(l.GetPath(serviceID), &snapshot)
		if err != nil {
			glog.Errorf("Could not get snapshot %s: %s", serviceID, err)
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
			if err := l.conn.Set(l.GetPath(serviceID), &snapshot); err != nil {
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