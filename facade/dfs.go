// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"errors"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfsnew"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

// Backup takes a backup of all installed applications
func (f *Facade) Backup(ctx datastore.Context, w io.Writer) error {
	stime := time.Now()
	message := fmt.Sprintf("started backup at %s", stime.UTC())
	glog.Infof("Starting backup")
	templates, images, err := f.GetServiceTemplatesAndImages(ctx)
	if err != nil {
		glog.Errorf("Could not get service templates and images: %s", err)
		return err
	}
	glog.Infof("Loaded templates and their images")
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		glog.Errorf("Could not get resource pools: %s", err)
		return err
	}
	glog.Infof("Loaded resource pools")
	hosts, err := f.GetHosts(ctx)
	if err != nil {
		glog.Errorf("Could not get hosts: %s", err)
		return err
	}
	glog.Infof("Loaded hosts")
	tenants, err := f.getTenantIDs(ctx)
	if err != nil {
		glog.Errorf("Could not get tenants: %s", err)
		return err
	}
	snapshots := make([]string, len(tenants))
	for i, tenant := range tenants {
		tag := fmt.Sprintf("backup-%s-%s", tenant, stime)
		snapshot, err := f.Snapshot(ctx, tenant, message, []string{tag})
		if err != nil {
			glog.Errorf("Could not snapshot %s: %s", tenant, err)
			return err
		}
		defer f.DeleteSnapshot(ctx, snapshot)
		snapshots[i] = snapshot
		glog.Infof("Created a snapshot for tenant %s at %s", tenant, snapshot)
	}
	glog.Infof("Loaded tenants")
	data := dfs.BackupInfo{
		Templates:  templates,
		BaseImages: images,
		Pools:      pools,
		Hosts:      hosts,
		Snapshots:  snapshots,
		Timestamp:  stime,
	}
	if err := f.dfs.Backup(data, w); err != nil {
		glog.Errorf("Could not backup: %s", err)
		return err
	}
	glog.Infof("Completed backup in %s", time.Since(stime))
	return nil
}

// BackupInfo returns metadata info about a backup
func (f *Facade) BackupInfo(ctx datastore.Context, r io.Reader) (*dfs.BackupInfo, error) {
	info, err := f.dfs.BackupInfo(r)
	if err != nil {
		glog.Errorf("Could not get info for backup: %s", err)
		return nil, err
	}
	return info, nil
}

// Commit commits a container to the docker registry and takes a snapshot.
func (f *Facade) Commit(ctx datastore.Context, ctrID, message string, tags []string) (string, error) {
	tenantID, err := f.dfs.Commit(ctrID)
	if err != nil {
		glog.Errorf("Could not commit container %s: %s", ctrID, err)
		return "", err
	}
	snapshotID, err := f.Snapshot(ctx, tenantID, message, tags)
	if err != nil {
		glog.Errorf("Could not snapshot %s: %s", tenantID, err)
		return "", err
	}
	return snapshotID, nil
}

// DeleteSnapshot removes a snapshot from an application.
func (f *Facade) DeleteSnapshot(ctx datastore.Context, snapshotID string) error {
	if err := f.dfs.Delete(snapshotID); err != nil {
		glog.Errorf("Could not delete snapshot %s: %s", snapshotID, err)
		return err
	}
	return nil
}

// DeleteSnapshots removes all snapshots for an application.
func (f *Facade) DeleteSnapshots(ctx datastore.Context, serviceID string) error {
	snapshots, err := f.ListSnapshots(ctx, serviceID)
	if err != nil {
		return err
	}
	for _, snapshotID := range snapshots {
		if err := f.DeleteSnapshot(ctx, snapshotID); err != nil {
			return err
		}
	}
	return nil
}

// DFSLock returns the locker for the dfs
func (f *Facade) DFSLock(ctx datastore.Context) sync.Locker {
	return f.dfs
}

// GetSnapshotInfo returns information about a snapshot.
func (f *Facade) GetSnapshotInfo(ctx datastore.Context, snapshotID string) (*dfs.SnapshotInfo, error) {
	info, err := f.dfs.Info(snapshotID)
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s: %s", snapshotID, err)
		return nil, err
	}
	return info, nil
}

