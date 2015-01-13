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
	"path"
	"sort"
	"strings"

	"github.com/control-center/serviced/coordinator/client"
	zklib "github.com/samuel/go-zookeeper/zk"
)

var ErrEmptyQueue = errors.New("zk: empty queue")

type sortqueue []string

func (q sortqueue) Len() int { return len(q) }
func (q sortqueue) Less(a, b int) bool {
	sa, _ := parseSeq(q[a])
	sb, _ := parseSeq(q[b])
	return sa < sb
}
func (q sortqueue) Swap(a, b int) { q[a], q[b] = q[b], q[a] }

// Queue is the zookeeper construct for FIFO locking queue
type Queue struct {
	c        *Connection // zookeeper connection
	path     string      // path to the queue
	children []string    // children currently enqueued
	lockPath string      // lock that is owned by the queue instance
}

func (q *Queue) prefix() string {
	return join(q.path, "queue-")
}

func (q *Queue) lock(node string) (string, error) {
	// do not lock the node if a lock is already available
	if q.HasLock() {
		return "", ErrDeadlock
	}

	if q.c.conn == nil {
		// TODO: race condition exists
		return "", fmt.Errorf("connection lost")
	}

	// create a lock node
	prefix := join(q.path, node, "lock-")
	path, err := q.c.conn.CreateProtectedEphemeralSequential(prefix, []byte{}, zklib.WorldACL(zklib.PermAll))
	if err != nil {
		return "", err
	}

	seq, err := parseSeq(path)
	if err != nil {
		return "", err
	}

	for {
		if q.c.conn == nil {
			// TODO: race condition exists
			return "", fmt.Errorf("connection lost")
		}

		// ErrNoNode is ok here, because it means the parent has completed or cancelled its task
		children, _, err := q.c.conn.Children(path)
		if err != nil {
			return "", err
		}

		// check if the instance owns the lock
		lowestSeq := seq
		var prevSeq uint64 = 0
		prevSeqPath := ""
		for _, p := range children {
			s, err := parseSeq(p)
			if err != nil {
				return "", err
			}
			if s < lowestSeq {
				lowestSeq = s
			}
			if s < seq && s > prevSeq {
				prevSeq = s
				prevSeqPath = p
			}

			if seq == lowestSeq {
				// Acquired the lock
				break
			}

			// Wait on the node next in line for the lock
			_, _, ch, err := q.c.conn.GetW(join(q.path, prevSeqPath))
			if err != nil && err != zklib.ErrNoNode {
				return "", err
			} else if err != nil && err == zklib.ErrNoNode {
				// try again
				continue
			}

			ev := <-ch
			if ev.Err != nil {
				return "", ev.Err
			}
		}
	}

	return path, nil
}

// Put enqueues the desired node and returns the path to the node
func (q *Queue) Put(node client.Node) (string, error) {
	if q.c.conn == nil {
		// TODO: race condition exists
		return "", fmt.Errorf("connection lost")
	}

	prefix := q.prefix()

	data, err := json.Marshal(node)
	if err != nil {
		return "", xlateError(err)
	}

	// add the node to the queue
	path := ""
	for i := 0; i < 3; i++ {
		if q.c.conn == nil {
			// TODO: race condition exists
			return "", fmt.Errorf("connection lost")
		}

		path, err := q.c.conn.CreateProtectedEphemeralSequential(prefix, data, zklib.WorldACL(zklib.PermAll))
		if err == zklib.ErrNoNode {
			// Create parent node
			parts := strings.Split(q.path, "/")
			pth := ""
			for _, p := range parts[1:] {
				path += "/" + p
				if q.c.conn == nil {
					// TODO: race condition exists
					return "", fmt.Errorf("connection lost")
				}

				_, err := q.c.conn.Create(pth, []byte{}, 0, zklib.WorldACL(zklib.PermAll))
				if err != nil && err != zklib.ErrNodeExists {
					return "", xlateError(err)
				}
			}
		} else if err == nil {
			break
		} else {
			return "", xlateError(err)
		}
	}

	if err != nil {
		return "", xlateError(err)
	}
	return path, nil
}

