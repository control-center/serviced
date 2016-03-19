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
	"encoding/json"
	"errors"
	"os/exec"
	"time"
)

// DefaultTolerance is the default tolerance value in determining
// non-responsive health checks.
const DefaultTolerance int = 2

// DefaultExpiration is the default expiration for deprecated health status
// updates.
const DefaultExpiration time.Duration = time.Minute

const (
	// OK means health check is passing
	OK = "passed"
	// Failed means health check is responsive, but failing
	Failed = "failed"
	// Timeout means health check is non-responsive in the given time
	Timeout = "timeout"
	// NotRunning means the instance is not running
	NotRunning = "not running"
	// Unknown means the instance hasn't checked-in within the provided time
	// limit.
	Unknown = "unknown"
)

// ErrTimeout is the error message returned when a health check is not responsive
var ErrTimeout = errors.New("health check timed out")

// HealthStatus is the output from a provided health check.
type HealthStatus struct {
	Status    string
	Output    []byte
	Err       error
	StartedAt time.Time
	Duration  time.Duration
}

// HealthCheck is the health check object.
type HealthCheck struct {
	Script    string
	Timeout   time.Duration
	Interval  time.Duration
	Tolerance int
}

// MarshalJSON implements json.Marshaller
func (hc *HealthCheck) MarshalJSON() ([]byte, error) {
	jhc := struct {
		Script    string
		Timeout   float64
		Interval  float64
		Tolerance int
	}{
		Script:    hc.Script,
		Timeout:   hc.Timeout.Seconds(),
		Interval:  hc.Interval.Seconds(),
		Tolerance: hc.Tolerance,
	}
	return json.Marshal(jhc)
}

// UnmarshalJSON implements json.Unmarshaller
func (hc *HealthCheck) UnmarshalJSON(data []byte) error {
	jhc := struct {
		Script    string
		Timeout   float64
		Interval  float64
		Tolerance int
	}{}
	if err := json.Unmarshal(data, &jhc); err != nil {
		return err
	}
	*hc = HealthCheck{
		Script:    jhc.Script,
		Timeout:   time.Duration(jhc.Timeout) * time.Second,
		Interval:  time.Duration(jhc.Interval) * time.Second,
		Tolerance: jhc.Tolerance,
	}
	return nil
}

// Expires calculates the time to live on the cache item.
func (hc *HealthCheck) Expires() time.Duration {
	tolerance := hc.Tolerance
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	return time.Duration(tolerance) * (hc.Timeout + hc.Interval)
}

// NotRunning returns the health status for a service instance that is not
// running.
func (hc *HealthCheck) NotRunning() HealthStatus {
	return HealthStatus{
		Status:    NotRunning,
		Output:    []byte{},
		StartedAt: time.Now(),
	}
}

// Unknown returns the health status for a service instance whose health check
// response is unknown.
func (hc *HealthCheck) Unknown() HealthStatus {
	return HealthStatus{
		Status:    Unknown,
		Output:    []byte{},
		StartedAt: time.Now(),
	}
}

// Run returns the health status as a result of running the health check
// script.
func (hc *HealthCheck) Run() (stat HealthStatus) {
	stat.StartedAt = time.Now()
	buf := &bytes.Buffer{}
	cmd := exec.Command("sh", "-c", hc.Script)
	cmd.Stdout = buf
	cmd.Stderr = buf
	cmd.Start()
	timer := time.NewTimer(hc.Timeout)
	errC := make(chan error, 1)
	go func() {
		defer timer.Stop()
		errC <- cmd.Wait()
	}()
	go func() {
		defer cmd.Process.Kill()
		<-timer.C
		errC <- ErrTimeout
	}()
	if stat.Err = <-errC; stat.Err == nil {
		stat.Status = OK
	} else if stat.Err == ErrTimeout {
		stat.Status = Timeout
	} else {
		stat.Status = Failed
	}
	stat.Output = buf.Bytes()
	stat.Duration = time.Since(stat.StartedAt)
	return
}
