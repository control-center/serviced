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
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
)

const retryLimit = 2

// Errors
var (
	ErrTimeout  = errors.New("connection timeout")
	ErrShutdown = errors.New("listener shutdown")
	ErrBadConn  = errors.New("bad connection")
)

// HostLeader is the node to store leader information for a host
type HostLeader struct {
	HostID  string
	Realm   string
	version interface{}
}

// Version implements client.Node
func (node *HostLeader) Version() interface{} { return node.version }

// SetVersion implements client.Node
func (node *HostLeader) SetVersion(version interface{}) { node.version = version }

// GetHostID finds the host of a led node
func GetHostID(leader client.Leader) (string, error) {
	var hl HostLeader
	if err := leader.Current(&hl); err != nil {
		return "", err
	}
	return hl.HostID, nil
}

func MonitorRealm(shutdown <-chan interface{}, conn client.Connection, path string) <-chan string {
	realmC := make(chan string)

	go func() {
		defer close(realmC)
		var realm string
		leader, err := conn.NewLeader(path)
		if err != nil {
			return
		}
		done := make(chan struct{})
		defer func(channel *chan struct{}) { close(*channel) }(&done)
		for {
			// monitor path for changes
			_, event, err := conn.ChildrenW(path, done)
			if err != nil {
				return
			}

			// Get the current leader and check for changes in its realm
			var hl HostLeader
			if err := leader.Current(&hl); err == zookeeper.ErrNoLeaderFound {
				// pass
			} else if err != nil {
				return
			} else if hl.Realm != realm {
				realm = hl.Realm
				select {
				case realmC <- realm:
				case <-shutdown:
					return
				}
			}

			select {
			case <-event:
			case <-shutdown:
				return
			}

			close(done)
			done = make(chan struct{})
		}
	}()
	return realmC
}

// Listener is zookeeper node listener type
type Listener interface {
	// SetConnection sets the connection object
	SetConnection(conn client.Connection)
	// GetPath concatenates the base path with whatever child nodes that are specified
	GetPath(nodes ...string) string
	// Ready verifies that the listener can start listening
	Ready() error
	// Done performs any cleanup when the listener exits
	Done()
	// Spawn is the action to be performed when a child node is found on the parent
	Spawn(<-chan interface{}, string)
	// PostProcess performs additional action based on the nodes that are in processing
	PostProcess(p map[string]struct{})
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
	ok, err := conn.Exists("/")
	if err != nil {
		return err
	} else if !ok {
		return client.ErrNoNode
	}

	done := make(chan struct{})
	defer func() { close(done) }()
	for {
		ok, ev, err := conn.ExistsW(p, done)
		if err != nil {
			return err
		} else if ok {
			return nil
		}

		select {
		case <-ev:
		case <-shutdown:
			return ErrShutdown
		}
		close(done)
		done = make(chan struct{})
	}
}

