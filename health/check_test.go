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

package health_test

import (
	"time"

	. "github.com/control-center/serviced/health"
	. "gopkg.in/check.v1"
)

var _ = Suite(&HealthCheckTestSuite{})

type HealthCheckTestSuite struct{}

func (s *HealthCheckTestSuite) TestExpires(c *C) {
	// Verify the expiration duration
	check := HealthCheck{
		Script:    "echo testscript",
		Timeout:   time.Second,
		Interval:  2 * time.Second,
		Tolerance: 0,
	}
	expected := time.Duration(DefaultTolerance) * (check.Timeout + check.Interval)
	c.Assert(check.Expires(), Equals, expected)
	check.Tolerance = 100
	expected = time.Duration(check.Tolerance) * (check.Timeout + check.Interval)
	c.Assert(check.Expires(), Equals, expected)
}

func (s *HealthCheckTestSuite) TestNotRunning(c *C) {
	// Verify the health status is not running
	check := HealthCheck{
		Script:    "echo testscript",
		Timeout:   time.Second,
		Interval:  2 * time.Second,
		Tolerance: 0,
	}
	stat := check.NotRunning()
	c.Check(stat.Status, Equals, NotRunning)
	c.Check(stat.Output, DeepEquals, []byte{})
	c.Check(stat.Err, IsNil)
	c.Check(stat.StartedAt.IsZero(), Equals, false)
	c.Check(stat.Duration, Equals, time.Duration(0))
}

func (s *HealthCheckTestSuite) TestUnknown(c *C) {
	// Verify the health status is unknown
	check := HealthCheck{
		Script:    "echo testscript",
		Timeout:   time.Second,
		Interval:  2 * time.Second,
		Tolerance: 0,
	}
	stat := check.Unknown()
	c.Check(stat.Status, Equals, Unknown)
	c.Check(stat.Output, DeepEquals, []byte{})
	c.Check(stat.Err, IsNil)
	c.Check(stat.StartedAt.IsZero(), Equals, false)
	c.Check(stat.Duration, Equals, time.Duration(0))
}

func (s *HealthCheckTestSuite) TestRun_Passed(c *C) {
	// Verify a passing health check
	check := HealthCheck{
		Script:    "echo -n testscript",
		Timeout:   time.Second,
		Interval:  time.Second,
		Tolerance: 0,
	}
	stat := check.Run()
	c.Check(stat.Status, Equals, OK)
	c.Check(stat.Output, DeepEquals, []byte("testscript"))
	c.Check(stat.Err, IsNil)
	c.Check(stat.Duration > 0, Equals, true)
}

func (s *HealthCheckTestSuite) TestRun_Timeout(c *C) {
	// Verify a timed out health check
	check := HealthCheck{
		Script:    "sleep 1",
		Timeout:   250 * time.Millisecond,
		Interval:  time.Second,
		Tolerance: 0,
	}
	stat := check.Run()
	c.Check(stat.Status, Equals, Timeout)
	c.Check(stat.Output, DeepEquals, []byte{})
	c.Check(stat.Err, Equals, ErrTimeout)
	c.Check(stat.Duration >= check.Timeout, Equals, true)
}

func (s *HealthCheckTestSuite) TestRun_Failed(c *C) {
	// Verify a failing health check
	check := HealthCheck{
		Script:    "echo -n failure >&2; return 1",
		Timeout:   time.Second,
		Interval:  time.Second,
		Tolerance: 0,
	}
	stat := check.Run()
	c.Check(stat.Status, Equals, Failed)
	c.Check(stat.Output, DeepEquals, []byte("failure"))
	c.Check(stat.Err, NotNil)
	c.Check(stat.Duration > 0, Equals, true)
}
