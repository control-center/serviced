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
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

// Snapshot takes a snapshot of the dfs as well as the docker images for the
// given service ID
func (dfs *DistributedFilesystem) Snapshot(tenantID string) (string, error) {
	// Get the tenant (parent) service
	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Could not get service %s: %s", tenantID, err)
		return "", err
	}

	// Pause all running services
	svcs, err := dfs.facade.GetServices(datastore.Get(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not get all services: %s", err)
		return "", err
	}

	type status struct {
		id  string
		err error
	}

	processing := make(map[string]struct{})
	cancel := make(chan interface{})
	done := make(chan status)

	for _, svc := range svcs {
		if svc.DesiredState == service.SVCRun {
			defer dfs.facade.StartService(datastore.Get(), svc.ID)
			processing[svc.ID] = struct{}{}

			go func(poolID, serviceID string) {
				conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(poolID))
				if err != nil {
					glog.Errorf("Could not acquire connection to coordinator (%s): %s", poolID, err)
					done <- status{serviceID, err}
					return
				}

				err = dfs.pause(cancel, conn, serviceID)
				done <- status{serviceID, err}
			}(svc.PoolID, svc.ID)
		}
	}

	// wait for all services to pause
	timeout := time.After(dfs.timeout)
	if err := func() error {
		defer func() {
			close(cancel)
			for len(processing) > 0 {
				obj := <-done
				delete(processing, obj.id)
			}
		}()
		for len(processing) > 0 {
			select {
			case obj := <-done:
				delete(processing, obj.id)
				if obj.err != nil {
					return err
				}
			case <-timeout:
				return fmt.Errorf("request timed out")
			}
		}
		return nil
	}(); err != nil {
		glog.Errorf("Could not pause all running services: %s", err)
		return "", err
	}

	// create the snapshot
	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not acquire the snapshot volume for %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	tagID := time.Now().Format(node.TIMEFMT)
	label := fmt.Sprintf("%s_%s", tenantID, tagID)

	// add the snapshot to the volume
	if err := snapshotVolume.Snapshot(label); err != nil {
		glog.Errorf("Could not snapshot service %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	// tag all of the images
	if err := tag(tenantID, DockerLatest, tagID); err != nil {
		glog.Errorf("Could not tag new snapshot for %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	// dump the service definitions
	if err := exportJSON(filepath.Join(snapshotVolume.SnapshotPath(label), serviceJSON), getChildServices(tenantID, svcs)); err != nil {
		glog.Errorf("Could not export existing services at %s: %s", snapshotVolume.SnapshotPath(label), err)
		return "", err
	}

	return label, nil
}

// Rollback rolls back the dfs and docker images to the state of a given snapshot
func (dfs *DistributedFilesystem) Rollback(snapshotID string) error {
	tenantID, timestamp, err := parseLabel(snapshotID)
	if err != nil {
		glog.Errorf("Could not rollback snapshot %s: %s", snapshotID, err)
		return err
	}

	// fail if any services are running
	svcs, err := dfs.facade.GetServices(datastore.Get(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not acquire the list of all services: %s", err)
		return err
	}

	for _, svc := range svcs {
		conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
		if err != nil {
			glog.Errorf("Could not acquire connection to coordinator (%s): %s", svc.PoolID, err)
			return err
		}
		if states, err := zkservice.GetServiceStates(conn, svc.ID); err != nil {
			glog.Errorf("Could not look up running instances for %s (%s): %s", svc.Name, svc.ID, err)
			return err
		} else if running := len(states); running > 0 {
			err := fmt.Errorf("service %s (%s) has %d running instances", svc.Name, svc.ID, running)
			glog.Errorf("Cannot rollback to %s: %s", snapshotID, err)
			return err
		}
	}

	// check the snapshot
	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Could not find service %s: %s", tenantID, err)
		return err
	}

	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not find volume for service %s: %s", tenantID, err)
		return err
	}

	// rollback the dfs
	glog.V(0).Infof("Performing rollback for %s (%s) using %s", tenant.Name, tenant.ID, snapshotID)
	if err := snapshotVolume.Rollback(snapshotID); err != nil {
		glog.Errorf("Error while trying to roll back to %s: %s", snapshotID, err)
		return err
	}

	// restore the tags
	glog.V(0).Infof("Restoring image tags for %s", snapshotID)
	if err := tag(tenantID, timestamp, DockerLatest); err != nil {
		glog.Errorf("Could not restore snapshot tags for %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}

	// restore services
	var restore []*service.Service
	if err := importJSON(filepath.Join(snapshotVolume.SnapshotPath(snapshotID), serviceJSON), &restore); err != nil {
		glog.Errorf("Could not acquire services from %s: %s", snapshotID, err)
		return err
	}

	if err := dfs.restoreServices(restore); err != nil {
		glog.Errorf("Could not restore services from %s: %s", snapshotID, err)
		return err
	}

	return nil
}

// ListSnapshots lists all the snapshots for a particular tenant
func (dfs *DistributedFilesystem) ListSnapshots(tenantID string) ([]string, error) {
	// Get the tenant (parent) service
	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Could not get service %s: %s", tenantID, err)
		return nil, err
	}

	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not find volume for service %s (%s): %s", tenant.Name, tenant.ID, err)
		return nil, err
	}

	return snapshotVolume.Snapshots()
}

// DeleteSnapshot deletes an existing snapshot as identified by its snapshotID
func (dfs *DistributedFilesystem) DeleteSnapshot(snapshotID string) error {
	tenantID, timestamp, err := parseLabel(snapshotID)
	if err != nil {
		glog.Errorf("Could not parse snapshot ID %s: %s", snapshotID, err)
		return err
	}

	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Service not found %s: %s", tenantID, err)
		return err
	}

	snapshotVolume, err := dfs.GetVolume(tenant)
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
		imageID := image.ID
		imageID.Tag = timestamp
		img, err := docker.FindImage(imageID.String(), false)
		if err != nil {
			glog.Errorf("Could not remove tag from image %s: %s", imageID, err)
			continue
		}
		img.Delete()
	}

	return nil
}

