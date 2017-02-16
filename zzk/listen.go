// Copyright 2017 The Serviced Authors.
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

import "github.com/control-center/serviced/coordinator/client"

// Listener2 is for monitoring the listener and its connection to zookeeper
type Listener2 interface {

	// Listen is the method to call to start the listener
	Listen(cancel <-chan interface{}, conn client.Connection)

	// Exited does additional cleanup once shutdown is called
	Exited()
}

// Manage continuously restarts the listener until it shuts down
func Manage(shutdown <-chan interface{}, root string, l Listener2) {
	defer l.Exited()
	logger := plog.WithField("zkroot", root)

	for {
		select {
		case conn := <-connect(root):
			if conn != nil {
				logger.Info("Acquired a client connection to zookeeper")
				l.Listen(shutdown, conn)
			}
		case <-shutdown:
		}

		// shutdown takes precedence
		select {
		case <-shutdown:
			return
		default:
		}
	}
}

// Spawner manages the spawning of individual goroutines for managing nodes
// under a particular zookeeper
type Spawner interface {

	// SetConn sets the zookeeper connection
	SetConn(conn client.Connection)

	// Path returns the parent path of the zookeeper node whose children are
	// the target of spawn
	Path() string

	// Pre performs a synchronous action to occur before spawn
	Pre()

	// Spawn is intended to manage individual nodes that exist from Path()
	Spawn(cancel <-chan struct{}, n string)

	// Post presents the complete list of nodes that are children of Path() for
	// further processing and synchronization
	Post(p map[string]struct{})
}

// Listen2 manages spawning threads to handle nodes created under the parent
// path.
func Listen2(shutdown <-chan interface{}, conn client.Connection, s Spawner) {
	var (
		cancel = make(chan struct{})
		exited = make(chan string)
		active = make(map[string]struct{})
	)

	logger := plog.WithField("zkpath", s.Path())

	// set the connection
	s.SetConn(conn)

	// make sure all running spawns exit on shutdown
	defer func() {
		close(cancel)
		for len(active) > 0 {
			delete(active, <-exited)
		}
	}()

	done := make(chan struct{})
	defer func() { close(done) }()
	for {
		// wait for the path to be available
		ok, ev, err := conn.ExistsW(s.Path(), done)
		if err != nil {
			logger.WithError(err).Error("Could not watch path")
			return
		}

		// get the path's children
		ch := []string{}
		if ok {
			ch, ev, err = conn.ChildrenW(s.Path(), done)
			if err == client.ErrNoNode {
				// path was deleted, so we need to monitor the existance
				close(done)
				done = make(chan struct{})
				continue
			} else if err != nil {
				logger.WithError(err).Error("Could not watch path children")
				return
			}
		}

		// spawn a goroutine for each new node
		for _, n := range ch {
			if _, ok := active[n]; !ok {
				logger.WithField("node", n).Debug("Spawning a goroutine for node")
				s.Pre()
				active[n] = struct{}{}
				go func(n string) {
					s.Spawn(cancel, n)
					exited <- n
				}(n)
			}
		}

		// trigger post-processing actions (for orphaned nodes, for example)
		s.Post(copyMap(active))

		select {
		case <-ev:
		case n := <-exited:
			delete(active, n)
		case <-shutdown:
		}

		// shutdown takes precedence
		select {
		case <-shutdown:
			return
		default:
		}

		close(done)
		done = make(chan struct{})
	}

}

func connect(root string) <-chan client.Connection {
	return Connect(root, GetLocalConnection)
}

func copyMap(a map[string]struct{}) (b map[string]struct{}) {
	b = make(map[string]struct{})
	for i := range a {
		b[i] = struct{}{}
	}
	return
}
