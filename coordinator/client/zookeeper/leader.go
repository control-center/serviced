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
	"math"
	"path"
	"strconv"
	"strings"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
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
	c        *zklib.Conn
	path     string
	lockPath string
}

// NewLeader instantiates a new leader for a given path
func NewLeader(conn *zklib.Conn, path string) *Leader {
	return &Leader{
		c:        conn,
		path:     path,
		lockPath: "",
	}
}

// Current returns the currect elected leader and deserializes it in to node.
// It will return ErrNoLeaderFound if no leader has been elected.
func (l *Leader) Current(node client.Node) error {
	path, _, err := l.getLowestSequence()
	if err != nil {
		return err
	}
	bytes, stat, err := l.c.Get(path)
	if err != nil {
		return xlateError(err)
	}
	if len(bytes) == 0 {
		return client.ErrEmptyNode
	}
	if err := json.Unmarshal(bytes, node); err != nil {
		return client.ErrSerialization
	}
	node.SetVersion(stat)
	return nil
}

// TakeLead attempts to aquire the leader role. When aquired it returns a
// channel on the leader node so the caller can react to changes in zookeeper
func (l *Leader) TakeLead(node client.Node, cancel <-chan struct{}) (<-chan client.Event, error) {
	if l.lockPath != "" {
		return nil, ErrDeadlock
	}
	bytes, err := json.Marshal(node)
	if err != nil {
		return nil, xlateError(err)
	}
	prefix := l.prefix()
	if err := l.ensurePath(prefix); err != nil {
		return nil, err
	}
	l.lockPath, err = l.c.CreateProtectedEphemeralSequential(prefix, bytes, zklib.WorldACL(zklib.PermAll))
	if err != nil {
		return nil, err
	}
	lockSeq, err := parseSeq(l.lockPath)
	if err != nil {
		return nil, err
	}
	// This implements the leader election recipe recommeded by ZooKeeper
	// https://zookeeper.apache.org/doc/trunk/recipes.html#sc_leaderElection
	for {
		leader, seq, err := l.getLowestSequence()
		if err != nil {
			return nil, err
		}
		exists, _, ch, err := l.c.ExistsW(leader)
		if err != nil && err != zklib.ErrNoNode {
			return nil, xlateError(err)
		} else if !exists {
			l.c.CancelEvent(ch)
			continue
		}
		if leader == l.lockPath {
			return l.toClientEvent(ch, cancel), nil
		} else if seq > lockSeq {
			l.c.CancelEvent(ch)
			return nil, client.ErrNoNode
		}
		if ev := <-ch; ev.Err != nil {
			return nil, xlateError(ev.Err)
		}
	}
}

func (l *Leader) toClientEvent(ch <-chan zklib.Event, cancel <-chan struct{}) <-chan client.Event {
	evCh := make(chan client.Event, 1)
	go func() {
		select {
		case zkEv := <-ch:
			ev := client.Event{Type: client.EventType(zkEv.Type)}
			select {
			case evCh <- ev:
			case <-cancel:
			}
		case <-cancel:
			l.c.CancelEvent(ch)
		}
	}()
	return evCh
}

// ReleaseLead release the current leader role. It will return ErrNotLocked if
// the current object is not locked.
func (l *Leader) ReleaseLead() error {
	if l.lockPath == "" {
		return ErrNotLocked
	}
	if err := l.c.Delete(l.lockPath, -1); err != nil {
		return xlateError(err)
	}
	l.lockPath = ""
	return nil
}

// prefix returns the node's name prefix
func (l *Leader) prefix() string {
	return path.Join(l.path, "leader-")
}

// getLowestSequence returns the node in the path of the lowest sequence
func (l *Leader) getLowestSequence() (string, uint64, error) {
	children, _, err := l.c.Children(l.path)
	if err != nil {
		return "", 0, xlateError(err)
	}
	var lowestSeq uint64 = math.MaxUint64
	firstChild := ""
	for _, p := range children {
		s, err := parseSeq(p)
		if err != nil {
			return "", 0, xlateError(err)
		}
		if s < lowestSeq {
			lowestSeq = s
			firstChild = p
		}
	}
	if lowestSeq == math.MaxUint64 {
		return "", 0, ErrNoLeaderFound
	}
	return path.Join(l.path, firstChild), lowestSeq, nil
}

// ensurePath makes sure the dirpath leading to the node is available
func (l *Leader) ensurePath(p string) error {
	dp := path.Dir(p)
	exists, _, err := l.c.Exists(dp)
	if err != nil && err != zklib.ErrNoNode {
		return xlateError(err)
	}
	if !exists {
		if err := l.ensurePath(dp); err != nil {
			return err
		}
		if _, err := l.c.Create(dp, []byte{}, 0, zklib.WorldACL(zklib.PermAll)); err != nil && err != zklib.ErrNodeExists {
			return xlateError(err)
		}
	}
	return nil
}

func parseSeq(path string) (uint64, error) {
	parts := strings.Split(path, "-")
	return strconv.ParseUint(parts[len(parts)-1], 10, 64)
}
