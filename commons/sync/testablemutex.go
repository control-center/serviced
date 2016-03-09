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

// A TestableLocker represents an object that can be locked and unlocked.
type TestableLocker interface {
	Lock(string)
	TestAndLock(name string) (bool, string)
	Unlock()
}

// A TestableMutex is a mutual exclusion lock that provides an additional method for
// requesting a lock without blocking.
type TestableMutex struct {
	ch     chan struct{}
	holder string
}

// NewTestableMutex initializes a new TestableMutex. Its initial state is unlocked.
func NewTestableMutex() *TestableMutex {
	m := &TestableMutex{
		ch: make(chan struct{}, 1),
	}
	m.ch <- struct{}{} // Unlocked
	return m
}

// Lock locks the TestableMutex.
// If the mutex is already in use, the calling goroutine
// blocks until the mutex is available.
func (m *TestableMutex) Lock(name string) {
	select {
	case <-m.ch:
		m.holder = name
	}
}

// TestAndLock attempts to lock the TestableMutex but returns immediately.
// The bool returned indicates whether the lock was obtained.
// The string is the name of the current holder of the lock.
func (m *TestableMutex) TestAndLock(name string) (gotLock bool, holder string) {
	select {
	case <-m.ch:
		m.holder = name
		return true, m.holder
	default:
		return false, m.holder
	}
}

// Unlock unlocks the TestableMutex.
// It is a run-time error if the mutex is not locked on entry to Unlock.
//
// A locked TestableMutex is not associated with a particular goroutine.
// It is allowed for one goroutine to lock a TestableMutex and then
// arrange for another goroutine to unlock it.
func (m *TestableMutex) Unlock() {
	m.holder = ""
	select {
	case m.ch <- struct{}{}:
	default:
		panic("unlock of unlocked testable mutex")
	}
}
