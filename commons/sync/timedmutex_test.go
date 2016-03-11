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

// +build unit

package sync

import (
	"fmt"
	"testing"
	"time"
)

type LockWithTimeoutResponse struct {
	ok    bool
	owner string
}

func TestTimedMutex_Lock(t *testing.T) {
	name1 := "lock 1"
	name2 := "lock 2"

	locker := NewTimedMutex()

	// lock the mutex
	lock1Response := make(chan struct{})
	go func() {
		locker.Lock(name1)
		lock1Response <- struct{}{}
	}()
	select {
	case <-lock1Response:
		t.Logf("(ok) lock 1 acquired")
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("timeout while acquiring lock 1")
	}

	// test that a second lock blocks
	lock2Response := make(chan struct{})
	go func() {
		locker.Lock(name2)
		t.Logf("--- got lock 2, sending event")
		lock2Response <- struct{}{}
	}()
	select {
	case <-lock2Response:
		t.Fatalf("lock 2 did not block as expected")
	case <-time.After(time.Second):
		t.Logf("(ok) lock 2 is blocking")
	}

	// unlock
	unlock1Response := make(chan struct{})
	go func() {
		t.Logf("--- unlocking 1")
		locker.Unlock()
		unlock1Response <- struct{}{}
	}()
	select {
	case <-unlock1Response:
		t.Logf("(ok) unlocked 1")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking 1")
	}

	// test that the second lock unblocked
	select {
	case <-lock2Response:
		t.Logf("(ok) lock 2 unblocked")
	case <-time.After(time.Second * 3):
		t.Errorf("timeout waiting for lock 2 to unblock")
	}

	// check if the second lock can unlock
	unlock2Response := make(chan struct{})
	go func() {
		locker.Unlock()
		unlock2Response <- struct{}{}
	}()
	select {
	case <-unlock2Response:
		t.Logf("(ok) unlocked 2")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking 2")
	}
}

func TestTimedMutex_LockWithTimeoutBlocks(t *testing.T) {
	name1 := "lock 1"
	name2 := "lock 2"

	locker := NewTimedMutex()

	// lock the mutex
	lock1Response := make(chan struct{})
	go func() {
		locker.Lock(name1)
		lock1Response <- struct{}{}
	}()
	select {
	case <-lock1Response:
		t.Logf("(ok) lock 1 acquired")
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("timeout while acquiring lock 1")
	}

	// test that lock-with-timeout blocks
	lock2Response := make(chan struct{})
	go func() {
		locker.LockWithTimeout(name2, time.Minute)
		t.Logf("--- got lock 2, sending event")
		lock2Response <- struct{}{}
	}()
	select {
	case <-lock2Response:
		t.Fatalf("lock 2 did not block as expected")
	case <-time.After(2 * time.Second):
		t.Logf("(ok) lock 2 is blocking")
	}

	// unlock
	unlock1Response := make(chan struct{})
	go func() {
		t.Logf("--- unlocking 1")
		locker.Unlock()
		unlock1Response <- struct{}{}
	}()
	select {
	case <-unlock1Response:
		t.Logf("(ok) unlocked 1")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking 1")
	}

	// test that the second lock unblocked
	select {
	case <-lock2Response:
		t.Logf("(ok) lock 2 unblocked")
	case <-time.After(time.Second * 3):
		t.Errorf("timeout waiting for lock 2 to unblock")
	}

	// check if the second lock can unlock
	unlock2Response := make(chan struct{})
	go func() {
		locker.Unlock()
		unlock2Response <- struct{}{}
	}()
	select {
	case <-unlock2Response:
		t.Logf("(ok) unlocked 2")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking 2")
	}
}

func TestTimedMutex_LockWithTimeoutTimesOut(t *testing.T) {
	name1 := "lock 1"
	name2 := "lock 2"

	locker := NewTimedMutex()

	// lock the mutex
	lock1Response := make(chan struct{})
	go func() {
		locker.Lock(name1)
		lock1Response <- struct{}{}
	}()
	select {
	case <-lock1Response:
		t.Logf("(ok) lock 1 acquired")
	case <-time.After(250 * time.Millisecond):
		t.Fatalf("timeout while acquiring lock 1")
	}

	// test that lock-with-timeout times out
	lock2Response := make(chan struct{})
	go func() {
		locker.LockWithTimeout(name2, time.Second)
		t.Logf("--- got lock 2, sending event")
		lock2Response <- struct{}{}
	}()
	select {
	case <-lock2Response:
		t.Logf("(ok) lock 2 timed out")
	case <-time.After(2 * time.Second):
		t.Errorf("lock 2 did not time out")
	}

	// unlock
	unlock1Response := make(chan struct{})
	go func() {
		t.Logf("--- unlocking 1")
		locker.Unlock()
		unlock1Response <- struct{}{}
	}()
	select {
	case <-unlock1Response:
		t.Logf("(ok) unlocked 1")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking 1")
	}
}

