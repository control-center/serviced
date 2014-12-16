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

package docker

import (
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
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
func NewActionListener(handler ActionHandler, hostID string) *ActionListener {
	return &ActionListener{handler: handler, hostID: hostID}
}

// GetConnection implements zzk.Listener
func (l *ActionListener) SetConnection(conn client.Connection) { l.conn = conn }

// GetPath implements zzk.Listener
func (l *ActionListener) GetPath(nodes ...string) string {
	return actionPath(append([]string{l.hostID}, nodes...)...)
}

// Ready implements zzk.Listener
func (l *ActionListener) Ready() (err error) { return }

// Done implements zzk.Listener
func (l *ActionListener) Done() { return }

// PostProcess implements zzk.Listener
func (l *ActionListener) PostProcess(p map[string]struct{}) {}

// Spawn attaches to a container and performs the requested action
func (l *ActionListener) Spawn(shutdown <-chan interface{}, actionID string) {
	defer func() {
		glog.V(2).Infof("Action %s complete: ", actionID)
		if err := l.conn.Delete(l.GetPath(actionID)); err != nil {
			glog.Errorf("Could not delete %s: %s", l.GetPath(actionID), err)
		}
	}()

	var action Action
	if err := l.conn.Get(l.GetPath(actionID), &action); err != nil {
		glog.V(1).Infof("Could not get action %s: %s", l.GetPath(actionID), err)
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
	} else if err := conn.Set(node, action); err != nil {
		return "", err
	}
	return uuid, nil
}
