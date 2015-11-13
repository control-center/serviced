// Copyright 2015 The Serviced Authors.
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

// +build unit

package pool

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var (
	_                = Suite(&MySuite{})
	factoryCount     = int32(0)
	IntentionalError = errors.New("intentional error")
)

func Factory(maxCreate int) ItemFactory {
	factoryCount = 0
	return func() (interface{}, error) {
		val := int(factoryCount)
		x := atomic.AddInt32(&factoryCount, 1)
		if x > int32(maxCreate) {
			return nil, fmt.Errorf("Tried to create %d items, max is %d", x, maxCreate)
		}
		return val, nil
	}

}

func ErrFactory() (interface{}, error) {
	return nil, IntentionalError
}

func (s *MySuite) TestBorrowReturnRemove(c *C) {

	capacity := 1

	//Allow creation of one more item than capacity
	p, err := NewPool(capacity, Factory(capacity+1))
	c.Assert(err, IsNil)

	items := []*Item{}
	for i := 0; i < capacity; i++ {
		x, err := p.Borrow()
		c.Assert(int(factoryCount), Equals, i+1)
		c.Assert(err, IsNil)
		c.Assert(x, NotNil)
		c.Assert(x.Item, Equals, i)
		items = append(items, x)
		c.Assert(p.Idle(), Equals, 0) //items are lazily created
		c.Assert(p.Borrowed(), Equals, i+1)
	}
	c.Assert(p.Idle(), Equals, 0)
	c.Assert(p.Borrowed(), Equals, capacity)

	_, err = p.Borrow()
	c.Assert(err, Equals, ErrItemUnavailable)

	for idx, item := range items {
		p.Return(item)
		c.Assert(p.Idle(), Equals, idx+1)
		c.Assert(p.Borrowed(), Equals, capacity-idx-1)
	}

	c.Assert(p.Idle(), Equals, capacity)
	c.Assert(p.Borrowed(), Equals, 0)

	for i := 0; i < capacity; i++ {
		item, err := p.Borrow()
		c.Assert(err, IsNil)
		c.Assert(item, NotNil)
		p.Remove(item)
		c.Assert(p.Idle(), Equals, capacity-i)
		c.Assert(p.Borrowed(), Equals, 0)
	}

	// Test that one more than capacity is available
	item, err := p.Borrow()
	c.Assert(item.Item, Equals, capacity)

	c.Assert(p.Idle(), Equals, 0)
	item, err = p.Borrow()
	c.Assert(item, IsNil)
	c.Assert(err, Equals, ErrItemUnavailable)
}

func (s *MySuite) TestWaitOnRemove(c *C) {
	capacity := 1

	//Allow creation of one more item than capacity
	p, err := NewPool(capacity, Factory(capacity+1))
	c.Assert(err, IsNil)

	item, err := p.Borrow()
	c.Assert(err, IsNil)
	c.Assert(item, NotNil)

	startedWG := sync.WaitGroup{}
	wg := sync.WaitGroup{}
	wg.Add(1)
	startedWG.Add(1)
	var newItem *Item
	go func() {
		defer wg.Done()
		startedWG.Done()
		newItem, err = p.BorrowWait(3 * time.Second)
		c.Assert(err, IsNil)
		//second item created should be int 1
		c.Assert(newItem.Item, Equals, 1)
	}()
	startedWG.Wait()
	time.Sleep(250 * time.Millisecond)
	p.Remove(item)
	wg.Wait()
	c.Assert(item, Not(Equals), newItem)
}

func (s *MySuite) TestWait(c *C) {

	waitTime := 250 * time.Millisecond
	capacity := 1
	p, err := NewPool(capacity, Factory(capacity))
	c.Assert(err, IsNil)

	item, err := p.Borrow()
	start := time.Now()
	x, err := p.BorrowWait(waitTime)
	c.Assert(x, IsNil)
	c.Assert(err, Equals, ErrItemUnavailable)
	elapsed := time.Now().Sub(start)
	if elapsed < waitTime {
		c.Errorf("elapsed wait less than %s", waitTime)
	}

	go func() {
		time.Sleep(waitTime)
		p.Return(item)
	}()

	item, err = p.BorrowWait(waitTime * 2)
	c.Assert(err, IsNil)
	c.Assert(item, NotNil)
	if elapsed < waitTime {
		c.Errorf("elapsed wait less than %s", waitTime)
	}

}

func (s *MySuite) TestConcurrent(c *C) {

	capacity := 100

	p, err := NewPool(capacity, Factory(capacity))
	c.Assert(err, IsNil)

	items := []*Item{}
	itemLock := sync.Mutex{}
	found := int32(0)
	notAVail := int32(0)
	wg := sync.WaitGroup{}
	start := make(chan struct{})
	startWG := sync.WaitGroup{}

	for i := 0; i < capacity*2; i++ {
		wg.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			<-start
			if x, err := p.Borrow(); err == ErrItemUnavailable {
				atomic.AddInt32(&notAVail, 1)
			} else if err != nil {
				c.Errorf("Unexpected error %v", err)
			} else if x != nil {
				atomic.AddInt32(&found, 1)
				itemLock.Lock()
				items = append(items, x)
				itemLock.Unlock()
			}
			wg.Done()
		}()
	}
	startWG.Wait() //wait for the go routines to be ready
	close(start)   //start the go routines
	wg.Wait()      //wait for go routines to finish

	c.Assert(int(found), Equals, capacity)
	//tried to get 2*capacity co not found should equal capacity
	c.Assert(int(notAVail), Equals, capacity)

	c.Assert(p.Idle(), Equals, 0)
	c.Assert(p.Borrowed(), Equals, capacity)

	c.Assert(len(items), Equals, capacity)

	start = make(chan struct{})
	for idx := range items {
		item := items[idx]
		wg.Add(1)
		startWG.Add(1)

		go func() {
			startWG.Done()
			<-start
			err := p.Return(item)
			c.Assert(err, IsNil)
			wg.Done()
		}()
	}
	startWG.Wait()
	close(start)
	wg.Wait()

	c.Assert(p.Idle(), Equals, capacity)
	c.Assert(p.Borrowed(), Equals, 0)

	found = int32(0)
	notAVail = int32(0)
	start = make(chan struct{})

	//mix borrow and returns
	for i := 0; i < capacity*2; i++ {
		wg.Add(1)
		startWG.Add(1)
		go func() {
			startWG.Done()
			<-start
			if x, err := p.BorrowWait(5 * time.Second); err == ErrItemUnavailable {
				atomic.AddInt32(&notAVail, 1)
			} else if err != nil {
				c.Errorf("Unexpected error %v", err)
			} else if x != nil {
				atomic.AddInt32(&found, 1)
				time.Sleep(500 * time.Millisecond)
				p.Return(x)
			}
			wg.Done()
		}()
	}
	startWG.Wait() //wait for the go routines to be ready
	close(start)   //start the go routines
	wg.Wait()      //wait for go routines to finish
	c.Assert(int(found), Equals, capacity*2)
	//tried to get 2*capacity co not found should equal capacity
	c.Assert(int(notAVail), Equals, 0)
	c.Assert(p.Idle(), Equals, capacity)
	c.Assert(p.Borrowed(), Equals, 0)
}

func (s *MySuite) TestBorrowFactoryReturnsErr(c *C) {
	p, _ := NewPool(1, ErrFactory)
	x, err := p.Borrow()
	c.Assert(err, Equals, IntentionalError)
	c.Assert(x, IsNil)
}
