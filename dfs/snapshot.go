// Copyright 2014 The Serviced Authors.
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

package dfs

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

const (
	snapshotMeta = "snapshot.json"
	timeFormat   = "20060102-150405"
)

type SnapshotMetadata struct {
	Description string
}

// Snapshot takes a snapshot of the dfs as well as the docker images for the
// given service ID
func (dfs *DistributedFilesystem) Snapshot(tenantID string, description string) (string, error) {
	// Get the tenant (parent) service
	tenant, err := dfs.facade.GetService(dfs.datastoreGet(), tenantID)
	if err != nil {
		glog.Errorf("Could not get service %s: %s", tenantID, err)
		return "", err
	} else if tenant == nil {
		err = fmt.Errorf("service not found")
		glog.Errorf("Service %s not found", tenantID)
		return "", err
	}

	// Pause all running services for that tenant
	svcs, err := dfs.facade.GetServices(dfs.datastoreGet(), dao.ServiceRequest{TenantID: tenantID})
	if err != nil {
		glog.Errorf("Could not get all services: %s", err)
		return "", err
	}

	serviceIDs := make([]string, len(svcs))
	for i, svc := range svcs {
		// Check if each service is paused (or stopped)
		if svc.DesiredState != int(service.SVCPause) && svc.DesiredState != int(service.SVCStop) {
			// Restore the service state when the function exits
			defer dfs.facade.ScheduleService(dfs.datastoreGet(), svc.ID, false, service.DesiredState(svc.DesiredState))
			// Set the service to "paused"
			glog.V(1).Infof("Scheduling service %s (%s) to %s", svc.Name, svc.ID, service.SVCPause)
			if _, err := dfs.facade.ScheduleService(dfs.datastoreGet(), svc.ID, false, service.SVCPause); err != nil {
				glog.Errorf("Could not %s service %s (%s): %s", service.SVCPause, svc.Name, svc.ID, err)
				return "", err
			}
		}
		// Add the service ID to the list of services that we need to watch
		serviceIDs[i] = svc.ID
	}

	if err := dfs.facade.WaitService(dfs.datastoreGet(), service.SVCPause, dfs.timeout, serviceIDs...); err != nil {
		glog.Errorf("Error while waiting for services to %s: %s", service.SVCPause, err)
		return "", err
	}

	// create the snapshot
	snapshotVolume, err := dfs.GetVolume(tenant.ID)
	if err != nil {
		glog.Errorf("Could not acquire the snapshot volume for %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	// dump the service definitions
	if err := exportJSON(filepath.Join(snapshotVolume.Path(), serviceJSON), svcs); err != nil {
		glog.Errorf("Could not export existing services at %s: %s", snapshotVolume.Path(), err)
		return "", err
	}

	// dump metadata for this snapshot
	if err := exportJSON(filepath.Join(snapshotVolume.Path(), snapshotMeta), SnapshotMetadata{description}); err != nil {
		glog.Errorf("Could not export %s: %s", snapshotMeta, err)
		return "", err
	}

	tagID := time.Now().UTC().Format(timeFormat)
	label := fmt.Sprintf("%s_%s", tenantID, tagID)

	// add the snapshot to the volume
	if err := snapshotVolume.Snapshot(label); err != nil {
		glog.Errorf("Could not snapshot service %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	// tag all of the images
	if err := tag(tenantID, docker.DockerLatest, tagID); err != nil {
		glog.Errorf("Could not tag new snapshot for %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	glog.Infof("Snapshot succeeded for tenantID:%s with fsType:%s using snapshotID:%s", tenantID, dfs.fsType, label)
	return label, nil
}

// Rollback rolls back the dfs and docker images to the state of a given snapshot
func (dfs *DistributedFilesystem) Rollback(snapshotID string, forceRestart bool) error {
	tenantID, timestamp, err := parseLabel(snapshotID)
	if err != nil {
		glog.Errorf("Could not rollback snapshot %s: %s", snapshotID, err)
		return err
	}

	svcs, err := dfs.facade.GetServices(dfs.datastoreGet(), dao.ServiceRequest{TenantID: tenantID})
	if err != nil {
		glog.Errorf("Could not acquire the list of all services: %s", err)
		return err
	}

	if forceRestart {
		// wait for all services to stop
		serviceIDs := make([]string, len(svcs))
		for i, svc := range svcs {
			// If the service's desired state is not stopped, set the state as such
			if svc.DesiredState != int(service.SVCStop) {
				// restore the service's desired state when this function exits
				defer dfs.facade.ScheduleService(dfs.datastoreGet(), svc.ID, false, service.DesiredState(svc.DesiredState))
				// set the service to "stop"
				glog.V(1).Infof("Scheduling service %s (%s) to %s", svc.Name, svc.ID, service.SVCStop)
				if _, err := dfs.facade.ScheduleService(dfs.datastoreGet(), svc.ID, false, service.SVCStop); err != nil {
					glog.Errorf("Could not %s service %s (%s): %s", service.SVCPause, svc.Name, svc.ID, err)
					return err
				}
			}
			// Add the service to the list of service IDs we need to watch
			serviceIDs[i] = svc.ID
		}

		// Wait for all service instances to reach the desired state
		if err := dfs.facade.WaitService(dfs.datastoreGet(), service.SVCStop, dfs.timeout, serviceIDs...); err != nil {
			glog.Errorf("Error while waiting for services to %s: %s", service.SVCStop, err)
			return err
		}
	} else {
		// error if services aren't stopped
		for _, svc := range svcs {
			if states, err := dfs.facade.GetServiceStates(dfs.datastoreGet(), svc.ID); err != nil {
				glog.Errorf("Could not look up service states for %s (%s): %s", svc.Name, svc.ID, err)
				return err
			} else if running := len(states); running > 0 {
				err := fmt.Errorf("service %s (%s) has %d running instances", svc.Name, svc.ID, running)
				glog.Errorf("Cannot rollback to %s: %s", snapshotID, err)
				return err
			}
		}
	}

	// check the snapshot
	tenant, err := dfs.facade.GetService(dfs.datastoreGet(), tenantID)
	if err != nil {
		glog.Errorf("Could not find service %s: %s", tenantID, err)
		return err
	} else if tenant == nil {
		glog.Errorf("Service %s not found", tenantID)
		return fmt.Errorf("service not found")
	}

	snapshotVolume, err := dfs.GetVolume(tenant.ID)
	if err != nil {
		glog.Errorf("Could not find volume for service %s: %s", tenantID, err)
		return err
	}

	err = func() error {
		if err := dfs.networkDriver.Stop(); err != nil {
			glog.Warningf("Could not stop network driver: %s", err)
		}
		defer dfs.networkDriver.Restart()
		// rollback the dfs
		glog.V(0).Infof("Performing rollback for %s (%s) using %s", tenant.Name, tenant.ID, snapshotID)
		if err := snapshotVolume.Rollback(snapshotID); err != nil {
			glog.Errorf("Error while trying to roll back to %s: %s", snapshotID, err)
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	// restore the tags
	glog.V(0).Infof("Restoring image tags for %s", snapshotID)
	if err := tag(tenantID, timestamp, docker.DockerLatest); err != nil {
		glog.Errorf("Could not restore snapshot tags for %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}

	// restore services
	var restore []*service.Service
	if err := importJSON(filepath.Join(snapshotVolume.SnapshotMetadataPath(snapshotID), serviceJSON), &restore); err != nil {
		glog.Errorf("Could not acquire services from %s: %s", snapshotID, err)
		return err
	}

	if err := dfs.restoreServices(tenantID, restore); err != nil {
		glog.Errorf("Could not restore services from %s: %s", snapshotID, err)
		return err
	}

	glog.Infof("Rollback succeeded for tenantID:%s with fsType:%s using snapshotID:%s", tenant.ID, dfs.fsType, snapshotID)
	return nil
}

// ListSnapshots lists all the snapshots for a particular tenant
func (dfs *DistributedFilesystem) ListSnapshots(tenantID string) ([]dao.SnapshotInfo, error) {

	// Get the tenant (parent) service
	tenant, err := dfs.facade.GetService(dfs.datastoreGet(), tenantID)
	if err != nil {
		glog.Errorf("Could not get service %s: %s", tenantID, err)
		return nil, err
	} else if tenant == nil {
		glog.Errorf("Service %s not found", tenantID)
		return nil, fmt.Errorf("service not found")
	}

	snapshotVolume, err := dfs.GetVolume(tenant.ID)
	if err != nil {
		glog.Errorf("Could not find volume for service %s (%s): %s", tenant.Name, tenant.ID, err)
		return nil, err
	}

	snapshotIDs, err := snapshotVolume.Snapshots()
	if err != nil {
		return nil, err
	}

	snapshots := make([]dao.SnapshotInfo, len(snapshotIDs))
	for i, snapshotID := range snapshotIDs {
		var description string
		var metadata *SnapshotMetadata
		if err := importJSON(filepath.Join(snapshotVolume.SnapshotMetadataPath(snapshotID), snapshotMeta), &metadata); err != nil {
			description = ""
		} else {
			description = metadata.Description
		}
		snapshots[i] = dao.SnapshotInfo{snapshotID, description}
	}

	return snapshots, err
}

// DeleteSnapshot deletes an existing snapshot as identified by its snapshotID
func (dfs *DistributedFilesystem) DeleteSnapshot(snapshotID string) error {
	tenantID, timestamp, err := parseLabel(snapshotID)
	if err != nil {
		glog.Errorf("Could not parse snapshot ID %s: %s", snapshotID, err)
		return err
	}

	tenant, err := dfs.facade.GetService(dfs.datastoreGet(), tenantID)
	if err != nil {
		glog.Errorf("Service not found %s: %s", tenantID, err)
		return err
	} else if tenant == nil {
		glog.Errorf("Service %s not found", tenantID)
		return fmt.Errorf("service not found")
	}

	snapshotVolume, err := dfs.GetVolume(tenant.ID)
	if err != nil {
		glog.Errorf("Could not find the volume for service %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}

	// delete the snapshot
	if err := snapshotVolume.RemoveSnapshot(snapshotID); err != nil {
		glog.Errorf("Could not delete snapshot %s: %s", snapshotID, err)
		return err
	}

	// update the tags
	images, err := findImages(tenantID, timestamp)
	if err != nil {
		glog.Errorf("Could not find images for snapshot %s: %s", snapshotID, err)
		return err
	}

	for _, image := range images {
		glog.Infof("Removing image %s", image.ID)
		if err := image.Delete(); err != nil {
			glog.Warningf("Error while removing image %s: %s", image.ID, err)
		}
	}

	return nil
}

// DeleteSnapshots deletes all snapshots relating to a particular tenantID
func (dfs *DistributedFilesystem) DeleteSnapshots(tenantID string) error {
	// delete the snapshot subvolume
	driver, err := volume.GetDriver(dfs.fsType, dfs.varpath)
	if err != nil {
		glog.Errorf("Couldn't load the %s storage driver for %s", dfs.fsType, dfs.varpath)
	}
	if !driver.Exists(tenantID) {
		glog.Errorf("Could not find the volume for service %s: %s", tenantID, err)
		return err
	}

	err = func() error {
		if err := dfs.networkDriver.Stop(); err != nil {
			glog.Warningf("Could not stop network driver: %s", err)
		}
		defer dfs.networkDriver.Restart()
		if err := driver.Remove(tenantID); err != nil {
			glog.Errorf("Could not remove volume for service %s: %s", tenantID, err)
			return err
		}
		return nil
	}()
	if err != nil {
		return err
	}

	// delete images for that tenantID
	images, err := searchImagesByTenantID(tenantID)
	if err != nil {
		glog.Errorf("Error looking up images for %s: %s", tenantID, err)
		return err
	}

	for _, image := range images {
		if err := image.Delete(); err != nil {
			glog.Warningf("Could not delete image %s (%s): %s", image.ID, image.UUID, err)
		}
	}

	return nil
}

func (dfs *DistributedFilesystem) restoreServices(tenantID string, svcs []*service.Service) error {
	// get the resource pools
	pools, err := dfs.facade.GetResourcePools(dfs.datastoreGet())
	if err != nil {
		return err
	}
	poolMap := make(map[string]struct{})
	for _, pool := range pools {
		poolMap[pool.ID] = struct{}{}
	}

	// map parentServiceID to service
	serviceTree := make(map[string][]service.Service)
	for _, svc := range svcs {
		serviceTree[svc.ParentServiceID] = append(serviceTree[svc.ParentServiceID], *svc)
	}

	// map service id to service
	current, err := dfs.facade.GetServices(dfs.datastoreGet(), dao.ServiceRequest{TenantID: tenantID})
	if err != nil {
		glog.Errorf("Could not get services: %s", err)
		return err
	}

	currentServices := make(map[string]*service.Service)
	for i, svc := range current {
		currentServices[svc.ID] = &current[i]
	}

	// updates all of the services
	var traverse func(parentID string) error
	traverse = func(parentID string) error {
		for _, svc := range serviceTree[parentID] {
			serviceID := svc.ID
			svc.DatabaseVersion = 0
			svc.DesiredState = int(service.SVCStop)
			svc.ParentServiceID = parentID

			// update the image
			if svc.ImageID != "" {
				image, err := commons.ParseImageID(svc.ImageID)
				if err != nil {
					glog.Errorf("Invalid image %s for %s (%s): %s", svc.ImageID, svc.Name, svc.ID, err)
				}
				image.Host = dfs.dockerHost
				image.Port = dfs.dockerPort
				svc.ImageID = image.BaseName()
			}

			// check the pool
			if _, ok := poolMap[svc.PoolID]; !ok {
				glog.Warningf("Could not find pool %s for %s (%s). Setting pool to default.", svc.PoolID, svc.Name, svc.ID)
				svc.PoolID = "default"
			}

			if _, ok := currentServices[serviceID]; ok {
				glog.Infof("Updating service %s (%s)", svc.Name, svc.ID)
				if err := dfs.facade.UpdateService(dfs.datastoreGet(), svc); err != nil {
					glog.Errorf("Could not update service %s: %s", svc.ID, err)
					return err
				}
				delete(currentServices, serviceID)
			} else {
				glog.Infof("Adding service %s (%s)", svc.Name, svc.ID)
				if err := dfs.facade.AddService(dfs.datastoreGet(), svc); err != nil {
					glog.Errorf("Could not add service %s: %s", serviceID, err)
					return err
				}
			}

			// restore the address assignments
			if err := dfs.facade.RestoreIPs(dfs.datastoreGet(), svc); err != nil {
				glog.Warningf("Could not restore address assignments for service %s (%s): %s", svc.Name, svc.ID, err)
			}

			if err := traverse(serviceID); err != nil {
				return err
			}
		}
		return nil
	}

	if err := traverse(""); err != nil {
		glog.Errorf("Error while rolling back services: %s", err)
		return err
	}

	// delete remaining services hierarchically
	deleted := make(map[string]struct{}) // list of services that have been deleted

	// if the parent is to be deleted, then delete that first because that
	// will recursively delete the children and limit the number of calls to
	// the facade
	var rmsvc func(*service.Service) error
	rmsvc = func(s *service.Service) error {
		if _, ok := deleted[s.ID]; ok {
			// service has already been deleted
			return nil
		} else if p, ok := currentServices[s.ParentServiceID]; ok {
			// if the parent needs to be deleted, delete the parent first
			if err := rmsvc(p); err != nil {
				return err
			}
		} else {
			// otherwise just delete the node
			glog.Infof("Removing service %s (%s)", s.Name, s.ID)
			if err := dfs.facade.RemoveService(dfs.datastoreGet(), s.ID); err != nil {
				glog.Errorf("Could not remove service %s (%s): %s", s.Name, s.ID, err)
				return err
			}
		}
		// update the list of deleted services
		deleted[s.ID] = struct{}{}
		return nil
	}
	for _, svc := range currentServices {
		if err := rmsvc(svc); err != nil {
			return err
		}
	}

	return nil
}

func parseLabel(snapshotID string) (string, string, error) {
	parts := strings.SplitN(snapshotID, "_", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed label")
	}
	return parts[0], parts[1], nil
}

func exportJSON(filename string, v interface{}) error {
	file, err := os.Create(filename)
	if err != nil {
		glog.Errorf("Could not create file at %s: %v", filename, err)
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			glog.Warningf("Error while closing file %s: %s", filename, err)
		}
	}()

	encoder := json.NewEncoder(file)
	if err := encoder.Encode(v); err != nil {
		glog.Errorf("Could not write JSON data to %s: %s", filename, err)
		return err
	}

	return nil
}

func importJSON(filename string, v interface{}) error {
	file, err := os.Open(filename)
	if err != nil {
		glog.Errorf("Could not open file at %s: %v", filename, err)
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			glog.Warningf("Error while closing file %s: %s", filename, err)
		}
	}()

	decoder := json.NewDecoder(file)
	if err := decoder.Decode(v); err != nil {
		glog.Errorf("Could not read JSON data from %s: %s", filename, err)
		return err
	}

	return nil
}
