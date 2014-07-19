// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package zzk

import (
	"errors"
	"path"
	"sync"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
)

var zClient *client.Client
var poolBasedConnections = make(map[string]client.Connection)

// Errors
var (
	ErrShutdown = errors.New("listener shutdown")
)

func InitializeGlobalCoordClient(myZClient *client.Client) {
	zClient = myZClient
}

// GeneratePoolPath is used to convert a pool ID to /pools/POOLID
func GeneratePoolPath(poolID string) string {
	return "/pools/" + poolID
}

// GetBasePathConnection returns a connection based on the basePath provided
func GetBasePathConnection(basePath string) (client.Connection, error) { // TODO figure out how/when to Close connections
	if _, ok := poolBasedConnections[basePath]; ok {
		return poolBasedConnections[basePath], nil
	}

	if zClient == nil {
		glog.Errorf("zkdao zClient has not been initialized!")
	}

	myNewConnection, err := zClient.GetCustomConnection(basePath)
	if err != nil {
		glog.Errorf("Failed to obtain a connection to %v: %v", basePath, err)
		return nil, err
	}

	// save off the new connection to the map
	poolBasedConnections[basePath] = myNewConnection

	return myNewConnection, nil
}

// HostLeader is the node to store leader information for a host
type HostLeader struct {
	HostID  string
	version interface{}
}

// Version implements client.Node
func (node *HostLeader) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *HostLeader) SetVersion(version interface{}) { node.version = version }

// NewHostLeader initializes a new host leader
func NewHostLeader(conn client.Connection, hostID, path string) client.Leader {
	return conn.NewLeader(path, &HostLeader{HostID: hostID})
}

// GetHostID finds the host of a led node
func GetHostID(leader client.Leader) (string, error) {
	var hl HostLeader
	if err := leader.Current(&hl); err != nil {
		return "", err
	}
	return hl.HostID, nil
}

// Listener is zookeeper node listener type
type Listener interface {
	GetConnection() client.Connection
	GetPath(nodes ...string) string
	Ready() error
	Done()
	Spawn(<-chan interface{}, string)
}

// PathExists verifies if a path exists and does not raise an exception if the
// path does not exist
func PathExists(conn client.Connection, p string) (bool, error) {
	exists, err := conn.Exists(p)
	if err == client.ErrNoNode {
		return false, nil
	}
	return exists, err
}

// Ready waits for a node to be available for watching
func Ready(shutdown <-chan interface{}, conn client.Connection, p string) error {
	if exists, err := PathExists(conn, p); err != nil {
		return err
	} else if exists {
		return nil
	}

	for {
		if err := Ready(shutdown, conn, path.Dir(p)); err != nil {
			return err
		} else if exists, err := PathExists(conn, p); err != nil {
			return err
		} else if exists {
			return nil
		}
		_, event, err := conn.ChildrenW(path.Dir(p))
		if err != nil {
			return err
		}
		select {
		case <-event:
			// pass
		case <-shutdown:
			return ErrShutdown
		}
	}
}

// Listen initializes a listener for a particular zookeeper node
func Listen(shutdown <-chan interface{}, l Listener) {
	var (
		_shutdown  = make(chan interface{})
		done       = make(chan string)
		processing = make(map[string]interface{})
		conn       = l.GetConnection()
	)

	glog.Infof("Starting a listener at %s", l.GetPath())
	if err := Ready(shutdown, conn, l.GetPath()); err != nil {
		glog.Errorf("Could not start listener at %s: %s", l.GetPath(), err)
		return
	} else if err := l.Ready(); err != nil {
		glog.Errorf("Could not start listener at %s: %s", l.GetPath(), err)
		return
	}

	defer func() {
		glog.Infof("Listener at %s receieved interrupt", l.GetPath())
		close(_shutdown)
		for len(processing) > 0 {
			delete(processing, <-done)
		}
		l.Done()
	}()

	glog.V(1).Infof("Listener %s started; waiting for data", l.GetPath())
	for {
		nodes, event, err := conn.ChildrenW(l.GetPath())
		if err != nil {
			glog.Errorf("Could not watch for nodes at %s: %s", l.GetPath(), err)
			return
		}

		for _, node := range nodes {
			if _, ok := processing[node]; !ok {
				glog.V(1).Infof("Spawning a goroutine for %s", l.GetPath(node))
				processing[node] = nil
				go func(node string) {
					defer func() {
						glog.V(1).Infof("Goroutine at %s was shutdown", l.GetPath(node))
						done <- node
					}()
					l.Spawn(_shutdown, node)
				}(node)
			}
		}

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				glog.V(1).Infof("Node %s has been removed; shutting down listener", l.GetPath())
				return
			}
			glog.V(4).Infof("Node %s receieved event %v", l.GetPath(), e)
		case node := <-done:
			glog.V(3).Infof("Cleaning up %s", l.GetPath(node))
			delete(processing, node)
		case <-shutdown:
			return
		}
	}
}

// Start starts a group of listeners that are governed by a master listener.
// When the master exits, it shuts down all of the child listeners and waits
// for all of the subprocesses to exit
func Start(shutdown <-chan interface{}, master Listener, listeners ...Listener) {
	var wg sync.WaitGroup
	_shutdown := make(chan interface{})
	for _, listener := range listeners {
		wg.Add(1)
		go func(l Listener) {
			// TODO: implement restarts?
			defer wg.Done()
			Listen(_shutdown, l)
		}(listener)
	}

	done := make(chan interface{})
	go func() {
		defer close(done)
		Listen(_shutdown, master)
	}()

	// Wait for the master to finish or shutdown signal received
	select {
	case <-done:
	case <-shutdown:
	}
	close(_shutdown)
	wg.Wait()
}

// Node manages zookeeper actions
type Node interface {
	ID() string
	Create(conn client.Connection) error
	Update(conn client.Connection) error
	Delete(conn client.Connection) error
}

// Sync synchronizes zookeeper data with what is in elastic or any other storage facility
func Sync(conn client.Connection, data []Node, path string) error {
	var current []string
	if exists, err := zzk.PathExists(conn, path); err != nil {
		return err
	} else if !exists{
		// pass
	} else if current, err = conn.Children(path); err != nil {
		return err
	}

	datamap := make(map[string]CRUD)
	for i, node := range data {
		datamap[node.ID()] = datamap[i]
	}

	for _, id := range current {
		if node, ok := datamap[id]; ok {
			if err := node.Update(conn); err != nil {
				return nil
			}
			delete(datamap, id)
		} else if err := node.Delete(conn); err != nil {
			return err
		}
	}

	for _, node := range datamap {
		if err := node.Create(conn); err != nil {
			return err
		}
	}

	return nil
}