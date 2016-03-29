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
	"encoding/json"
	"os/exec"
	"time"
)

// DefaultTolerance is the default coefficient used to determine when a health
// check is expired
const DefaultTolerance int = 2

// DefaultTimeout is the default timeout setting used to determine when a
// health check is non-responsive
const DefaultTimeout time.Duration = 30 * time.Second

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
	NotRunning = "not_running"
	// Unknown means the instance hasn't checked in within the provided time
	// limit.
	Unknown = "unknown"
)

// HealthStatus is the output from a provided health check.
type HealthStatus struct {
	Status    string
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
func (hc HealthCheck) MarshalJSON() ([]byte, error) {
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

// GetTimeout returns the timeout duration.
func (hc *HealthCheck) GetTimeout() time.Duration {
	timeout := hc.Timeout
	if timeout <= 0 {
		timeout = DefaultTimeout
	}
	return timeout
}

// Expires calculates the time to live on the cache item.
func (hc *HealthCheck) Expires() time.Duration {
	tolerance := hc.Tolerance
	if tolerance <= 0 {
		tolerance = DefaultTolerance
	}
	return time.Duration(tolerance) * (hc.GetTimeout() + hc.Interval)
}

// NotRunning returns the health status for a service instance that is not
// running.
func (hc *HealthCheck) NotRunning() HealthStatus {
	return HealthStatus{
		Status:    NotRunning,
		StartedAt: time.Now(),
	}
}

// Unknown returns the health status for a service instance whose health check
// response is unknown.
func (hc *HealthCheck) Unknown() HealthStatus {
	return HealthStatus{
		Status:    Unknown,
		StartedAt: time.Now(),
	}
}

// Run returns the health status as a result of running the health check
// script.
func (hc *HealthCheck) Run() (stat HealthStatus) {
	stat.StartedAt = time.Now()
	cmd := exec.Command("sh", "-c", hc.Script)
	cmd.Start()
	timer := time.NewTimer(hc.GetTimeout())
	errC := make(chan error)
	go func() { errC <- cmd.Wait() }()
	select {
	case err := <-errC:
		timer.Stop()
		if err != nil {
			stat.Status = Failed
		} else {
			stat.Status = OK
		}
	case <-timer.C:
		cmd.Process.Kill()
		<-errC
		stat.Status = Timeout
	}
	stat.Duration = time.Since(stat.StartedAt)
	return
}

// Ping performs the health check on the specified interval.
func (hc *HealthCheck) Ping(cancel <-chan struct{}, report func(HealthStatus)) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			stat := hc.Run()
			timer.Reset(hc.Interval)
			report(stat)
		case <-cancel:
			return
		}
	}
}
