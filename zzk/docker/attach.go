package docker

import (
	"os"
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
)

// TODO: change this to accept a random port on the host to stream data

func attachPath(nodes ...string) string {
	p := []string{zkDocker, zkAttach}
	p = append(p, nodes...)
	return path.Join(p...)
}

type Attach struct {
	HostID   string
	DockerID string
	Command  []string
	Started  bool
	Output   []byte
	Error    error
	version  interface{}
}

func (a *Attach) Version() interface{}           { return a.version }
func (a *Attach) SetVersion(version interface{}) { a.version = version }

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

		for _, id := range nodes {
			// Get the request
			path := attachPath(hostID, id)
			var cmd Attach
			if err := conn.Get(path, &cmd); err != nil {
				glog.V(1).Infof("Could not get command at %s: %s", path, err)
				continue
			}

			// Attach already performed, continue
			if cmd.Started {
				continue
			}

			// Do attach
			glog.V(1).Infof("Attaching to service state via request: %v", cmd)
			cmd.Started = true
			if err := conn.Set(path, &cmd); err != nil {
				glog.V(1).Infof("Could not update command at %s", path, err)
				continue
			}
			go func() {
				defer glog.V(1).Infof("Finished attaching to command: %v", cmd)
				exec, err := attach(cmd.DockerID, cmd.Command)
				if err != nil {
					cmd.Error = err
				} else {
					cmd.Output, cmd.Error = exec.CombinedOutput()
				}
				if err := conn.Set(path, &cmd); err != nil {
					glog.V(1).Infof("Could not update command at %s", path, err)
				}
			}()
		}
		// Wait for an event that something changed
		<-event
	}
}

func SendAttach(conn client.Connection, cmd *Attach) (string, error) {
	node := attachPath(cmd.HostID, newuuid())
	if err := mkdir(conn, path.Dir(node)); err != nil {
		return "", err
	}
	if err := conn.Create(node, cmd); err != nil {
		return "", err
	}
	return path.Base(node), nil
}

func RecvAttach(conn client.Connection, hostID string, id string) (*Attach, error) {
	var cmd Attach
	node := attachPath(hostID, id)

	for {
		event, err := conn.GetW(node, &cmd)
		if err != nil {
			return nil, err
		}
		if cmd.Started {
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

func LocalAttach(cmd *Attach) error {
	exec, err := attach(cmd.DockerID, cmd.Command)
	if err != nil {
		return err
	}

	exec.Stdin = os.Stdin
	exec.Stdout = os.Stdout
	exec.Stderr = os.Stderr
	return exec.Run()
}