// Listen initializes a listener for a particular zookeeper node
// shutdown:	signal to shutdown the listener
// ready:		signal to indicate that the listener has started watching its
//				child nodes (must set buffer size >= 1)
// l:			object that manages the zk interface for a specific path
func Listen(shutdown <-chan interface{}, ready chan<- error, conn client.Connection, l Listener) {
	var (
		_shutdown  = make(chan interface{})
		done       = make(chan string)
		processing = make(map[string]struct{})
	)

	l.SetConnection(conn)
	logger := plog.WithField("path", l.GetPath())
	logger.Info("Starting a listener")
	if err := Ready(shutdown, conn, l.GetPath()); err != nil {
		logger.WithError(err).Error("Could not start listener")
		ready <- err
		return
	} else if err := l.Ready(); err != nil {
		logger.WithError(err).Error("Could not start listener, listener not ready")
		ready <- err
		return
	}

	close(ready)

	defer func() {
		logger.Info("Listener received interrupt")
		l.Done()
		close(_shutdown)
		for len(processing) > 0 {
			delete(processing, <-done)
		}
	}()

	logger.Debug("Listener started; waiting for data")
	doneW := make(chan struct{})
	defer func(channel *chan struct{}) { close(*channel) }(&doneW)
	for {
		nodes, event, err := conn.ChildrenW(l.GetPath(), doneW)
		if err != nil {
			logger.WithError(err).Error("Could not watch for nodes")
			return
		}

		for _, node := range nodes {
			if _, ok := processing[node]; !ok {
				nodeLogger := logger.WithField("nodepath", l.GetPath(node))
				nodeLogger.Debug("Spawning a goroutine for node")
				processing[node] = struct{}{}
				go func(node string) {
					defer func() {
						nodeLogger.Debug("Goroutine for node was shutdown")
						done <- node
					}()
					l.Spawn(_shutdown, node)
				}(node)
			}
		}

		l.PostProcess(processing)

		select {
		case e := <-event:
			if e.Type == client.EventNodeDeleted {
				logger.Debug("Node has been removed; shutting down listener")
				return
			} else if e.Type == client.EventSession || e.Type == client.EventNotWatching {
				logger.Warn("Node had a reconnect; resetting listener")
				if err := l.Ready(); err != nil {
					logger.Errorf("Could not ready listener; shutting down")
					return
				}
			}
			logger.WithField("event", e).Debug("Node received event")
		case node := <-done:
			logger.WithField("nodepath", l.GetPath(node)).Debug("Cleaning up node")
			delete(processing, node)
		case <-shutdown:
			logger.Debug("shutting down listener")
			return
		}

		close(doneW)
		doneW = make(chan struct{})
	}
}

// Start starts a group of listeners that are governed by a master listener.
// When the master exits, it shuts down all of the child listeners and waits
// for all of the subprocesses to exit
func Start(shutdown <-chan interface{}, conn client.Connection, master Listener, listeners ...Listener) {
	logger := plog.WithField("path", master.GetPath())
	logger.Debug("master listener starting")
	defer logger.Debug("master listener exit")

	// shutdown the parent and child listeners
	_shutdown := make(chan interface{})

	// start the master
	masterDone := make(chan struct{})
	defer func() { <-masterDone }()
	masterReady := make(chan error, 1)
	go func() {
		defer close(masterDone)
		Listen(_shutdown, masterReady, conn, master)
	}()

	// wait for the master to be ready and then start the slaves
	var childDone chan struct{}
	select {
	case err := <-masterReady:
		if err != nil {
			logger.WithError(err).Error("master listener failed to start")
			return
		}

		childDone := make(chan struct{})
		defer func() { <-childDone }()

		go func() {
			defer close(childDone)
			// this handles restarts; retryLimit to reduce flapping
			for i := 0; i <= retryLimit; i++ {
				start(_shutdown, conn, listeners...)
				select {
				case <-_shutdown:
					return
				default:
					logger.Warning("Restarting child listeners for master")
				}
			}
			logger.Warning("Shutting down master listener; child listeners exceeded retry limit")
		}()
	case <-masterDone:
		logger.Debug("master listener done")
	case <-shutdown:
		logger.Debug("master listener received shutdown")
	}

	defer close(_shutdown)
	select {
	case <-masterDone:
		logger.Warning("Master listener died prematurely; shutting down")
	case <-childDone:
		logger.Warning("Child listeners for master died prematurely; shutting down")
	case <-shutdown:
		logger.Info("Received signal to shutdown for master listener")
	}
}

func start(shutdown <-chan interface{}, conn client.Connection, listeners ...Listener) {
	var count int
	done := make(chan int)
	defer func() {
		plog.WithField("count", len(listeners)).Info("Shutting down child listeners")
		for count > 0 {
			count -= <-done
		}
	}()

	_shutdown := make(chan interface{})
	defer close(_shutdown)

	for i := range listeners {
		count++
		go func(l Listener) {
			defer func() { done <- 1 }()
			Listen(_shutdown, make(chan error, 1), conn, l)
			plog.WithField("path", l.GetPath()).Info("Listener exited")
		}(listeners[i])
	}

	select {
	case i := <-done:
		plog.Warn("Listener exited prematurely, stopping all listeners")
		count -= i
	case <-shutdown:
		plog.Info("Received signal to shutdown")
	}
}
