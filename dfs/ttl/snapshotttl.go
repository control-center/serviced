// Copyright 2015 The Serviced Authors.
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

package ttl

import (
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

const timeFormat = "20060102-150405.000"

// SnapshotTTLInterface is the client handler for SnapshotTTL
type SnapshotTTLInterface interface {
	// GetServices returns all services
	GetServices(dao.ServiceRequest, *[]service.Service) error
	// ListSnapshots returns the list of all snapshots given a service id
	ListSnapshots(string, *[]dao.SnapshotInfo) error
	// DeleteSnapshot deletes a snapshot by SnapshotID
	DeleteSnapshot(string, *int) error
}

// SnapshotTTL is the TTL for snapshots
type SnapshotTTL struct {
	client SnapshotTTLInterface
}

// RunSnapshotTTL runs the ttl for snapshots
func RunSnapshotTTL(client SnapshotTTLInterface, cancel <-chan interface{}, min, max time.Duration) {
	utils.RunTTL(&SnapshotTTL{client}, cancel, min, max)
}

// Purge deletes snapshots as they reach a particular age.  Returns the time to
// wait til the next snapshot is to be deleted.
// Implements utils.TTL
func (ttl *SnapshotTTL) Purge(age time.Duration) (time.Duration, error) {
	expire := time.Now().Add(-age)
	var svcs []service.Service
	if err := ttl.client.GetServices(dao.ServiceRequest{}, &svcs); err != nil {
		glog.Errorf("Could not look up services: %s", err)
		return 0, err
	}

	for _, svc := range svcs {
		// find the tenant services
		if svc.ParentServiceID == "" {
			var snapshots []dao.SnapshotInfo
			if err := ttl.client.ListSnapshots(svc.ID, &snapshots); err != nil {
				glog.Errorf("Could not look up snapshots for tenant service %s (%s): %s", svc.Name, svc.ID, err)
				return 0, err
			}
			for _, s := range snapshots {
				//ignore snapshots that have any tag
				if len(s.Tags) == 0 {
					// check the age of the snapshot
					if timeToLive := s.Created.Sub(expire); timeToLive <= 0 {
						if err := ttl.client.DeleteSnapshot(s.SnapshotID, nil); err != nil {
							glog.Errorf("Could not delete snapshot %s for tenant service %s (%s): %s", s.SnapshotID, svc.Name, svc.ID, err)
							return 0, err
						}
					} else if timeToLive < age {
						// set the new time to live based on the age of the
						// oldest non-expired snapshot.
						age = timeToLive
					}
				}
			}
		}
	}

	return age, nil
}
