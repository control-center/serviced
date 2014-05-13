package docker

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/domain/servicestate"
)

const (
	zkAttach = "/docker/attach"
)

func attachPath(nodes ...string) string {
	p := []string{zkAttach}
	p = append(p, nodes...)
	return path.Join(p...)
}

func mkdir(conn client.Connection, dir string) error {
	if exists, err := conn.Exists(dir); err != nil && err != client.ErrNoNode {
		return err
	} else if exists {
		return nil
	} else if err := mkdir(conn, path.Dir(dir)); err != nil {
		return err
	}
	return conn.CreateDir(dir)
}

type Attach struct {
	ServiceState servicestate.ServiceState
	Command      []string
	Output       []byte
	Error        error
	version      interface{}
}

func (a *Attach) Version() interface{}           { return a.version }
func (a *Attach) SetVersion(version interface{}) { a.version = version }
func (a *Attach) done() bool                     { return len(a.Output) > 0 || a.Error != nil }

func ListenAttach(conn client.Connection, hostID string) {

	// Make the path if it doesn't exist
	node := attachPath(hostID)
	if err := mkdir(conn, node); err != nil {
		glog.Errorf("Could not create path %s: %s", node, err)
		return
	}

	// Wait for attach commands
	for {
		nodes, event, err := conn.ChildrenW(node)
		if err != nil {
			glog.Errorf("Could not listen for commands %s: %s", node, err)
			return
		}

		for _, ssID := range nodes {
			// Get the request
			path := attachPath(hostID, ssID)
			var cmd Attach
			if err := conn.Get(path, &cmd); err != nil {
				glog.V(1).Infof("Could not get command at %s: %s", path, err)
				continue
			}

			// Attach already performed, continue
			if cmd.done() {
				continue
			}

			// Do attach
			glog.V(1).Infof("Attaching to service state via request: %v", cmd)
			exec := cmd.ServiceState.Attach(cmd.Command...)
			cmd.Output, cmd.Error = exec.CombinedOutput()
			if err := conn.Set(path, &cmd); err != nil {
				glog.V(1).Infof("Could not update command at %s", path, err)
				continue
			}

			glog.V(1).Infof("Finished attaching command: %v", cmd)
		}
		// Wait for an event that something changed
		<-event
	}
}

func SendAttach(conn client.Connection, cmd *Attach) error {
	node := attachPath(cmd.ServiceState.HostId, cmd.ServiceState.Id)
	if err := mkdir(conn, path.Dir(node)); err != nil {
		return err
	}
	return conn.Create(node, cmd)
}

func RecvAttach(conn client.Connection, hostID, ssID string) (*Attach, error) {
	var cmd Attach
	node := attachPath(hostID, ssID)

	for {
		event, err := conn.GetW(node, &cmd)
		if err != nil {
			return nil, err
		}
		if cmd.done() {
			// Delete the request
			if err := conn.Delete(node); err != nil {
				glog.Warningf("Could not delete command request %s: %s", node, err)
			}
			return &cmd, nil
		}
		// Wait for something to happen
		<-event
	}
}