// DeleteSnapshots deletes all snapshots relating to a particular tenantID
func (dfs *DistributedFilesystem) DeleteSnapshots(tenantID string) error {
	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Service not found %s: %s", tenantID, err)
		return err
	}

	// delete the snapshot subvolume
	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not find the volume for service %s (%s): %s")
		return err
	}
	if err := snapshotVolume.Unmount(); err != nil {
		glog.Errorf("Could not unmount volume for service %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}

	// delete the docker repos
	images, err := findImages(tenantID, DockerLatest)
	if err != nil {
		glog.Errorf("Could not find images for %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}

	for _, image := range images {
		img, err := docker.FindImage(image.ID.String(), false)
		if err != nil {
			glog.Errorf("Could not delete image %s: %s", image.ID, err)
			continue
		}
		img.Delete()
	}

	return nil
}

func (dfs *DistributedFilesystem) pause(cancel <-chan interface{}, conn client.Connection, serviceID string) error {
	if err := dfs.facade.PauseService(datastore.Get(), serviceID); err != nil {
		return err
	}

	states, err := zkservice.GetServiceStates(conn, serviceID)
	if err != nil {
		glog.Errorf("Could not get service states for service %s: %s", serviceID, err)
		return err
	}

	for _, state := range states {
		if err := zkservice.WaitPause(cancel, conn, serviceID, state.ID); err != nil {
			return fmt.Errorf("could not pause %s (%s): %s", serviceID, state.ID, err)
		}
		select {
		case <-cancel:
			return fmt.Errorf("request cancelled")
		default:
		}
	}

	return nil
}

func (dfs *DistributedFilesystem) restoreServices(svcs []*service.Service) error {
	// get the resource pools
	pools, err := dfs.facade.GetResourcePools(datastore.Get())
	if err != nil {
		return err
	}
	poolMap := make(map[string]struct{})
	for _, pool := range pools {
		poolMap[pool.ID] = struct{}{}
	}

	// map services to parent
	serviceTree := make(map[string][]service.Service)
	for _, svc := range svcs {
		serviceTree[svc.ParentServiceID] = append(serviceTree[svc.ParentServiceID], *svc)
	}

	// map service id to service
	current, err := dfs.facade.GetServices(datastore.Get(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not get services: %s", err)
		return err
	}

	currentServices := make(map[string]*service.Service)
	for _, svc := range current {
		currentServices[svc.ID] = &svc
	}

	// updates all of the services
	var traverse func(parentID string) error
	traverse = func(parentID string) error {
		for _, svc := range serviceTree[parentID] {
			serviceID := svc.ID
			svc.DatabaseVersion = 0
			svc.DesiredState = service.SVCStop
			svc.ParentServiceID = parentID

			// update the image
			if svc.ImageID != "" {
				image, err := commons.ParseImageID(svc.ImageID)
				if err != nil {
					glog.Errorf("Invalid image %s for %s (%s): %s", svc.ImageID, svc.Name, svc.ID)
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
				if err := dfs.facade.UpdateService(datastore.Get(), svc); err != nil {
					glog.Errorf("Could not update service %s: %s", svc.ID, err)
					return err
				}
				delete(currentServices, serviceID)
			} else {
				if err := dfs.facade.AddService(datastore.Get(), svc); err != nil {
					glog.Errorf("Could not add service %s: %s", serviceID, err)
					return err
				}

				/*
					// TODO: enable this to generate a new service ID, instead of recycling
					// the old one
					svc.ID = ""
					newServiceID, err = dfs.facade.AddService(svc)
					if err != nil {
						glog.Errorf("Could not add service %s: %s", serviceID, err)
						return err
					}

					// Update the service
					serviceTree[newServiceID] = serviceTree[serviceID]
					delete(serviceTree, serviceID)
					serviceID = newServiceID
				*/
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

	/*
		// TODO: enable this if we want to delete any non-matching services
		for serviceID := range currentServices {
			if err := dfs.facade.RemoveService(serviceID); err != nil {
				glog.Errorf("Could not remove service %s: %s", serviceID, err)
				return err
			}
		}
	*/

	return nil
}

func parseLabel(snapshotID string) (string, string, error) {
	parts := strings.SplitN(snapshotID, "_", 2)
	if len(parts) < 2 {
		return "", "", fmt.Errorf("malformed label")
	}
	return parts[0], parts[1], nil
}

func getChildServices(tenantID string, svcs []service.Service) []service.Service {
	var result []service.Service

	svcMap := make(map[string][]service.Service)
	for _, svc := range svcs {
		if svc.ID == tenantID {
			result = append(result, svc)
		} else {
			childSvcs := svcMap[svc.ParentServiceID]
			svcMap[svc.ParentServiceID] = append(childSvcs, svc)
		}
	}

	var walk func(root string)
	walk = func(root string) {
		for _, svc := range svcMap[root] {
			result = append(result, svc)
			walk(svc.ID)
		}
	}
	walk(tenantID)
	return result
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
