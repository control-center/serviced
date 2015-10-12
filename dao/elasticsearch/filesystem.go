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

package elasticsearch

import (
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
)

// ResetRegistry resets the docker registry
func (this *ControlPlaneDao) ResetRegistry(request dao.EntityRequest, _ *int) error {
	return this.facade.SyncRegistryImages(datastore.Get(), true)
}

// RepairRegistry repairs the docker registry during an upgrade
func (this *ControlPlaneDao) RepairRegistry(_ struct{}, _ *struct{}) error {
	return this.facade.RepairRegistry(datastore.Get())
}

// DeleteSnapshot deletes a particular snapshot
func (this *ControlPlaneDao) DeleteSnapshot(snapshotID string, _ *int) error {
	return this.facade.DeleteSnapshot(datastore.Get(), snapshotID)
}

// Rollback rolls back the dfs to a particular snapshot
func (this *ControlPlaneDao) Rollback(request dao.RollbackRequest, unused *int) error {
	return this.facade.Rollback(datastore.Get(), request.SnapshotID, request.ForceRestart)
}

// Snapshot takes a snapshot of the dfs and its respective images
func (this *ControlPlaneDao) Snapshot(request dao.SnapshotRequest, snapshotID *string) (err error) {
	*snapshotID, err = this.facade.Snapshot(datastore.Get(), request.ServiceID, request.Description, []string{})
	return
}

// ListSnapshots lists all the available snapshots for a particular service
func (this *ControlPlaneDao) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) error {
	snaps, err := this.facade.ListSnapshots(datastore.Get(), serviceID)
	if err != nil {
		return err
	}
	*snapshots = make([]dao.SnapshotInfo, len(snaps))
	for i, snap := range snaps {
		info, err := this.facade.GetSnapshotInfo(datastore.Get(), snap)
		if err != nil {
			return err
		}
		(*snapshots)[i] = dao.SnapshotInfo{info.Info.Name, info.Info.Message}
	}
	return nil
}

// DeleteSnapshots deletes all snapshots for a particular service
func (this *ControlPlaneDao) DeleteSnapshots(serviceID string, _ *int) error {
	snaps, err := this.facade.ListSnapshots(datastore.Get(), serviceID)
	if err != nil {
		return err
	}
	for _, snap := range snaps {
		if err := this.facade.DeleteSnapshot(datastore.Get(), snap); err != nil {
			return err
		}
	}
	return nil
}

// Commit commits a container to a particular tenant and snapshots the resulting image
func (this *ControlPlaneDao) Commit(containerID string, snapshotID *string) (err error) {
	*snapshotID, err = this.facade.Commit(datastore.Get(), containerID, "", []string{})
	return
}

func (this *ControlPlaneDao) ReadyDFS(unused bool, unusedint *int) error {
	return nil
}
