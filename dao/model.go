// Copyright 2014 The Serviced Authors.
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

package dao

import (
	"time"

	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
}

// An association between a host and a pool.
type PoolHost struct {
	HostID string
	PoolID string
	HostIP string
}

//AssignmentRequest is used to couple a serviceId to an IPAddress
type AssignmentRequest struct {
	ServiceID      string
	IPAddress      string
	AutoAssignment bool
}

// An exposed service endpoint
type ApplicationEndpoint struct {
	ServiceID      string
	Application    string
	ContainerPort  uint16
	HostPort       uint16
	HostIP         string
	ContainerIP    string
	Protocol       string
	VirtualAddress string
	InstanceID     int
}

// A request to deploy a service template
type ServiceTemplateDeploymentRequest struct {
	PoolID       string // Pool Id to deploy service into
	TemplateID   string // Id of template to be deployed
	DeploymentID string // Unique id of the instance of this template
}

// A request to deploy a service from a service definition
//  Pool and deployment ids are derived from the parent
type ServiceDeploymentRequest struct {
	ParentID string // ID of parent service
	Service  servicedefinition.ServiceDefinition
}

// This is created by selecting from service_state and joining to service
type RunningService struct {
	ID                string
	ServiceID         string
	HostID            string
	DockerID          string
	StartedAt         time.Time
	InSync            bool
	Name              string
	Startup           string
	Description       string
	Instances         int
	ImageID           string
	PoolID            string
	DesiredState      int
	ParentServiceID   string
	InstanceID        int
	MonitoringProfile domain.MonitorProfile
}

type Status struct {
	Key   int
	Value string
}

func (s Status) String() string {
	return s.Value
}

var (
	Scheduled = Status{1, "Scheduled"}
	Starting  = Status{2, "Starting"}
	Pausing   = Status{3, "Pausing"}
	Paused    = Status{4, "Paused"}
	Resuming  = Status{5, "Resuming"}
	Running   = Status{6, "Running"}
	Stopping  = Status{7, "Stopping"}
	Stopped   = Status{8, "Stopped"}
)

// An instantiation of a Snapshot request
type SnapshotRequest struct {
	ID            string
	ServiceID     string
	SnapshotLabel string
	SnapshotError string
}

// A new snapshot request instance (SnapshotRequest)
func NewSnapshotRequest(serviceId string, snapshotLabel string) (snapshotRequest *SnapshotRequest, err error) {
	snapshotRequest = &SnapshotRequest{}
	snapshotRequest.ID, err = utils.NewUUID36()
	if err == nil {
		snapshotRequest.ServiceID = serviceId
		snapshotRequest.SnapshotLabel = snapshotLabel
		snapshotRequest.SnapshotError = ""
	}
	return snapshotRequest, err
}
