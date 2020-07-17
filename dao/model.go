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
	"os"
	"time"

	"github.com/control-center/serviced/domain"
	svcdef "github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

// NullRequest identifies a no-nothing (empty) request
type NullRequest struct{}

// User contains creditials for a user.
type User struct {
	Name     string // the unique identifier for a user
	Password string // no requirements on passwords yet
}

// ServiceDeploymentRequest is a request to deploy a service from a service definition.
// Pool and deployment ids are derived from the parent.
type ServiceDeploymentRequest struct {
	PoolID    string // PoolID to deploy the service to
	ParentID  string // ID of parent service
	Overwrite bool   // Overwrites any existing service
	Service   svcdef.ServiceDefinition
}

// RunningService this is created by selecting from service_state and joining to service
type RunningService struct {
	ID                string
	ServiceID         string
	HostID            string
	IPAddress         string // IP that this service has assigned ports
	DockerID          string
	StartedAt         time.Time
	InSync            bool
	Name              string
	Startup           string
	Description       string
	Environment       []string
	Instances         int
	ImageID           string
	PoolID            string
	DesiredState      int
	ParentServiceID   string
	InstanceID        int
	RAMCommitment     utils.EngNotation
	RAMThreshold      uint
	CPUCommitment     uint64
	HostPolicy        svcdef.HostPolicy
	MonitoringProfile domain.MonitorProfile
}

// BackupFile is the structure for backup file data
type BackupFile struct {
	InProgress bool        `json:"in_progress"`
	FullPath   string      `json:"full_path"`
	Name       string      `json:"name"`
	Size       int64       `json:"size"`
	Mode       os.FileMode `json:"mode"`
	ModTime    time.Time   `json:"mod_time"`
}

// SnapshotInfo describes a snapshot
type SnapshotInfo struct {
	SnapshotID  string
	TenantID    string
	Description string
	Tags        []string
	Created     time.Time
	Invalid     bool
}

func (s SnapshotInfo) String() string {
	snapshotID := s.SnapshotID
	if s.Invalid {
		snapshotID += " [DEPRECATED]"
	}

	if s.Description == "" {
		return snapshotID
	}
	return snapshotID + " " + s.Description
}

// Equals returns true if the two SnapshotInfo objects have the same values.
func (s *SnapshotInfo) Equals(s2 *SnapshotInfo) bool {
	if len(s.Tags) != len(s2.Tags) {
		return false
	}

	for i := range s.Tags {
		if s.Tags[i] != s2.Tags[i] {
			return false
		}
	}

	return s.SnapshotID == s2.SnapshotID &&
		s.TenantID == s2.TenantID &&
		s.Description == s2.Description &&
		s.Created == s2.Created &&
		s.Invalid == s2.Invalid
}

// ServiceInstanceRequest requests information about a service instance given the service ID and instance ID.
type ServiceInstanceRequest struct {
	ServiceID  string
	InstanceID int
}

// BackupRequest is a request to create a backup.
type BackupRequest struct {
	Dirpath              string
	SnapshotSpacePercent int
	Excludes             []string
	Force                bool
	Username             string
}

// RestoreRequest is a request to restore from a backup file.
type RestoreRequest struct {
	Filename string
	Username string
}

// BackupEstimate is a set of fields that describe the estimated resource utilization of a backup.
type BackupEstimate struct {
	AvailableBytes  uint64
	EstimatedBytes  uint64
	AvailableString string
	EstimatedString string
	BackupPath      string
	AllowBackup     bool
}
