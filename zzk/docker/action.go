package docker

import (
	"path"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/utils"
)

const (
	zkAction = "/docker/action"
)

func actionPath(nodes ...string) string {
	p := []string{zkAction}
	p = append(p, nodes...)
	return path.Join(p...)
}

// Action is the request node for initialized a serviced action on a host
type Action struct {
	HostID   string
	DockerID string
	Command  []string
	Started  bool
	version  interface{}
}

// Version is an implementation of client.Node
func (a *Action) Version() interface{} { return a.version }

// SetVersion is an implementation of client.Node
func (a *Action) SetVersion(version interface{}) { a.version = version }

// ListenAction listens for new actions for a particular host
func ListenAction(conn client.Connection, hostID string) {
	// Make the path if it doesn't exist
	node := actionPath(hostID)
	if err := conn.CreateDir(node); err != nil {
		glog.Errorf("Could not create path %s: %s", node, err)
		return
	}

	// Wait for action commands
	for {
		nodes, event, err := conn.ChildrenW(node)
		if err != nil {
			glog.Errorf("Could not listen for commands %s: %s", node, err)
			return
		}

		for _, id := range nodes {
			// Get the request
			path := actionPath(hostID, id)
			var action Action
			if err := conn.Get(path, &action); err != nil {
				glog.V(1).Infof("Could not get action at %s: %s", path, err)
				continue
			}

			// action already started, continue
			if action.Started {
				continue
			}

			// do action
			glog.V(1).Infof("Performing action to service state via request: %v", &action)
			action.Started = true
			if err := conn.Set(path, &action); err != nil {
				glog.Warningf("Could not update command at %s", path, err)
				continue
			}

			go func() {
				defer conn.Delete(path)
				result, err := utils.RunNSEnter(action.DockerID, action.Command)
				if result != nil && len(result) > 0 {
					glog.Info(string(result))
				}
				if err != nil {
					glog.Warningf("Error running command `%s` on container %s: %s", action.Command, action.DockerID, err)
				} else {
					glog.V(1).Infof("Successfully ran command `%s` on container %s", action.Command, action.DockerID)
				}
			}()
		}

		// wait for an event that something changed
		<-event
	}
}

// SendAction sends an action request to a particular host
func SendAction(conn client.Connection, action *Action) (string, error) {
	uuid, err := utils.NewUUID()
	if err != nil {
		return "", err
	}

	node := actionPath(action.HostID, uuid)
	if err := conn.Create(node, action); err != nil {
		return "", err
	}
	return path.Base(node), nil
}