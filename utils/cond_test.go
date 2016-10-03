// Copyright 2015 The Serviced Authors.
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

// +build unit

package utils_test

import (
	"sync"
	"time"

	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

type ChannelCondSuite struct{}

var (
	_ = Suite(&ChannelCondSuite{})
)

func assertClosedWithinTimeout(c *C, timeout time.Duration, chans ...<-chan struct{}) {
	for _, ch := range chans {
		select {
		case <-ch:
		case <-time.After(timeout):
			c.Fatalf("A channel was not closed within the timeout")
			return
		}
	}
}

func assertAllOpen(c *C, chans ...<-chan struct{}) {
	var wg sync.WaitGroup
	for _, ch := range chans {
		wg.Add(1)
		go func(ch <-chan struct{}) {
			defer wg.Done()
			select {
			case <-ch:
				c.Fatalf("A channel was closed")
			case <-time.After(5 * time.Millisecond):

			}
		}(ch)
	}
	wg.Wait()
}

func (s *ChannelCondSuite) TestChannelCond(c *C) {
	cond := utils.NewChannelCond()

	// A function to create a bunch of channels that will close when the
	// channel returned by cond.Wait() closes, which we can use to verify
	// broadcast
	makechans := func(n int) []<-chan struct{} {
		donechans := make([]<-chan struct{}, n)
		for i := 0; i < n; i++ {
			done := make(chan struct{})
			ch := cond.Wait()
			go func(ch <-chan struct{}, done chan struct{}) {
				<-ch
				close(done)
			}(ch, done)
			donechans[i] = done
		}
		return donechans
	}

	// Basic test with 5 listeners
	donechans := makechans(5)
	assertAllOpen(c, donechans...)
	cond.Broadcast()
	assertClosedWithinTimeout(c, time.Second, donechans...)

	// Test another 5 listeners with another broadcast, to verify subsequent
	// conditions
	donechans2 := makechans(5)
	assertAllOpen(c, donechans2...)
	cond.Broadcast()
	assertClosedWithinTimeout(c, time.Second, donechans2...)
}