func TestTimedMutex_DoubleUnlock(t *testing.T) {
	name1 := "lock 1"

	locker := NewTimedMutex()

	// lock the mutex
	lock1Response := make(chan struct{})
	go func() {
		locker.Lock(name1)
		lock1Response <- struct{}{}
	}()
	select {
	case <-lock1Response:
		t.Logf("(ok) lock 1 acquired")
	case <-time.After(time.Second):
		t.Fatalf("timeout while acquiring lock 1")
	}

	// unlock
	unlock1Response := make(chan struct{})
	go func() {
		locker.Unlock()
		unlock1Response <- struct{}{}
	}()
	select {
	case <-unlock1Response:
		t.Logf("(ok) unlocked 1")
	case <-time.After(time.Second):
		t.Errorf("timeout while unlocking 1")
	}

	// second unlock, should panic
	unlock2Response := make(chan bool)
	go func() {
		defer func() {
			if r := recover(); r != nil {
				t.Logf("(ok) caught panic")
				unlock2Response <- true
			} else {
				unlock2Response <- false
			}
		}()
		locker.Unlock()
	}()
	select {
	case response2 := <-unlock2Response:
		if !response2 {
			t.Errorf("no panic from unlock 2")
		} else {
			// Panicked, as expected (logged by goroutine)
		}
	case <-time.After(time.Second):
		t.Errorf("timeout while unlocking 2")
	}
}

// Make sure multiple write locks are released one at a time
func TestTimedMutex_MultipleLockers(t *testing.T) {
	numGoroutines := 4

	// setup
	locker := NewTimedMutex()

	// data to be managed by the lock
	var listA []int
	var listB []int
	var listC []int

	// get a lock so goroutines block
	lockmainResponse := make(chan struct{})
	go func() {
		locker.Lock("lock main")
		lockmainResponse <- struct{}{}
	}()
	select {
	case <-lockmainResponse:
		// Acquired, as expected
	case <-time.After(time.Second):
		t.Fatalf("timeout while acquiring lock main")
	}

	// spin off several goroutines
	ready := make(chan int)
	done := make(chan int)
	for i := 1; i <= numGoroutines; i++ {
		go func(myId int) {
			defer func() { done <- myId }() // Signal that I am done
			// Signal that I am ready
			ready <- myId
			// Lock/block (using both lock methods)
			if myId%2 == 0 {
				locker.LockWithTimeout(fmt.Sprintf("goroutine %d", myId), 2*time.Minute)
			} else {
				locker.Lock(fmt.Sprintf("goroutine %d", myId))
			}
			// Update the data
			time.Sleep(250 * time.Millisecond)
			listA = append(listA, myId)
			time.Sleep(250 * time.Millisecond)
			listB = append(listB, myId)
			time.Sleep(250 * time.Millisecond)
			listC = append(listC, myId)
			// Unlock
			locker.Unlock()
		}(i)
	}

	// wait until all goroutines are ready
	numEvents := 0
	timeout := time.NewTimer(2 * time.Second)
	for numEvents < numGoroutines {
		select {
		case <-ready:
			numEvents++
		case <-timeout.C:
			t.Fatalf("goroutines never got ready")
		}
	}
	time.Sleep(250 * time.Microsecond) // event is sent before Lock() is called

	// unlock
	unlockmainResponse := make(chan struct{})
	go func() {
		locker.Unlock()
		unlockmainResponse <- struct{}{}
	}()
	select {
	case <-unlockmainResponse:
		t.Logf("main unlocked")
	case <-time.After(time.Second):
		t.Fatalf("timeout while unlocking main")
	}

	// wait for the goroutines to finish (they should proceed one at a time as another unlocks)
	numEvents = 0
	timeout = time.NewTimer((time.Duration(numGoroutines) * time.Second) + (2 * time.Second))
	for numEvents < numGoroutines {
		select {
		case ev := <-done:
			t.Logf("goroutine %d is done", ev)
			numEvents++
		case <-timeout.C:
			t.Fatalf("timed out waiting for goroutines to update the data")
		}
	}

	// verify all the data lists are the same
	t.Logf("data lists: A=%v, B=%v, C=%v", listA, listB, listC)
	for i := 0; i < len(listA); i++ {
		if listA[i] != listB[i] || listB[i] != listC[i] {
			t.Fatalf("lists do not match at element %d", i)
		}
	}
}
