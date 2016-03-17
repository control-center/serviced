// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"fmt"
	"time"
)

type DFSLocker interface {
	// Lock blocks until it can acquire the lock.
	// opName is the name of the operation you are about to perform.
	Lock(opName string)
	// LockWithTimeout blocks up to the timeout specified, then returns.
	// opName is the name of the operation you are about to perform.
	LockWithTimeout(opName string, timeout time.Duration) error
	// Unlock releases the lock.
	// It is a run-time error (panic) if the mutex is not locked.
	Unlock()
}

type ErrDfsBusy struct {
	blocker string
}

func (e ErrDfsBusy) Error() string {
	return fmt.Sprintf("DFS is locked for %s, try again later.", e.blocker)
}

func (dfs *DistributedFilesystem) Lock(opName string) {
	dfs.locker.Lock(opName)
}

func (dfs *DistributedFilesystem) LockWithTimeout(opName string, timeout time.Duration) error {
	if gotLock, blockingOp := dfs.locker.LockWithTimeout(opName, timeout); !gotLock {
		return ErrDfsBusy{blockingOp}
	}
	return nil
}

func (dfs *DistributedFilesystem) Unlock() {
	dfs.locker.Unlock()
}
