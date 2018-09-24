// Copyright 2016-2018 The Serviced Authors.
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

// +build unit test

package health_test

import (
	"encoding/json"
	"time"

	. "github.com/control-center/serviced/health"
	. "gopkg.in/check.v1"
)

type jsonhealthcheck struct {
	Script    string
	Timeout   float64
	Interval  float64
	Tolerance int
}

var (
	_     = Suite(&HealthCheckTestSuite{})
	hcKey = HealthStatusKey{
		InstanceID:      0,
		ServiceID:       "serviceid",
		HealthCheckName: "healthcheck",
	}
)

type HealthCheckTestSuite struct{}

func (s *HealthCheckTestSuite) TestMarshalJSON(c *C) {
	// Verify the marshaller
	check := HealthCheck{
		Script:    "echo testscript",
		Timeout:   time.Minute,
		Interval:  2 * time.Second,
		Tolerance: 5,
	}
	data, err := json.Marshal(&check)
	c.Assert(err, IsNil)

	expected := jsonhealthcheck{
		Script:    check.Script,
		Timeout:   60,
		Interval:  2,
		Tolerance: 5,
	}
	var actual jsonhealthcheck
	err = json.Unmarshal(data, &actual)
	c.Assert(err, IsNil)
	c.Check(actual, DeepEquals, expected)
}

func (s *HealthCheckTestSuite) TestMarshalJSON_Map(c *C) {
	// Verify the marshaller works given a map of HealthCheck data
	checkMap := map[string]HealthCheck{
		"testscript": {
			Script:    "echo testscript",
			Timeout:   time.Minute,
			Interval:  2 * time.Second,
			Tolerance: 5,
		},
	}
	data, err := json.Marshal(&checkMap)
	c.Assert(err, IsNil)
	expected := map[string]jsonhealthcheck{
		"testscript": {
			Script:    "echo testscript",
			Timeout:   60,
			Interval:  2,
			Tolerance: 5,
		},
	}
	actual := make(map[string]jsonhealthcheck)
	err = json.Unmarshal(data, &actual)
	c.Assert(err, IsNil)
	c.Check(actual, DeepEquals, expected)
}

