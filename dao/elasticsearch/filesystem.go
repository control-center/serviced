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
	"fmt"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/volume"

	"github.com/control-center/serviced/zzk"
	zkSnapshot "github.com/control-center/serviced/zzk/snapshot"
	"github.com/zenoss/glog"

	"errors"
)

// GetVolume gets the volume of a service
func (this *ControlPlaneDao) GetVolume(serviceID string, volume volume.Volume) error {
	var tenantID string
	if err := this.GetTenantId(serviceID, &tenantID); err != nil {
		glog.Errorf("Could not find tenant for service %s: %s", serviceID, err)
		return err
	}

	var err error
	volume, err = this.dfs.GetVolume(tenantID)
	return err
}

// ResetRegistry resets the docker registry
func (this *ControlPlaneDao) ResetRegistry(request dao.EntityRequest, unused *int) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	return this.dfs.ResetRegistry()
}

// DeleteSnapshot deletes a particular snapshot
func (this *ControlPlaneDao) DeleteSnapshot(snapshotID string, unused *int) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	return this.dfs.DeleteSnapshot(snapshotID)
}

// DeleteSnapshots deletes all snapshots given a tenant
func (this *ControlPlaneDao) DeleteSnapshots(serviceID string, unused *int) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	return this.dfs.DeleteSnapshots(serviceID)
}

// Rollback rolls back the dfs to a particular snapshot
func (this *ControlPlaneDao) Rollback(request dao.RollbackRequest, unused *int) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()
	return this.dfs.Rollback(request.SnapshotID, request.ForceRestart)
}

// Snapshot takes a snapshot of the dfs and its respective images
func (this *ControlPlaneDao) Snapshot(request dao.SnapshotRequest, snapshotID *string) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()

	var tenantID string
	if err := this.GetTenantId(request.ServiceID, &tenantID); err != nil {
		glog.Errorf("Could not snapshot %s: %s", request.ServiceID, err)
		return err
	}

	var err error
	*snapshotID, err = this.dfs.Snapshot(tenantID, request.Description)
	return err
}

// AsyncSnapshot is the asynchronous call to snapshot
func (this *ControlPlaneDao) AsyncSnapshot(serviceID string, snapshotID *string) error {
	poolID, err := this.facade.GetPoolForService(datastore.Get(), serviceID)
	if err != nil {
		glog.Errorf("Unable to get pool for service %v: %v", serviceID, err)
		return err
	}

	conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
	if err != nil {
		glog.Errorf("Cannot establish connection to zk via pool %s: %s", poolID, err)
		return err
	}

	var ss zkSnapshot.Snapshot
	if nodeID, err := zkSnapshot.Send(conn, serviceID); err != nil {
		glog.Errorf("Could not submit snapshot for %s: %s", serviceID, err)
		return err
	} else if err := zkSnapshot.Recv(conn, nodeID, &ss); err != nil {
		glog.Errorf("Could not receieve snapshot for %s (%s): %s", serviceID, nodeID, err)
		return err
	}

	*snapshotID = ss.Label
	if ss.Err != "" {
		return errors.New(ss.Err)
	}
	return nil
}

// ListSnapshots lists all the available snapshots for a particular service
func (this *ControlPlaneDao) ListSnapshots(serviceID string, snapshots *[]dao.SnapshotInfo) error {
	var tenantID string
	if err := this.GetTenantId(serviceID, &tenantID); err != nil {
		glog.Errorf("Could not find tenant for %s: %s", serviceID, err)
		return err
	} else if *snapshots, err = this.dfs.ListSnapshots(tenantID); err != nil {
		glog.Errorf("Could not get snapshots for %s (%s): %s", serviceID, tenantID, err)
		return err
	}

	return nil
}

// Commit commits a container to a particular tenant and snapshots the resulting image
func (this *ControlPlaneDao) Commit(containerID string, snapshotID *string) error {
	this.dfs.Lock()
	defer this.dfs.Unlock()

	var err error
	*snapshotID, err = this.dfs.Commit(containerID)
	return err
}

// ReadyDFS verifies that no other dfs operations are in progress
func (this *ControlPlaneDao) ReadyDFS(unused bool, unusedint *int) (err error) {
	if locked, err := this.dfs.IsLocked(); err != nil {
		return err
	} else if locked {
		return fmt.Errorf("another dfs operation is running")
	}
	return nil
}