// ListSnapshots returns a list of strings that describes the snapshots for the
// given application.
func (f *Facade) ListSnapshots(ctx datastore.Context, serviceID string) ([]string, error) {
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not find tenant for service %s: %s", serviceID, err)
		return nil, err
	}
	snapshots, err := f.dfs.List(tenantID)
	if err != nil {
		glog.Errorf("Could not list snapshots for tenant %s: %s", tenantID, err)
		return nil, err
	}
	return snapshots, nil
}

// ResetLock resets locks for a specific tenant
func (f *Facade) ResetLock(ctx datastore.Context, serviceID string) error {
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not find tenant for service %s: %s", serviceID, err)
		return err
	}
	mutex := getTenantLock(tenantID)
	mutex.Lock()
	if err := f.unlockTenant(ctx, tenantID); err != nil {
		return err
	}
	return nil
}

// ResetLocks resets all tenant locks
func (f *Facade) ResetLocks(ctx datastore.Context) error {
	tenantIDs, err := f.getTenantIDs(ctx)
	if err != nil {
		glog.Errorf("Could not get tenants: %s", err)
		return err
	}
	for _, tenantID := range tenantIDs {
		if err := f.ResetLock(ctx, tenantID); err != nil {
			return err
		}
	}
	return nil
}

// RepairRegistry will load "latest" from the docker registry and save it to the
// database.
func (f *Facade) RepairRegistry(ctx datastore.Context) error {
	tenantIDs, err := f.getTenantIDs(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
		if err != nil {
			return err
		}
		var imagesMap = make(map[string]struct{})
		for _, svc := range svcs {
			if _, ok := imagesMap[svc.ImageID]; !ok {
				if _, err := f.dfs.Download(svc.ImageID, tenantID, true); err != nil {
					glog.Errorf("Could not download image %s from registry: %s", svc.ImageID, err)
					return err
				}
				imagesMap[svc.ImageID] = struct{}{}
			}
		}
	}
	return nil
}

// Restore restores application data from a backup.
func (f *Facade) Restore(ctx datastore.Context, r io.Reader) error {
	glog.Infof("Beginning restore from backup")
	data, err := f.dfs.Restore(r)
	if err != nil {
		glog.Errorf("Could not restore from backup: %s", err)
		return err
	}
	if err := f.RestoreServiceTemplates(ctx, data.Templates); err != nil {
		glog.Errorf("Could not restore service templates from backup: %s", err)
		return err
	}
	glog.Infof("Loaded service templates")
	if err := f.RestoreResourcePools(ctx, data.Pools); err != nil {
		glog.Errorf("Could not restore resource pools from backup: %s", err)
		return err
	}
	glog.Infof("Loaded resource pools")
	if err := f.RestoreHosts(ctx, data.Hosts); err != nil {
		glog.Errorf("Could not restore hosts from backup: %s", err)
		return err
	}
	glog.Infof("Loaded hosts")
	for _, snapshot := range data.Snapshots {
		if err := f.Rollback(ctx, snapshot, false); err != nil {
			glog.Errorf("Could not rollback snapshot %s: %s", snapshot, err)
			return err
		}
		glog.Infof("Rolled back snapshot %s", snapshot)
	}
	glog.Infof("Completed restore from backup")
	return nil
}

