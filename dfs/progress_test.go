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

// +build unit

package dfs_test

import (
	"time"

	"github.com/control-center/serviced/dfs"
	. "gopkg.in/check.v1"
)

var (
	oneSecondDuration  = time.Duration(1 * time.Second)
	fiveSecondDuration = time.Duration(5 * time.Second)
)

func (s *DFSTestSuite) TestProgressCounter_ShouldNotLogIfIntervalHasNotPassed(c *C) {
	logged := false

	counter := dfs.NewProgressCounterWithClock(3, getClock(oneSecondDuration))
	counter.Log = func() { logged = true }
	counter.Write([]byte("Data"))

	c.Assert(logged, Equals, false)
}

func (s *DFSTestSuite) TestProgressCounter_ShouldLogIfIntervalPassed(c *C) {
	logged := false

	counter := dfs.NewProgressCounterWithClock(3, getClock(fiveSecondDuration))
	counter.Log = func() { logged = true }
	counter.Write([]byte("Data"))

	c.Assert(logged, Equals, true)
}

func (s *DFSTestSuite) TestProgressCounter_ShouldLogIfIntervalIsZero(c *C) {
	timesLogged := 0

	counter := dfs.NewProgressCounterWithClock(0, getClock(oneSecondDuration))
	counter.Log = func() { timesLogged += 1 }
	counter.Write([]byte("Data"))
	counter.Write([]byte("Data"))
	counter.Write([]byte("Data"))

	c.Assert(timesLogged, Equals, 3)
}

func (s *DFSTestSuite) TestProgressCounter_ShouldLogOnceIfSecondWriteAfterIntervalButNotFirst(c *C) {
	timesLogged := 0

	counter := dfs.NewProgressCounterWithClock(7, getClock(fiveSecondDuration))
	counter.Log = func() { timesLogged += 1 }
	counter.Write([]byte("Data"))
	counter.Write([]byte("Data"))

	c.Assert(timesLogged, Equals, 1)
}

func (s *DFSTestSuite) TestProgressCounter_ShouldNotLogIfMultipleWritesBeforeInterval(c *C) {
	logged := false

	counter := dfs.NewProgressCounterWithClock(7, getClock(oneSecondDuration))
	counter.Log = func() { logged = true }
	counter.Write([]byte("Data"))
	counter.Write([]byte("Data"))

	c.Assert(logged, Equals, false)
}

func (s *DFSTestSuite) TestProgressCounter_TotalShouldBeZeroIfNoWrite(c *C) {
	counter := dfs.NewProgressCounterWithClock(3, getClock(oneSecondDuration))
	c.Assert(counter.Total, Equals, uint64(0))
}

func (s *DFSTestSuite) TestProgressCounter_CountsBytesCorrectly(c *C) {
	counter := dfs.NewProgressCounterWithClock(3, getClock(oneSecondDuration))
	counter.Write([]byte("Two bytes meet."))
	counter.Write([]byte("The first byte asks, \"Are you ill?\""))
	counter.Write([]byte("The second byte replies, \"No, just feeling a little bit off.\""))

	c.Assert(counter.Total, Equals, uint64(111))
}

func (s *DFSTestSuite) TestProgressCounter_ShouldIncrementCountIfNoLoggerHasBeenSet(c *C) {
	counter := dfs.NewProgressCounterWithClock(3, getClock(fiveSecondDuration))
	counter.Write([]byte("Data"))
	c.Assert(counter.Total, Equals, uint64(4))
}

func getClock(duration time.Duration) dfs.Clock {
	now := time.Date(2017, 1, 7, 7, 34, 0, 0, time.UTC)
	return &mockClock{now: now, duration: duration}
}

// mockClock will increment the current time each time Now or Since is called
// by the specified duration.  The current time starts at 1/7/2017 7:34:00.
type mockClock struct {
	now      time.Time
	duration time.Duration
}

func (c *mockClock) Now() time.Time {
	now := c.now
	c.tick()
	return now
}

func (c *mockClock) Since(t time.Time) time.Duration {
	since := c.now.Sub(t)
	c.tick()
	return since
}

func (c *mockClock) tick() {
	c.now = c.now.Add(c.duration)
}
