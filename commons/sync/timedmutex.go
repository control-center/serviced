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
// limitations under the License.package rpcutils

package sync

import (
	"sync/atomic"
	"time"
)

// A TimedLocker represents an object that can be locked and unlocked.
type TimedLocker interface {
	Lock(string)
	LockWithTimeout(name string, timeout time.Duration) (gotLock bool, holder string)
	Unlock()
}

// A TimedMutex is a mutual exclusion lock that provides an additional Lock method
// that will time out if it cannot acquire the lock.
type TimedMutex struct {
	ch     chan struct{}
	holder atomic.Value
}

// NewTimedMutex initializes a new TimedMutex. Its initial state is unlocked.
func NewTimedMutex() *TimedMutex {
	m := &TimedMutex{
		ch: make(chan struct{}, 1),
	}
	m.ch <- struct{}{} // Unlocked
	return m
}

// Lock locks the TimedMutex.
// name is a string to identify you to callers who fail to acquire the lock.
//
// If the mutex is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *TimedMutex) Lock(name string) {
	select {
	case <-m.ch:
		m.holder.Store(name)
	}
}

// LockWithTimeout attempts to lock the TimedMutex but returns if it cannot acquire
// the lock within the time specified.
// name is a string to identify you to callers who fail to acquire the lock.
//
// The bool returned indicates whether you acquired the lock.
// The string is the name of the current holder of the lock.
func (m *TimedMutex) LockWithTimeout(name string, timeout time.Duration) (gotLock bool, holder string) {
	select {
	case <-m.ch:
		m.holder.Store(name)
		return true, m.holder.Load().(string)
	case <-time.After(timeout):
		return false, m.holder.Load().(string)
	}
}

// Unlock unlocks the TimedMutex.
// It is a run-time error if the mutex is not locked on entry to Unlock.
//
// A locked TimedMutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a TimedMutex and then
// arrange for another goroutine to unlock it.
func (m *TimedMutex) Unlock() {
	m.holder.Store("")
	select {
	case m.ch <- struct{}{}:
	default:
		panic("unlock of unlocked timed-mutex")
	}
}
