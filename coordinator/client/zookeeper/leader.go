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
	lpath "path"
	"strconv"
	"strings"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/control-center/serviced/coordinator/client"
	"github.com/zenoss/glog"
)

var (
	// ErrNoLeaderFound is returned when a leader has not been elected
	ErrNoLeaderFound = errors.New("zk: no leader found")
)

// Leader is an object to facilitate creating an election in zookeeper.
type Leader struct {
	c        *Connection
	path     string
	lockPath string
	seq      int64 // Note: ZK sequence number is actually signed int32
	node     client.Node
}

func parseSeq(path string) (int64, error) {
	parts := strings.Split(path, "-")
	return strconv.ParseInt(parts[len(parts)-1], 10, 64)
}

func (l *Leader) prefix() string {
	return join(l.path, "leader-")
}

// Current returns the currect elected leader and deserializes it in to node.
// It will return ErrNoLeaderFound if no leader has been elected.
func (l *Leader) Current(node client.Node) (err error) {
	lock := l.c.newRWLock(lpath.Dir(l.path))
	if lockerr := lock.RLock(); lockerr != nil {
		glog.Errorf("Could not acquire read lock for %s: %s", lpath.Dir(l.path), lockerr)
		return lockerr
	}
	children, _, err := l.c.conn.Children(l.path)
	lock.Unlock()

	if err != nil {
		return xlateError(err)
	}

	var lowestSeq int64
	lowestSeq = math.MaxInt64
	leader := ""
	for _, p := range children {
		s, err := parseSeq(p)
		if err != nil {
			return xlateError(err)
		}
		if s < lowestSeq {
			lowestSeq = s
			leader = p
		}
	}
	if lowestSeq == math.MaxInt64 {
		return ErrNoLeaderFound
	}
	leader = join(l.path, leader)

	lock = l.c.newRWLock(lpath.Dir(leader))
	if lockerr := lock.RLock(); lockerr != nil {
		glog.Errorf("Could not acquire read lock for %s: %s", lpath.Dir(leader), lockerr)
		return lockerr
	}
	data, stat, err := l.c.conn.Get(leader)
	lock.Unlock()

	err = json.Unmarshal(data, node)
	node.SetVersion(stat)
	return xlateError(err)
}

// TakeLead attempts to aquire the leader role. When aquired it return a
// channel on the leader node so the caller can react to changes in zookeeper
func (l *Leader) TakeLead(done <-chan struct{}) (echan <-chan client.Event, err error) {
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

		lock := l.c.newRWLock(lpath.Dir(prefix))
		if lockerr := lock.Lock(); lockerr != nil {
			glog.Errorf("Could not acquire write lock for %s: %s", lpath.Dir(prefix), lockerr)
			return nil, lockerr
		}
		path, err = l.c.conn.CreateProtectedEphemeralSequential(prefix, data, zklib.WorldACL(zklib.PermAll))
		lock.Unlock()

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

				lock := l.c.newRWLock(lpath.Dir(pth))
				if lockerr := lock.Lock(); lockerr != nil {
					glog.Errorf("Could not acquire write lock for %s: %s", lpath.Dir(pth), lockerr)
					return nil, lockerr
				}
				_, err := l.c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
				lock.Unlock()

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
		lock := l.c.newRWLock(lpath.Dir(l.path))
		if lockerr := lock.RLock(); lockerr != nil {
			glog.Errorf("Could not acquire read lock for %s: %s", lpath.Dir(l.path), lockerr)
			return nil, lockerr
		}
		children, _, err := l.c.conn.Children(l.path)
		lock.Unlock()

		if err != nil {
			return nil, xlateError(err)
		}

		lowestSeq := seq
		prevSeq := int64(-1)
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
		nextLockNode := join(l.path, prevSeqPath)
		glog.V(2).Info("Waiting on leader lock: %s", nextLockNode)
		lock = l.c.newRWLock(lpath.Dir(nextLockNode))
		if lockerr := lock.RLock(); lockerr != nil {
			glog.Errorf("Could not acquire read lock for %s: %s", lpath.Dir(nextLockNode), lockerr)
			return nil, lockerr
		}
		_, _, ch, err := l.c.conn.GetW(nextLockNode)
		lock.Unlock()

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
	lock := l.c.newRWLock(lpath.Dir(path))
	if lockerr := lock.RLock(); lockerr != nil {
		glog.Errorf("Could not acquire read lock for %s: %s", lpath.Dir(path), lockerr)
		return nil, lockerr
	}
	_, _, event, err := l.c.conn.GetW(path)
	lock.Unlock()

	glog.Infof("from calling wait on %s: %s", path, err)
	if err == nil {
		l.seq = seq
		l.lockPath = path
		echan = l.c.toClientEvent(event, done)
	} else {
		err = xlateError(err)
		l.c.Delete(path)
	}
	return
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

	lock := l.c.newRWLock(lpath.Dir(l.lockPath))
	if lockerr := lock.Lock(); lockerr != nil {
		glog.Errorf("Could not acquire write lock for %s: %s", lpath.Dir(l.lockPath), lockerr)
		return lockerr
	}
	err := l.c.conn.Delete(l.lockPath, -1)
	lock.Unlock()

	if err != nil {
		return xlateError(err)
	}
	l.lockPath = ""
	l.seq = 0
	return nil
}
