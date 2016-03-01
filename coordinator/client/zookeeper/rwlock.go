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

package zookeeper

import (
	lpath "path"
	"strconv"
	"strings"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

// An implementation of Zookeeper's Shared Locks recipe at:
//   https://zookeeper.apache.org/doc/r3.4.5/recipes.html#Shared+Locks

const (
	// Root node for our RW locks
	rwLockRoot = "/rwlocks"
)

// RWLock is an implementation of a reader/writer mutual exclusion lock in zookeeper.
type RWLock struct {
	c           *Connection
	path        string
	acl         []zklib.ACL
	lockPath    string
	isShortPath bool
}

// NewRWLock creates a new reader/writer lock.
// path is relative to the connection's base path.
func (c *Connection) NewRWLock(path string) *RWLock {
	return &RWLock{
		c:           c,
		path:        path,
		acl:         zklib.WorldACL(zklib.PermAll),
		isShortPath: true,
	}
}

// newRWLock creates a new reader/writer lock.
// fullPath is the absolute path of the node to be locked.
func (c *Connection) newRWLock(fullPath string) *RWLock {
	return &RWLock{
		c:           c,
		path:        fullPath,
		acl:         zklib.WorldACL(zklib.PermAll),
		isShortPath: false,
	}
}

// RLock obtains a lock for reading.
func (l *RWLock) RLock() error {
	return xlateError(l.lock(false))
}

// Lock obtains a lock for writing.
func (l *RWLock) Lock() error {
	return xlateError(l.lock(true))
}

// Unlock unlocks the lock obtained by a call to Lock or RLock.
func (l *RWLock) Unlock() error {
	if l.lockPath == "" {
		return ErrNotLocked
	}

	if err := l.c.conn.Delete(l.lockPath, -1); err != nil {
		glog.Errorf("Could not delete lock file %s: %s", l.lockPath, err)
		return xlateError(err)
	}
	// DEBUG: KWW: Remove this line
	glog.Infof("KWW: Deleted lock file: %s", l.lockPath)

	l.lockPath = ""
	return nil
}

// lock implements the locking algorithm.
func (l *RWLock) lock(getWriteLock bool) error {
	if l.lockPath != "" {
		return ErrDeadlock
	}

	// Create our lock file (which will get assigned a sequence number to "save our place in line")
	var err error
	var parentPath string
	if parentPath, err = l.createLockFile(getWriteLock); err != nil {
		glog.Errorf("Could not create lock file: %s", err)
		return err
	}

	cleanUpLockFile := true
	defer func() {
		if cleanUpLockFile {
			l.Unlock()
		}
	}()

	for {
		// Get the other lock files (without setting the watch flag)
		var lockFiles []string
		if lockFiles, _, err = l.c.conn.Children(parentPath); err != nil {
			glog.Errorf("Could not retrieve other lock files from %s: %s", parentPath, err)
			return err
		}

		// See if someone else has a lock that prevents us from obtaining ours
		var blockingLockFile string
		if blockingLockFile, err = findBlockingLock(l.lockPath, lockFiles); err != nil {
			glog.Errorf("Could not search for existing lock: %s", err)
			return err
		}

		// If no blocking lock, then we can proceed
		if blockingLockFile == "" {
			// DEBUG: KWW: Remove this line
			glog.Infof("KWW: Obtained lock: %s", l.lockPath)
			cleanUpLockFile = false
			return nil
		}

		// Otherwise, watch our blocker. Once it's gone, go back and make sure no other lock
		// blocks us.
		// TODO: If we're blocking on highest seq, why do we need to go check again?
		var event <-chan zklib.Event
		// DEBUG: KWW: Remove this line
		glog.Infof("KWW: Blocking on %s...", blockingLockFile)
		if _, _, event, err = l.c.conn.GetW(lpath.Join(parentPath, blockingLockFile)); err != nil {
			if err == zklib.ErrNoNode {
				// It disappeared between the time we gathered the list of lock files and now
				continue
			} else {
				glog.Errorf("Could not watch lock file %s: %s", blockingLockFile, err)
				return err
			}
		}
		// TODO: Implement timeout here?
		select {
		case <-event:
			// DEBUG: KWW: Remove this line
			glog.Infof("KWW: Unblocked by %s", blockingLockFile)
			continue
		}
	}
}

// createLockFile creates a lock file, creating the parent directory if necessary.
func (l *RWLock) createLockFile(makeWriteLock bool) (string, error) {
	var path string
	var err error
	noData := []byte{}

	prefix := l.getPrefix(makeWriteLock)
	path, err = l.c.conn.CreateProtectedEphemeralSequential(prefix, noData, l.acl)
	if err == zklib.ErrNoNode {
		// Create parent node and try again
		parts := strings.Split(prefix, "/")
		pth := ""
		for _, p := range parts[1 : len(parts)-1] {
			pth += "/" + p
			// DEBUG: KWW: Remove this line
			glog.Infof("KWW: Creating dir: %s", pth)
			_, err := l.c.conn.Create(pth, noData, 0, l.acl)
			if err != nil && err != zklib.ErrNodeExists {
				glog.Errorf("Could not create lock parent dir %s: %s", pth, err)
				return "", err
			}
		}
		path, err = l.c.conn.CreateProtectedEphemeralSequential(prefix, noData, l.acl)
	}

	if err != nil {
		glog.Errorf("Could not create lock file (prefix=%s): %s", prefix, err)
		return "", err
	}
	l.lockPath = path
	// DEBUG: KWW: Remove this line
	glog.Infof("KWW: Created lock file: %s", l.lockPath)

	return lpath.Dir(l.lockPath), nil
}

// getPrefix builds the prefix for the lock file to be created.
func (l *RWLock) getPrefix(isWriteLock bool) string {
	var lockFilePrefix string
	if isWriteLock {
		lockFilePrefix = "write-"
	} else {
		lockFilePrefix = "read-"
	}
	var fullPath string
	if l.isShortPath {
		fullPath = lpath.Join(rwLockRoot, l.c.basePath, l.path, lockFilePrefix)
	} else {
		fullPath = lpath.Join(rwLockRoot, l.path, lockFilePrefix)
	}
	return fullPath
}

// getLockInfo breaks a lock file path down into important components.
func getLockInfo(lockPath string) (ok bool, isWriteLock bool, seq int) {
	seq = -1

	// Lock files should have three parts: ZK session ID, read or write, sequence number
	parts := strings.Split(lpath.Base(lockPath), "-")
	if len(parts) != 3 {
		// Some other node that is not a lock file
		// DEBUG: KWW: Convert to V(3)
		glog.Infof("Skipping non-lock file: %s", lockPath)
		return
	} else {
		lockType := parts[len(parts)-2]
		if lockType == "write" {
			isWriteLock = true
		} else if lockType == "read" {
			isWriteLock = false
		} else {
			// Some other node that happens to have three parts--ignore it
			// DEBUG: KWW: Convert to V(3)
			glog.Infof("Skipping non-lock file (not read or write lock): %s", lockPath)
			return
		}

		seqNum := parts[len(parts)-1]
		var err error
		if seq, err = strconv.Atoi(seqNum); err != nil {
			// DEBUG: KWW: Convert to V(3)
			glog.Infof("Skipping lock file with invalid sequence number: %s: %s", lockPath, err)
			return
		}
	}

	ok = true
	return
}

// processChildren examines all the lock files and returns the lock blocking us, if there is one.
func findBlockingLock(myLockPath string, lockPaths []string) (string, error) {
	// Here's where we diverge from the standard recipe. We've flattened the directory structure (in
	// getPrefix()) so all the lock files for a category (the root directory of the item being locked)
	// are under one node. We can compare the lock files' names to determine which ones should affect
	// each other (i.e., which have a parent-child relationship, vs. which are protecting different
	// directories).

	var err error
	ok, myIsWrite, mySeq := getLockInfo(myLockPath)
	if !ok {
		glog.Errorf("Could not evaluate new lock file %s", myLockPath)
		return "", err
	}

	var blockingLock string
	blockingSeq := -1
	for _, otherPath := range lockPaths {
		ok, otherIsWrite, otherSeq := getLockInfo(otherPath)
		if ok {
			// If we want a write lock, we have to examine all other locks.
			// If we want a read lock, only examine write locks.
			if myIsWrite || otherIsWrite {
				// As a side effect, this comparison avoids us comparing against ourself
				if otherSeq < mySeq && otherSeq > blockingSeq {
					blockingLock = otherPath
					blockingSeq = otherSeq
				}
			}
		}
	}
	return blockingLock, nil
}