// Rollback rolls back an application to state described in the provided
// snapshot.
func (f *Facade) Rollback(ctx datastore.Context, snapshotID string, force bool) error {
	glog.Infof("Beginning rollback of snapshot %s", snapshotID)
	info, err := f.dfs.Info(snapshotID)
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s: %s", snapshotID, err)
		return err
	}
	if err := f.lockTenant(ctx, info.TenantID); err != nil {
		glog.Errorf("Could not lock tenant %s: %s", info.TenantID, err)
		return err
	}
	defer f.retryUnlockTenant(ctx, info.TenantID, nil, time.Second)
	glog.Infof("Checking states for services under %s", info.TenantID)
	svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: info.TenantID})
	if err != nil {
		glog.Errorf("Could not get services under %s: %s", info.TenantID, err)
		return err
	}
	serviceids := make([]string, len(svcs))
	for i, svc := range svcs {
		if svc.DesiredState != int(service.SVCStop) {
			if force {
				defer f.ScheduleService(ctx, svc.ID, false, service.DesiredState(svc.DesiredState))
				if _, err := f.scheduleService(ctx, svc.ID, false, service.SVCStop, true); err != nil {
					glog.Errorf("Could not %s service %s (%s): %s", service.SVCStop, svc.Name, svc.ID, err)
					return err
				}
			} else {
				glog.Errorf("Could not rollback to snapshot %s: service %s (%s) is running", snapshotID, svc.Name, svc.ID)
				return errors.New("service is running")
			}
		}
		serviceids[i] = svc.ID
	}
	if err := f.WaitService(ctx, service.SVCStop, f.dfs.Timeout(), serviceids...); err != nil {
		glog.Errorf("Could not wait for services to %s during rollback of snapshot %s: %s", service.SVCStop, snapshotID, err)
		return err
	}
	glog.Infof("Services are all stopped, reverting service data")
	if err := f.RestoreServices(ctx, info.TenantID, info.Services); err != nil {
		glog.Errorf("Could not restore services: %s", err)
		return err
	}
	glog.Infof("Service data is rolled back, now restoring disk and images")
	if err := f.dfs.Rollback(snapshotID); err != nil {
		glog.Errorf("Could not rollback snapshot %s: %s", snapshotID, err)
		return err
	}
	glog.Infof("Successfully restored application data from snapshot %s", snapshotID)
	return nil
}

// Snapshot takes a snapshot for a particular application.
func (f *Facade) Snapshot(ctx datastore.Context, serviceID, message string, tags []string) (string, error) {
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not get tenant id of service %s: %s", serviceID, err)
		return "", err
	}
	if err := f.lockTenant(ctx, tenantID); err != nil {
		glog.Errorf("Could not lock tenant %s: %s", tenantID, err)
		return "", err
	}
	defer f.retryUnlockTenant(ctx, tenantID, nil, time.Second)
	glog.Infof("Checking states for services under %s", tenantID)
	svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
	if err != nil {
		glog.Errorf("Could not get services under %s: %s", tenantID, err)
		return "", err
	}
	imagesMap := make(map[string]struct{})
	images := make([]string, 0)
	serviceids := make([]string, len(svcs))
	for i, svc := range svcs {
		if svc.DesiredState == int(service.SVCRun) {
			defer f.ScheduleService(ctx, svc.ID, false, service.DesiredState(svc.DesiredState))
			if _, err := f.scheduleService(ctx, svc.ID, false, service.SVCPause, true); err != nil {
				glog.Errorf("Could not %s service %s (%s): %s", service.SVCPause, svc.Name, svc.ID, err)
				return "", err
			}
		}
		serviceids[i] = svc.ID
		if _, ok := imagesMap[svc.ImageID]; !ok {
			imagesMap[svc.ImageID] = struct{}{}
			images = append(images, svc.ImageID)
		}
	}
	if err := f.WaitService(ctx, service.SVCPause, f.dfs.Timeout(), serviceids...); err != nil {
		glog.Errorf("Could not wait for services to %s during snapshot of %s: %s", service.SVCStop, tenantID, err)
		return "", err
	}
	glog.Infof("Services are now paused, capturing state")
	data := dfs.SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: tenantID,
			Message:  message,
			Tags:     tags,
		},
		Services: svcs,
		Images:   images,
	}
	snapshotID, err := f.dfs.Snapshot(data)
	if err != nil {
		glog.Errorf("Could not snapshot disk and images for tenant %s: %s", tenantID, err)
		return "", err
	}
	glog.Infof("Successfully captured application data from tenant %s and created snapshot %s", tenantID, snapshotID)
	return snapshotID, nil
}
