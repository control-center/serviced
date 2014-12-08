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

package zookeeper

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/control-center/serviced/coordinator/client"
	zklib "github.com/samuel/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

var (
	// ErrDeadlock is returned when a lock is aquired twice on the same object.
	ErrDeadlock = errors.New("zk: trying to acquire a lock twice")

	// ErrNotLocked is returned when a caller attempts to release a lock that
	// has not been aquired
	ErrNotLocked = errors.New("zk: not locked")

	// ErrNoLeaderFound is returned when a leader has not been elected
	ErrNoLeaderFound = errors.New("zk: no leader found")
)

// Leader is an object to facilitate creating an election in zookeeper.
type Leader struct {
	c        *Connection
	path     string
	lockPath string
	seq      uint64
	node     client.Node
}

func parseSeq(path string) (uint64, error) {
	parts := strings.Split(path, "-")
	return strconv.ParseUint(parts[len(parts)-1], 10, 64)
}

func (l *Leader) prefix() string {
	return join(l.path, "leader-")
}

// Current returns the currect elected leader and deserializes it in to node.
// It will return ErrNoLeaderFound if no leader has been elected.
func (l *Leader) Current(node client.Node) (err error) {

	children, _, err := l.c.conn.Children(l.path)
	if err != nil {
		return xlateError(err)
	}

	var lowestSeq uint64
	lowestSeq = math.MaxUint64
	path := ""
	for _, p := range children {
		s, err := parseSeq(p)
		if err != nil {
			return xlateError(err)
		}
		if s < lowestSeq {
			lowestSeq = s
			path = p
		}
	}
	if lowestSeq == math.MaxUint64 {
		return ErrNoLeaderFound
	}
	path = fmt.Sprintf("%s/%s", l.path, path)
	data, stat, err := l.c.conn.Get(path)
	err = json.Unmarshal(data, node)
	node.SetVersion(stat)
	return xlateError(err)
}

// TakeLead attempts to aquire the leader role. When aquired it return a
// channel on the leader node so the caller can react to changes in zookeeper
func (l *Leader) TakeLead() (echan <-chan client.Event, err error) {
	if l.lockPath != "" {
		return nil, ErrDeadlock
	}

	prefix := l.prefix()

	data, err := json.Marshal(l.node)
	if err != nil {
		return nil, xlateError(err)
	}

	path := ""
	for i := 0; i < 3; i++ {
		if l.c.conn == nil {
			// TODO: race condition exists
			return nil, fmt.Errorf("connection lost")
		}
		path, err = l.c.conn.CreateProtectedEphemeralSequential(prefix, data, zklib.WorldACL(zklib.PermAll))

		if err == zklib.ErrNoNode {
			// Create parent node.
			parts := strings.Split(l.path, "/")
			pth := ""
			for _, p := range parts[1:] {
				pth += "/" + p
				if l.c.conn == nil {
					// TODO: race condition exists
					return nil, fmt.Errorf("connection lost")
				}
				_, err := l.c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
				if err != nil && err != zklib.ErrNodeExists {
					return nil, xlateError(err)
				}
			}
		} else if err == nil {
			break
		} else {
			return nil, xlateError(err)
		}
	}
	if err != nil {
		return nil, xlateError(err)
	}
	seq, err := parseSeq(path)
	if err != nil {
		return nil, xlateError(err)
	}

	// This implements the leader election recipe recommeded by ZooKeeper
	// https://zookeeper.apache.org/doc/trunk/recipes.html#sc_leaderElection
	for {
		children, _, err := l.c.conn.Children(l.path)
		if err != nil {
			return nil, xlateError(err)
		}

		lowestSeq := seq
		var prevSeq uint64
		prevSeqPath := ""
		// find the lowest sequenced node
		for _, p := range children {
			s, err := parseSeq(p)
			if err != nil {
				return nil, xlateError(err)
			}
			if s < lowestSeq {
				lowestSeq = s
			}
			if s < seq && s > prevSeq {
				prevSeq = s
				prevSeqPath = p
			}
		}

		if seq == lowestSeq {
			// Acquired the lock
			break
		}

		// Wait on the node next in line for the lock
		_, _, ch, err := l.c.conn.GetW(l.path + "/" + prevSeqPath)
		if err != nil && err != zklib.ErrNoNode {
			return nil, xlateError(err)
		} else if err != nil && err == zklib.ErrNoNode {
			// try again
			continue
		}

		ev := <-ch
		if ev.Err != nil {
			return nil, xlateError(ev.Err)
		}
	}

	glog.Infof("w %s", path)
	_, _, event, err := l.c.conn.GetW(path)
	glog.Infof("calling wait on %s: %s", path, err)
	if err == nil {
		l.seq = seq
		l.lockPath = path
	} else {
		l.c.Delete(path)
	}
	return toClientEvent(event), xlateError(err)
}

// ReleaseLead release the current leader role. It will return ErrNotLocked if
// the current object is not locked.
func (l *Leader) ReleaseLead() error {
	if l.lockPath == "" {
		return ErrNotLocked
	}
	if l.c.conn == nil {
		// TODO: race condition exists
		return fmt.Errorf("lost connection")
	}
	if err := l.c.conn.Delete(l.lockPath, -1); err != nil {
		return xlateError(err)
	}
	l.lockPath = ""
	l.seq = 0
	return nil
}
