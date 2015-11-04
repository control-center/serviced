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

package queue

import (
	"math"
	"sync"
	"sync/atomic"
	"testing"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MySuite struct{}

var _ = Suite(&MySuite{})

func (s *MySuite) TestNewQueue(c *C) {
	var q Queue
	_, err := NewChannelQueue(-1)
	c.Assert(err, NotNil)

	_, err = NewChannelQueue(0)
	c.Assert(err, NotNil)

	q, err = NewChannelQueue(1)
	c.Assert(q.Capacity(), Equals, int32(1))
	c.Assert(q.Size(), Equals, int32(0))

	q, err = NewChannelQueue(math.MaxInt8)

	c.Assert(q.Capacity(), Equals, int32(math.MaxInt8))

}
func (s *MySuite) TestPutTake(c *C) {
	cap := 38
	q, _ := NewChannelQueue(cap)

	for i := 0; i < cap; i++ {
		q.Put(i)
		c.Assert(q.Size(), Equals, int32(min(i+1, cap)))
	}

	for i := 0; i < cap; i++ {
		x := q.Take().(int)
		c.Assert(x, Equals, i)
	}
}

func (s *MySuite) TestOffer(c *C) {
	cap := 32
	q, _ := NewChannelQueue(cap)
	if q == nil {
		c.Fail()
	}
	for i := 0; i < cap*2; i++ {
		if q.Offer(i) && i >= cap {
			c.Fail()
		}
		c.Assert(q.Size(), Equals, int32(min(i+1, cap)))
	}
	for i := 0; i < cap*2; i++ {
		x, found := q.Poll()
		if !found && i < cap {
			c.Errorf("expected to find %d", i)
		}
		if found {
			c.Assert(i, Equals, x)
		}
	}
}

func (s *MySuite) TestConcurrent(c *C) {
	cap := 32
	q, _ := NewChannelQueue(cap)
	if q == nil {
		c.Fail()
	}
	wg := sync.WaitGroup{}
	for i := 0; i < cap*2; i++ {
		wg.Add(1)
		val := i
		go func() {
			q.Offer(val)
			wg.Done()
		}()
	}
	wg.Wait()
	c.Assert(q.Size(), Equals, int32(cap))

	numberFound := int32(0)
	for i := 0; i < cap*2; i++ {
		wg.Add(1)
		go func() {
			if _, found := q.Poll(); found {
				atomic.AddInt32(&numberFound, 1)
			}
			wg.Done()
		}()
	}

	wg.Wait()
	c.Assert(q.Size(), Equals, int32(0))
	c.Assert(numberFound, Equals, int32(cap))
}

func min(x, y int) int32 {
	if x < y {
		return int32(x)
	}
	return int32(y)
}