func (s *HealthCheckTestSuite) TestUnmarshalJSON(c *C) {
	// Verify the unmarshaller
	jhc := jsonhealthcheck{
		Script:    "echo testscript",
		Timeout:   60,
		Interval:  2,
		Tolerance: 5,
	}
	bytes, err := json.Marshal(jhc)
	c.Assert(err, IsNil)

	expected := HealthCheck{
		Script:    "echo testscript",
		Timeout:   time.Minute,
		Interval:  2 * time.Second,
		Tolerance: 5,
	}
	var actual HealthCheck
	err = json.Unmarshal(bytes, &actual)
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (s *HealthCheckTestSuite) TestUnmarshalJSON_Map(c *C) {
	// Verify the unmarshaller works given a map of HealthCheck data
	jhcMap := map[string]jsonhealthcheck{
		"testscript": {
			Script:    "echo testscript",
			Timeout:   60,
			Interval:  2,
			Tolerance: 5,
		},
	}
	bytes, err := json.Marshal(jhcMap)
	c.Assert(err, IsNil)

	expected := map[string]HealthCheck{
		"testscript": {
			Script:    "echo testscript",
			Timeout:   time.Minute,
			Interval:  2 * time.Second,
			Tolerance: 5,
		},
	}
	actual := make(map[string]HealthCheck)
	err = json.Unmarshal(bytes, &actual)
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (s *HealthCheckTestSuite) TestGetTimeout(c *C) {
	// Verify the healthcheck timeout
	check := HealthCheck{
		Script:    "echo testscript",
		Timeout:   0,
		Interval:  2 * time.Second,
		Tolerance: 0,
	}
	c.Check(check.GetTimeout(), Equals, DefaultTimeout)
	check.Timeout = 15 * time.Second
	c.Check(check.GetTimeout(), Equals, check.Timeout)
}

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
	c.Check(int(stat.Status), Equals, NotRunning)
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
	c.Check(int(stat.Status), Equals, Unknown)
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
	stat := check.Run(hcKey)
	c.Check(stat.Status, Equals, OK)
	c.Check(stat.Duration > 0, Equals, true)
}

func (s *HealthCheckTestSuite) TestRun_Timeout(c *C) {
	// Verify a timed out health check
	check := HealthCheck{
		Script:    "sleep 5",
		Timeout:   250 * time.Millisecond,
		Interval:  time.Second,
		Tolerance: 0,
	}
	stat := check.Run(hcKey)
	c.Check(int(stat.Status), Equals, Timeout)
	c.Check(stat.Duration >= check.Timeout, Equals, true)
	c.Check(stat.Duration < 5*time.Second, Equals, true)
}

func (s *HealthCheckTestSuite) TestRun_Failed(c *C) {
	// Verify a failing health check
	check := HealthCheck{
		Script:    "echo -n failure >&2; return 1",
		Timeout:   time.Second,
		Interval:  time.Second,
		Tolerance: 0,
	}
	stat := check.Run(hcKey)
	c.Check(int(stat.Status), Equals, Failed)
	c.Check(stat.Duration > 0, Equals, true)
}

func (s *HealthCheckTestSuite) TestPing(c *C) {
	// Verify ping
	check := HealthCheck{
		Script:    "echo -n testscript",
		Timeout:   time.Second,
		Interval:  500 * time.Millisecond,
		Tolerance: 0,
	}
	cancel := make(chan struct{})
	startTime := time.Now()
	interval := 0
	check.Ping(cancel, hcKey, func(stat HealthStatus) {
		switch interval {
		case 0:
			c.Check(time.Since(startTime) < check.Interval, Equals, true)
			startTime = time.Now()
		case 1:
			close(cancel)
			c.Check(time.Since(startTime) >= check.Interval, Equals, true)
		default:
			c.Errorf("Ping ran %d times longer than expected", interval-1)
		}
		interval++
	})
}

func (s *HealthCheckTestSuite) TestKillScript_MatchingSingleExitCode(c *C) {
	// Verify that if we fail a healthcheck 3 times that the result has the kill flag when
	// our exit code matches the single exit code in our list.
	hc := HealthCheck{
		Script:         "exit 28",
		Timeout:        time.Second,
		Interval:       500 * time.Millisecond,
		Tolerance:      0,
		KillExitCodes:  []int{28},
		KillCountLimit: 3,
	}
	cancel := make(chan struct{})
	interval := 0
	hc.Ping(cancel, hcKey, func(status HealthStatus) {
		interval++
		switch interval {
		case 1:
			c.Check(hc.KillCounter, Equals, 1)
			c.Check(status.KillFlag, Equals, false)
		case 2:
			c.Check(hc.KillCounter, Equals, 2)
			c.Check(status.KillFlag, Equals, false)
		case 3:
			c.Check(hc.KillCounter, Equals, 3)
			c.Check(status.KillFlag, Equals, true)
			close(cancel)
		}
	})
}

func (s *HealthCheckTestSuite) TestKillScript_MatchingMultipleExitCodes(c *C) {
	// Verify that if we fail a healthcheck 3 times that the result has the kill flag when
	// our exit code matches one of the multiple exit codes in our list.
	hc := HealthCheck{
		Script:         "exit 28",
		Timeout:        time.Second,
		Interval:       500 * time.Millisecond,
		Tolerance:      0,
		KillExitCodes:  []int{15, 28, 55},
		KillCountLimit: 3,
	}
	cancel := make(chan struct{})
	interval := 0
	hc.Ping(cancel, hcKey, func(status HealthStatus) {
		interval++
		switch interval {
		case 1:
			c.Check(hc.KillCounter, Equals, 1)
			c.Check(status.KillFlag, Equals, false)
		case 2:
			c.Check(hc.KillCounter, Equals, 2)
			c.Check(status.KillFlag, Equals, false)
		case 3:
			c.Check(hc.KillCounter, Equals, 3)
			c.Check(status.KillFlag, Equals, true)
			close(cancel)
		}
	})
}

func (s *HealthCheckTestSuite) TestKillScript_MatchingEmptyExitCodes(c *C) {
	// Verify that if we fail a healthcheck 3 times that the result has the kill flag when
	// our exit code list is empty.
	hc := HealthCheck{
		Script:         "exit 28",
		Timeout:        time.Second,
		Interval:       500 * time.Millisecond,
		Tolerance:      0,
		KillExitCodes:  []int{},
		KillCountLimit: 3,
	}
	cancel := make(chan struct{})
	interval := 0
	hc.Ping(cancel, hcKey, func(status HealthStatus) {
		interval++
		switch interval {
		case 1:
			c.Check(hc.KillCounter, Equals, 1)
			c.Check(status.KillFlag, Equals, false)
		case 2:
			c.Check(hc.KillCounter, Equals, 2)
			c.Check(status.KillFlag, Equals, false)
		case 3:
			c.Check(hc.KillCounter, Equals, 3)
			c.Check(status.KillFlag, Equals, true)
			close(cancel)
		}
	})
}

func (s *HealthCheckTestSuite) TestKillScript_KillCountResetsOnRecovery(c *C) {
	// Verify that if we have 2 failing kill codes, then the check recovers (exit 0), that
	// the kill counter is reset to 0.
	hc := HealthCheck{
		Script:         "exit 28",
		Timeout:        time.Second,
		Interval:       500 * time.Millisecond,
		Tolerance:      0,
		KillExitCodes:  []int{28},
		KillCountLimit: 3,
	}
	cancel := make(chan struct{})
	interval := 0
	hc.Ping(cancel, hcKey, func(status HealthStatus) {
		interval++
		switch interval {
		case 1:
			c.Check(hc.KillCounter, Equals, 1)
			c.Check(status.KillFlag, Equals, false)
		case 2:
			c.Check(hc.KillCounter, Equals, 2)
			c.Check(status.KillFlag, Equals, false)
			hc.Script = "exit 0"
		case 3:
			c.Check(hc.KillCounter, Equals, 0)
			c.Check(status.KillFlag, Equals, false)
			close(cancel)
		}
	})
}

func (s *HealthCheckTestSuite) TestKillScript_KillCountResetsOnOtherErrors(c *C) {
	// Verify that if we have 2 failing kill codes, then the script exits on any other
	// non-zero code (but doesn't match our kill code list), that the counter resets to 0.
	hc := HealthCheck{
		Script:         "exit 28",
		Timeout:        time.Second,
		Interval:       500 * time.Millisecond,
		Tolerance:      0,
		KillExitCodes:  []int{28},
		KillCountLimit: 3,
	}
	cancel := make(chan struct{})
	interval := 0
	hc.Ping(cancel, hcKey, func(status HealthStatus) {
		interval++
		switch interval {
		case 1:
			c.Check(hc.KillCounter, Equals, 1)
			c.Check(status.KillFlag, Equals, false)
		case 2:
			c.Check(hc.KillCounter, Equals, 2)
			c.Check(status.KillFlag, Equals, false)
			hc.Script = "exit 1"
		case 3:
			c.Check(hc.KillCounter, Equals, 0)
			c.Check(status.KillFlag, Equals, false)
			close(cancel)
		}
	})
}
