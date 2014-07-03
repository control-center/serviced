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
	version  interface{}
}

// Version is an implementation of client.Node
func (a *Action) Version() interface{} { return a.version }

// SetVersion is an implementation of client.Node
func (a *Action) SetVersion(version interface{}) { a.version = version }

// ActionHandler handles all non-zookeeper interactions required by the Action
type ActionHandler interface {
	AttachAndRun(dockerID string, command []string) ([]byte, error)
}

// ActionListener is the listener object for /docker/actions
type ActionListener struct {
	conn    client.Connection
	handler ActionHandler
	hostID  string
}

// NewActionListener instantiates a new action listener for /docker/actions
func NewActionListener(conn client.Connection, handler ActionHandler, hostID string) *ActionListener {
	return &ActionListener{conn, handler, hostID}
}

// Listen listens for new actions for a particular host
func (l *ActionListener) Listen(shutdown <-chan interface{}) {
	var (
		processing = make(map[string]interface{})
		done       = make(chan string)
	)

	apath := actionPath(l.hostID)
	if exists, err := l.conn.Exists(apath); err != nil {
		glog.Error("Unable to look up docker path on zookeeper: ", err)
		return
	} else if exists {
		// pass
	} else if err := l.conn.CreateDir(apath); err != nil {
		glog.Error("Unable to create docker path on zookeeper: ", err)
		return
	}

	// Wait for action commands
	for {
		nodes, event, err := l.conn.ChildrenW(apath)
		if err != nil {
			glog.Errorf("Could not listen for commands %s: %s", apath, err)
			return
		}

		for _, actionID := range nodes {
			if _, ok := processing[actionID]; !ok {
				glog.V(1).Infof("Performing action to service state via request: %s", actionID)
				processing[actionID] = nil

				// do action
				go l.doAction(done, actionID)
			}
		}

		select {
		case e := <-event:
			glog.V(2).Infof("Receieved docker action event: %v", e)
		case actionID := <-done:
			glog.V(2).Info("Cleaning up action ", actionID)
			delete(processing, actionID)
		case <-shutdown:
			return
		}
	}
}

func (l *ActionListener) doAction(done chan<- string, actionID string) {
	apath := actionPath(l.hostID, actionID)

	defer func() {
		glog.V(2).Info("Action complete: ", actionID)
		l.conn.Delete(apath)
		done <- actionID
	}()

	var action Action
	if err := l.conn.Get(apath, &action); err != nil {
		glog.V(1).Infof("Could not get action %s: %s", apath, err)
		return
	}

	result, err := l.handler.AttachAndRun(action.DockerID, action.Command)
	if result != nil && len(result) > 0 {
		glog.Info(string(result))
	}
	if err != nil {
		glog.Warningf("Error running command `%s` on container %s: %s", action.Command, action.DockerID, err)
	} else {
		glog.V(1).Infof("Successfully ran command `%s` on container %s", action.Command, action.DockerID)
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
	return uuid, nil
}