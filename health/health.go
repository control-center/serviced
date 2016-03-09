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

package health

import (
	"bytes"
	"errors"
	"os/exec"
	"time"
)

const DefaultTolerance int = 60

const (
	OK         = "passed"
	Failed     = "failed"
	Timeout    = "timeout"
	NotRunning = "not running"
	Unknown    = "unknown"
)

var ErrTimeout = errors.New("health check timed out")

// HealthStatusRequest is a remote request to write to the health check cache
type HealthStatusRequest struct {
	Key     HealthStatusKey
	Value   *HealthStatus
	Expires time.Duration
}

// HealthStatus is the output from a provided health check
type HealthStatus struct {
	Status    string
	Output    []byte
	Err       error
	StartedAt time.Time
	Duration  time.Duration
}

// HealthCheck is the health check object
type HealthCheck struct {
	Script    string
	Timeout   time.Duration
	Interval  time.Duration
	Tolerance int
}

// Expires calculates the time to live on the cache item
func (hc *HealthCheck) Expires() time.Duration {
	tolerance := hc.Tolerance
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	return time.Duration(tolerance) * (hc.Timeout + hc.Interval)
}

// Unknown returns an unknown health status
func (hc *HealthCheck) Unknown() *HealthStatus {
	return &HealthStatus{
		Status:    Unknown,
		StartedAt: time.Now(),
	}
}

// NotRunning returns a not running health status
func (hc *HealthCheck) NotRunning() *HealthStatus {
	return &HealthStatus{
		Status:    NotRunning,
		StartedAt: time.Now(),
	}
}

// Run returns the health status for a provided container
// TODO: add a remote address option for docker
func (hc *HealthCheck) Run() *HealthStatus {
	stat := hc.Unknown()
	buf := bytes.NewBuffer(stat.Output)
	cmd := exec.Command("sh", "-c", hc.Script)
	cmd.Stdout = buf
	cmd.Stderr = buf
	cmd.Start()
	timer := time.NewTimer(hc.Timeout)
	defer timer.Stop()
	errC := make(chan error)
	go func() {
		select {
		case errC <- cmd.Wait():
		case <-timer.C:
			cmd.Process.Kill()
			errC <- ErrTimeout
		}
	}()
	if stat.Err = <-errC; stat.Err != nil {
		if idx := bytes.Index(stat.Output, []byte("No such container")); idx >= 0 {
			stat.Status = NotRunning
		} else if stat.Err == ErrTimeout {
			stat.Status = Timeout
		} else {
			stat.Status = Failed
		}
	} else {
		stat.Status = OK
	}
	stat.Duration = time.Since(stat.StartedAt)
	return stat
}
