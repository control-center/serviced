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

package zzk

import (
	"path"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

const (
	DefaultRetryTime = time.Minute
)

// Node manages zookeeper actions
type Node interface {
	client.Node
	// GetID relates to the child node mapping in zookeeper
	GetID() string
	// Create creates the object in zookeeper
	Create(conn client.Connection) error
	// Update updates the object in zookeeper
	Update(conn client.Connection) error
}

// Sync synchronizes zookeeper data with what is in elastic or any other storage facility
func Sync(conn client.Connection, data []Node, zkpath string) error {
	var current []string
	if exists, err := PathExists(conn, zkpath); err != nil {
		return err
	} else if !exists {
		// pass
	} else if current, err = conn.Children(zkpath); err != nil {
		return err
	}

	datamap := make(map[string]Node)
	for i, node := range data {
		datamap[node.GetID()] = data[i]
	}

	for _, id := range current {
		if node, ok := datamap[id]; ok {
			glog.V(2).Infof("Updating id:'%s' at zkpath:%s with: %+v", id, zkpath, node)
			if err := node.Update(conn); err != nil {
				return err
			}
			delete(datamap, id)
		} else {
			glog.V(2).Infof("Deleting id:'%s' at zkpath:%s not found in elastic\nzk current children: %v", id, zkpath, current)
			if err := conn.Delete(path.Join(zkpath, id)); err != nil {
				return err
			}
		}
	}

	for id, node := range datamap {
		glog.V(2).Infof("Creating id:'%s' at zkpath:%s with: %+v", id, zkpath, node)
		if err := node.Create(conn); err != nil {
			return err
		}
	}

	return nil
}

// SyncHandler is the handler for synchronizing remote coordinator to  local db
type SyncHandler interface {
	GetPathBasedConnection(path string) (client.Connection, error)
	GetPath(nodes ...string) string
	GetAll() ([]Node, error)
	AddOrUpdate(nodeID string, node Node) error
	Delete(nodeID string) error
}

// SyncListener is the listener for synchronizing remote coordinator to local db
type SyncListener struct {
	SyncHandler
	conn        client.Connection
	getListener []func(conn client.Connection, nodeID string) Listener
}

// NewSyncListener instantiates a new SyncListener
func NewSyncListener(conn client.Connection, handler SyncHandler) *SyncListener {
	return &SyncListener{SyncHandler: handler, conn: conn}
}

// AddListener adds a new Listener
func (l *SyncListener) AddListener(lfunc func(conn client.Connection, nodeID string) Listener) {
	l.getListener = append(l.getListener, lfunc)
}

// GetConnection gets the coordinator client connection
func (l *SyncListener) GetConnection() client.Connection { return l.conn }

// Ready deletes data from the db that does not exist in the remote coordinator
func (l *SyncListener) Ready() error { return nil }

// Done implements Listener
func (l *SyncListener) Done() {}

func (l *SyncListener) PostProcess(processing map[string]struct{}) {
	// Get all locally stored data
	nodes, err := l.GetAll()
	if err != nil {
		glog.Warningf("Could not access locally stored data: %s", err)
		return
	}

	// Delete any local nodes that do not exist remotely and are not in process
	for _, node := range nodes {
		if _, ok := processing[node.GetID()]; ok {
			// pass
		} else if err := l.Delete(node.GetID()); err != nil {
			glog.Warningf("Could not delete %s from locally stored data: %s", node.GetID(), err)
		}
	}
}

func (l *SyncListener) Spawn(shutdown <-chan interface{}, nodeID string) {
	// Start any dependent listeners
	_shutdown := make(chan interface{})
	var wg sync.WaitGroup

	go func() {
		wg.Add(1)
		defer wg.Done()
		l.startListeners(_shutdown, nodeID)
	}()

	defer func() {
		close(_shutdown)
		wg.Wait()
	}()

	var id string
	var wait <-chan time.Time
	for {
		var node Node
		event, err := l.conn.GetW(l.GetPath(nodeID), node)

		if err == client.ErrNoNode && id != "" {
			if err := l.Delete(id); err != nil {
				glog.Errorf("Could not delete node at %s: %s", l.GetPath(nodeID), err)
				wait = time.After(DefaultRetryTime)
			} else {
				return
			}
		} else if err != nil {
			glog.Errorf("Could not get node at %s: %s", l.GetPath(nodeID), err)
			return
		} else if err := l.AddOrUpdate(nodeID, node); err != nil {
			glog.Errorf("Could not update node at %s: %s", l.GetPath(nodeID), err)
			wait = time.After(time.Minute)
		}
		// Need to store the id if the node is ephemeral
		id = node.GetID()

		select {
		case <-event:
		case <-wait:
		case <-shutdown:
			return
		}
	}
}

func (l *SyncListener) startListeners(shutdown <-chan interface{}, nodeID string) {
	var conn client.Connection
	select {
	case conn = <-Connect(l.GetPath(nodeID), l.GetPathBasedConnection):
		if conn == nil {
			return
		}
	case <-shutdown:
		return
	}

	listeners := make([]Listener, len(l.getListener))
	for i, gl := range l.getListener {
		listeners[i] = gl(conn, nodeID)
	}

	Start(shutdown, nil, listeners...)
}
