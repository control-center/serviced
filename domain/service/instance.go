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
	"time"

	"github.com/control-center/serviced/domain/servicedefinition"
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
	ContainerID  string
	ImageSynced  bool
	DesiredState DesiredState
	CurrentState CurrentState
	HealthStatus map[string]health.Status
	Scheduled    time.Time
	Started      time.Time
	Terminated   time.Time
}

// StrategyInstance collects service strategy information about a service
// instance.
type StrategyInstance struct {
	HostID        string
	ServiceID     string
	CPUCommitment int
	RAMCommitment uint64
	HostPolicy    servicedefinition.HostPolicy
}

// LocationInstance collection location information about a service instance
type LocationInstance struct {
	HostID      string
	HostIP      string
	ContainerID string
}