// Get grabs and locks the next node in the queue
func (q *Queue) Get(node client.Node) error {
	if q.HasLock() {
		return ErrDeadlock
	}

	if q.c.conn == nil {
		// TODO: race condition exists
		return fmt.Errorf("connection lost")
	}

	for {
		if q.c.conn == nil {
			// TODO: race condition exists
			return fmt.Errorf("connection lost")
		}

		// exhaust the current list of nodes loaded to find the next available
		// to dequeue
		for _, p := range q.children {
			path := q.path + "/" + p

			// Lock the node
			lockPath, err := q.lock(path)
			if err == zklib.ErrNoNode {
				q.children = q.children[1:]
				continue
			} else if err != nil {
				return xlateError(err)
			}

			// Does the node still exist?
			data, stat, err := q.c.conn.Get(path)
			if err == zklib.ErrNoNode {
				q.children = q.children[1:]
				continue
			} else if err != nil {
				return xlateError(err)
			}

			if len(data) > 0 {
				err = json.Unmarshal(data, node)
			} else {
				err = client.ErrEmptyNode
			}

			node.SetVersion(stat)
			q.lockPath = lockPath
			return xlateError(err)
		}

		// if no nodes are available in memory, check zookeeper for any new nodes
		for {
			children, _, ch, err := q.c.conn.ChildrenW(q.path)
			if err != nil {
				return xlateError(err)
			}

			if len(children) > 0 {
				sort.Sort(sortqueue(children))
				q.children = children
				break
			}

			ev := <-ch
			if ev.Err != nil {
				return xlateError(ev.Err)
			}
		}
	}
}

// Consume pops the inflight node off the queue
func (q *Queue) Consume() error {
	if !q.HasLock() {
		return ErrNotLocked
	}

	if q.c.conn == nil {
		// TODO: race condition exists
		return fmt.Errorf("lost connection")
	}

	// Delete the parent of the lockPath
	if err := q.c.conn.Delete(path.Dir(q.lockPath), -1); err != nil {
		return xlateError(err)
	}

	q.lockPath = ""
	q.children = q.children[1:]
	return nil
}

// HasLock returns true when the Queue instance owns the lock
func (q *Queue) HasLock() bool {
	if q.lockPath == "" {
		return false
	}

	ok, _ := q.c.Exists(q.lockPath)
	if !ok {
		// this instance lost its lock, so reset the lockPath
		q.lockPath = ""
	}

	return ok
}

// Current reveals the current inflight node on the queue
func (q *Queue) Current(node client.Node) error {
	if !q.HasLock() {
		// if the instance doesn't own the lock, find the node with the current
		// lock
		for i := 0; i < 3; i++ {
			if q.c.conn == nil {
				// TODO: race condition exists
				return fmt.Errorf("lost connection")
			}

			// verify the first node in the queue has a lock to indicate that
			// it is inflight
			for _, p := range q.children {
				path := join(q.path, p)

				children, _, err := q.c.conn.Children(path)
				if err == zklib.ErrNoNode {
					q.children = q.children[1:]
					continue
				} else if err != nil {
					return xlateError(err)
				}

				if len(children) > 0 {
					data, stat, err := q.c.conn.Get(path)
					if err != nil {
						return xlateError(err)
					}

					err = json.Unmarshal(data, node)
					node.SetVersion(stat)
					return xlateError(err)
				}
				return ErrNotLocked
			}

			// grab the current list of nodes and try again
			children, _, err := q.c.conn.Children(q.path)
			if err != nil {
				return xlateError(err)
			}
			sort.Sort(sortqueue(children))
			q.children = children
		}
		return ErrEmptyQueue
	}

	// this instance of Queue owns the lock, so return the node where this lock
	// is held
	if q.c.conn == nil {
		// TODO: race condition exists
		return fmt.Errorf("lost connection")
	}
	data, stat, err := q.c.conn.Get(path.Dir(q.lockPath))
	if err != nil {
		return xlateError(err)
	}
	err = json.Unmarshal(data, node)
	node.SetVersion(stat)
	return xlateError(err)
}

// Next reveals the next node to be dequeued
func (q *Queue) Next(node client.Node) error {
	for i := 0; i < 3; i++ {
		// find the first node in the queue that does not own a lock
		for _, p := range q.children {
			path := join(q.path, p)

			children, _, err := q.c.conn.Children(path)
			if err == zklib.ErrNoNode {
				q.children = q.children[1:]
				continue
			} else if err != nil {
				return xlateError(err)
			}

			if len(children) == 0 {
				data, stat, err := q.c.conn.Get(path)
				if err != nil {
					return xlateError(err)
				}

				err = json.Unmarshal(data, node)
				node.SetVersion(stat)
				return xlateError(err)
			}
		}

		// if all the nodes are stale, refresh the node list
		children, _, err := q.c.conn.Children(q.path)
		if err != nil {
			return xlateError(err)
		}
		sort.Sort(sortqueue(children))
		q.children = children
	}
	return ErrEmptyQueue
}
