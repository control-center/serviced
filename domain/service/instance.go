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

package service

import (
	"bytes"
	"strconv"
	"text/template"
	"time"

	"github.com/control-center/serviced/health"
)

// CurrentState tracks the current state of a service instance
type CurrentState string

const (
	Stopping CurrentState = "stopping"
	Starting              = "starting"
	Pausing               = "pausing"
	Paused                = "paused"
	Running               = "running"
	Stopped               = "stopped"
)

// Instance describes an instance of a service
type Instance struct {
	ID           int
	HostID       string
	HostName     string
	ServiceID    string
	ServiceName  string // FIXME: service path would be better
	DockerID     string
	ImageSynced  bool
	DesiredState DesiredState
	CurrentState CurrentState
	HealthStatus map[string]health.Status
	Scheduled    time.Time
	Started      time.Time
	Terminated   time.Time
}

// GetPort returns the port number given the instance id
func GetPort(portTemplate string, defaultPort uint16, instanceID int) (uint16, error) {
	if portTemplate != "" {
		funcMap := template.FuncMap{
			"plus": func(a, b int) int { return a + b },
		}
		t := template.Must(template.New("PortNumber").Funcs(funcMap).Parse(portTemplate))
		buffer := &bytes.Buffer{}
		if err := t.Execute(buffer, struct{ InstanceID int }{InstanceID: instanceID}); err != nil {
			return 0, err
		}
		port, err := strconv.Atoi(buffer.String())
		if err != nil {
			return 0, err
		}
		return uint16(port), nil
	}
	return defaultPort, nil
}
