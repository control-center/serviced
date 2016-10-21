// Copyright 2016 The Serviced Authors.
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

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
)

type watcher struct {
	c chan<- struct{}
}

// LeaderListener is generic watcher and broadcaster of leader types
type LeaderListener struct {
	path     string
	mu       *sync.Mutex
	watchers []watcher // TODO: we may want to change this to a type
}

// NewLeaderListener instantiates a listener to watch the leader election at a
// given path.
func NewLeaderListener(path string) *LeaderListener {
	return &LeaderListener{
		path:     path,
		mu:       &sync.Mutex{},
		watchers: make([]watcher, 0),
	}
}

// Wait enqueues a watcher that will be updated when a new leader is elected
func (l *LeaderListener) Wait() <-chan struct{} {
	l.mu.Lock()
	defer l.mu.Unlock()
	var c = make(chan struct{}, 1)
	l.watchers = append(l.watchers, watcher{c: c})
	return c
}

// broadcast alerts all the watchers that a new leader has been elected
func (l *LeaderListener) broadcast() { // TODO: we may want to pass in a type
	l.mu.Lock()
	defer l.mu.Unlock()
	for _, w := range l.watchers {
		w.c <- struct{}{}
	}
	l.watchers = make([]watcher, 0)
}

// Run manages the event loop for this listener
func (l *LeaderListener) Run(cancel <-chan interface{}, conn client.Connection) {
	logger := plog.WithField("path", l.path)

	done := make(chan struct{})
	defer func() { close(done) }()

	for {

		// check if the path exists
		var ok, ev, err = conn.ExistsW(l.path, done)
		if err != nil {
			logger.WithError(err).Error("Could not monitor path")
			return
		}

		// watch the path for children, if the path exists
		var ch = make([]string, 0)
		if ok {
			ch, ev, err = conn.ChildrenW(l.path, done)
			if err == client.ErrNoNode {
				logger.Debug("Path was deleted, retrying")
				close(done)
				done = make(chan struct{})
				continue
			} else if err != nil {
				logger.WithError(err).Error("Could not monitor path children")
				return
			}
		}

		// if there is a leader, figure out which one it is and update the watch
		if len(ch) > 0 {
			leader, err := zookeeper.GetLowestSequence(ch)
			if err != nil {
				logger.WithError(err).Error("Could not determine leader from nodes")
				return
			}
			logger = logger.WithField("leader", leader)

			ok, ev, err = conn.ExistsW(path.Join(l.path, leader), done)
			if err != nil {
				logger.WithError(err).Error("Could not monitor leader")
				return
			}

			// this node is no longer the leader, so try again
			if !ok {
				logger.Debug("Leader lost its lead, retrying")
				close(done)
				done = make(chan struct{})
				continue
			}

			// tell everyone that there is a new sheriff in town
			l.broadcast()
		}

		select {
		case <-ev:
		case <-cancel:
			return
		}
	}
}
