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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/utils"
)

const timeFormat = "20060102-150405.000"

// instantiate the package logger
var plog = logging.PackageLogger()

// SnapshotTTLInterface is the client handler for SnapshotTTL
type SnapshotTTLInterface interface {
	// GetTenantIDs returns all tenant IDs
	GetTenantIDs(struct{}, *[]string) error
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

// Name identifies the TTL instance
func (ttl *SnapshotTTL) Name() string {
	return "SnapshotTTL"
}

// Purge deletes snapshots as they reach a particular age.  Returns the time to
// wait til the next snapshot is to be deleted.
// Implements utils.TTL
func (ttl *SnapshotTTL) Purge(age time.Duration) (time.Duration, error) {
	ctx := datastore.GetContext()
	defer ctx.Metrics().Stop(ctx.Metrics().Start("SnapshotTTL.Purge"))

	logger := plog.WithField("age", int(age.Minutes()))

	expire := time.Now().Add(-age)
	var tenantIDs []string
	var unused struct{}
	if err := ttl.client.GetTenantIDs(unused, &tenantIDs); err != nil {
		logger.WithError(err).Error("Could not look up tenantIDs")
		return 0, err
	}

	for _, tenantID := range tenantIDs {
		var snapshots []dao.SnapshotInfo
		if err := ttl.client.ListSnapshots(tenantID, &snapshots); err != nil {
			logger.WithField("tenantid", tenantID).
				WithError(err).Error("Could not look up snapshots for tenant service")
			return 0, err
		}
		for _, s := range snapshots {
			//ignore snapshots that have any tag
			if len(s.Tags) == 0 {
				// check the age of the snapshot
				if timeToLive := s.Created.Sub(expire); timeToLive <= 0 {
					snapshotLogger := logger.WithFields(log.Fields{
						"tenantid":   tenantID,
						"snapshotid": s.SnapshotID,
					})
					if err := ttl.client.DeleteSnapshot(s.SnapshotID, nil); err != nil {
						snapshotLogger.WithError(err).Error("Could not delete snapshot")
						return 0, err
					}
					snapshotLogger.Debug("Deleted snapshot")
				} else if timeToLive < age {
					// set the new time to live based on the age of the
					// oldest non-expired snapshot.
					age = timeToLive
				}
			}
		}
	}
	logger.WithField("age", int(age.Minutes())).Debug("Finished Purge")

	return age, nil
}
