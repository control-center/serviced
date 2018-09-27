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

package health

import (
	"encoding/json"
	"errors"
	"os/exec"
	"syscall"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
)

var (
	plog = logging.PackageLogger()
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

// Status is the status of a health check
type Status int

const (
	// OK means health check is passing
	OK Status = 0
	// Failed means health check is responsive, but failing
	Failed = 1
	// Timeout means health check is non-responsive in the given time
	Timeout = 2
	// Unknown means the instance hasn't checked in within the provided time
	// limit.
	Unknown = 3
	// NotRunning means the instance is not running
	NotRunning = 4
)

func (s Status) MarshalJSON() ([]byte, error) {
	switch s {
	case OK:
		return []byte(`"passed"`), nil
	case Failed:
		return []byte(`"failed"`), nil
	case Timeout:
		return []byte(`"timeout"`), nil
	case Unknown:
		return []byte(`"unknown"`), nil
	case NotRunning:
		return []byte(`"not_running"`), nil
	default:
		return []byte{}, errors.New("invalid status value")
	}
}

func (s *Status) UnmarshalJSON(data []byte) error {
	var stat string
	if err := json.Unmarshal(data, &stat); err != nil {
		return err
	}

	switch stat {
	case "passed":
		*s = OK
	case "failed":
		*s = Failed
	case "timeout":
		*s = Timeout
	case "unknown":
		*s = Unknown
	case "not_running":
		*s = NotRunning
	default:
		return errors.New("invalid status value")
	}
	return nil
}

// HealthStatus is the output from a provided health check.
type HealthStatus struct {
	Status    Status
	StartedAt time.Time
	Duration  time.Duration
	KillFlag  bool
}

// HealthCheck is the health check object.
type HealthCheck struct {
	Script    string
	Timeout   time.Duration
	Interval  time.Duration
	Tolerance int
	// Kill properties will kill the container if the observed error code is in the error code list
	// and this happens <count>-times.  If the error codes list is empty and the kill count is >0, then any
	// non-zero error code will count toward the kill count.
	KillExitCodes  []int
	KillCountLimit int
	KillCounter    int
}

// MarshalJSON implements json.Marshaller
func (hc HealthCheck) MarshalJSON() ([]byte, error) {
	jhc := struct {
		Script         string
		Timeout        float64
		Interval       float64
		Tolerance      int
		KillExitCodes  []int `json:",omitempty"`
		KillCountLimit int   `json:",omitempty"`
	}{
		Script:         hc.Script,
		Timeout:        hc.Timeout.Seconds(),
		Interval:       hc.Interval.Seconds(),
		Tolerance:      hc.Tolerance,
		KillCountLimit: hc.KillCountLimit,
		KillExitCodes:  hc.KillExitCodes,
	}
	return json.Marshal(jhc)
}

// UnmarshalJSON implements json.Unmarshaller
func (hc *HealthCheck) UnmarshalJSON(data []byte) error {
	jhc := struct {
		Script         string
		Timeout        float64
		Interval       float64
		Tolerance      int
		KillExitCodes  []int `json:",omitempty"`
		KillCountLimit int   `json:",omitempty"`
	}{}
	if err := json.Unmarshal(data, &jhc); err != nil {
		return err
	}
	*hc = HealthCheck{
		Script:         jhc.Script,
		Timeout:        time.Duration(jhc.Timeout) * time.Second,
		Interval:       time.Duration(jhc.Interval) * time.Second,
		Tolerance:      jhc.Tolerance,
		KillCountLimit: jhc.KillCountLimit,
		KillExitCodes:  jhc.KillExitCodes,
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
func (hc *HealthCheck) Run(key HealthStatusKey) (stat HealthStatus) {
	logger := plog.WithFields(log.Fields{
		"service":     key.ServiceID,
		"instance":    key.InstanceID,
		"healthcheck": key.HealthCheckName,
	})
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
			// If the command gives an error, the healthcheck status is Failed (curl command failed, connection
			// refused, or any other error message including one that might contribute to the kill count)
			stat.Status = Failed
			if hc.KillCountLimit > 0 {
				logger.Debug("Healthcheck has a KillCount.. checking the exit code")

				if len(hc.KillExitCodes) == 0 {
					// No error codes listed means any error code will count toward the health check kill count.
					hc.KillCounter++
					logger.Debugf("No KillExitCodes provided; KillCounter is now %d", hc.KillCounter)
				} else if exitError, ok := err.(*exec.ExitError); ok {
					// Get the exit code and compare to our kill code list.
					ws := exitError.Sys().(syscall.WaitStatus)
					exitcode := ws.ExitStatus()
					logger.Debugf("Got exit code: %d", exitcode)
					found := false
					for i := range hc.KillExitCodes {
						if hc.KillExitCodes[i] == exitcode {
							hc.KillCounter++
							logger.Infof("found matching exit code %d.. KillCounter is now %d", exitcode, hc.KillCounter)
							found = true
							break
						}
					}
					if !found {
						// This isn't in our list of kill codes; we don't care about any of the others.  Reset the count
						logger.Debug("No matching exit code was found")
						if hc.KillCounter > 0 {
							logger.Infof("Resetting KillCounter. KillCounter was %d", hc.KillCounter)
							hc.KillCounter = 0
						}
					}
				} else {
					logger.WithError(err).Warn("Unable to read the health check exit code")
				}
			}
		} else {
			if hc.KillCounter > 0 {
				logger.Infof("Resetting KillCounter. KillCounter was %d", hc.KillCounter)
				hc.KillCounter = 0
			}
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
func (hc *HealthCheck) Ping(cancel <-chan struct{}, key HealthStatusKey, report func(HealthStatus)) {
	timer := time.NewTimer(0)
	defer timer.Stop()
	for {
		select {
		case <-timer.C:
			stat := hc.Run(key)
			stat.KillFlag = hc.KillCountLimit > 0 && hc.KillCounter >= hc.KillCountLimit
			timer.Reset(hc.Interval)
			report(stat)
		case <-cancel:
			return
		}
	}
}
