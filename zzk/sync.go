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
	"errors"
	"path"
	"sync"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

// DefaultRetryTime is the time to retry a failed local operation
const DefaultRetryTime = time.Minute

// ErrInvalidType is the error for invalid zk data types
var ErrInvalidType = errors.New("invalid type")

// NewListener instantiates a new listener object
type NewListener func(string) Listener

// SyncHandler is the handler for the Synchronizer
type SyncHandler interface {
	// GetPath gets the path to the node
	GetPath(...string) string
	// Ready implements Listener
	Ready() error
	// Done implements Listener
	Done()
	// GetConnection acquires a path-based connection
	GetConnection(string) (client.Connection, error)
	// Allocate initialized a new Node object
	Allocate() Node
	// GetAll gets all local data
	GetAll() ([]Node, error)
	// AddUpdate performs a local update
	AddUpdate(string, Node) (string, error)
	// Delete deletes a Node locally
	Delete(string) error
}

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

// Synchronizer is the remote synchronizer object
type Synchronizer struct {
	SyncHandler
	conn   client.Connection
	lfuncs []NewListener
}

// NewSynchronizer instantiates a new synchronizer
func NewSynchronizer(handler SyncHandler) *Synchronizer {
	return &Synchronizer{SyncHandler: handler}
}

// AddListener creates new Listener objects based on the Synchronizer's child nodes
func (l *Synchronizer) AddListener(f NewListener) { l.lfuncs = append(l.lfuncs, f) }

// SetConnection implements Listener
func (l *Synchronizer) SetConnection(conn client.Connection) { l.conn = conn }

// PostProcess deletes any orphaned data that exists locally
func (l *Synchronizer) PostProcess(processing map[string]struct{}) {
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

// Spawn starts the remote Synchronizer based on nodeID
func (l *Synchronizer) Spawn(shutdown <-chan interface{}, nodeID string) {
	// Start dependent listeners
	var (
		wg        sync.WaitGroup
		_shutdown = make(chan interface{})
		stopping  = make(chan interface{})
	)

	defer func() {
		close(_shutdown)
		wg.Wait()
	}()

	var id string
	var wait <-chan time.Time
	done := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&done)
	for {
		node := l.Allocate()
		event, err := l.conn.GetW(l.GetPath(nodeID), node, done)
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
		}

		if key, err := l.AddUpdate(id, node); err == ErrInvalidType {
			glog.Errorf("Invalid type detected")
			return
		} else if err != nil {
			glog.Errorf("Could not update node at %s: %s", l.GetPath(nodeID), err)
			wait = time.After(DefaultRetryTime)
		} else if id == "" {
			wg.Add(1)
			go func() {
				defer wg.Done()
				if len(l.lfuncs) == 0 {
					return
				}
				defer close(stopping)
				l.startListeners(_shutdown, nodeID)
			}()
			id = key
		}

		select {
		case <-event:
		case <-wait:
		case <-stopping:
			return
		case <-shutdown:
			return
		}

		close(done)
		done = make(chan struct{})
	}
}

func (l *Synchronizer) startListeners(shutdown <-chan interface{}, nodeID string) {
	var conn client.Connection
	select {
	case conn = <-Connect(l.GetPath(nodeID), l.GetConnection):
		if conn == nil {
			return
		}
	case <-shutdown:
		return
	}

	listeners := make([]Listener, len(l.lfuncs))
	for i, newListener := range l.lfuncs {
		listeners[i] = newListener(nodeID)
	}

	start(shutdown, conn, listeners...)
}
