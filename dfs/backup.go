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
// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package dfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

const (
	imageDir     = "images"
	snapshotDir  = "snapshots"
	templateJSON = "templates.json"
	serviceJSON  = "services.json"
	imageJSON    = "images.json"
)

func (dfs *DistributedFilesystem) Backup(dirpath string) (string, error) {
	dfs.log("Starting backup")

	// get the full path of the backup
	name := time.Now().Format("backup-2006-01-02-150405")
	if dirpath == "" {
		dirpath = filepath.Join(getHome(), "backups")
	}
	filename := filepath.Join(dirpath, fmt.Sprintf("%s.tgz", name))
	dirpath = filepath.Join(dirpath, name)

	if err := mkdir(dirpath); err != nil {
		glog.Errorf("Could neither find nor create %s: %v", dirpath, err)
		return "", err
	}

	defer func() {
		if err := os.RemoveAll(dirpath); err != nil {
			glog.Errorf("Could not remove %s: %v", dirpath, err)
		}
	}()

	for _, dir := range []string{imageDir, snapshotDir} {
		p := filepath.Join(dirpath, dir)
		if err := mkdir(p); err != nil {
			glog.Errorf("Could not create %s: %s", p, err)
			return "", err
		}
	}

	// retrieve services
	svcs, err := dfs.facade.GetServices(datastore.Get(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not get services: %s", err)
		return "", err
	}

	// export all template definitions
	dfs.log("Exporting template definitions")
	templates, err := dfs.facade.GetServiceTemplates(datastore.Get())
	if err != nil {
		glog.Errorf("Could not get service templates: %s", err)
		return "", err
	}
	if err := exportJSON(filepath.Join(dirpath, templateJSON), templates); err != nil {
		glog.Errorf("Could not export service templates: %s", err)
		return "", err
	}
	dfs.log("Template definition export successful")

	// export all of the docker images
	dfs.log("Exporting docker images")
	imageTags, err := dfs.exportImages(filepath.Join(dirpath, imageDir), templates, svcs)
	if err != nil {
		glog.Errorf("Could not export docker images: %s", err)
		return "", err
	}
	if err := exportJSON(filepath.Join(dirpath, imageJSON), &imageTags); err != nil {
		glog.Errorf("Could not export images: %s", err)
		return "", err
	}
	dfs.log("Docker image export successful")

	// export snapshots
	for _, svc := range svcs {
		if svc.ParentServiceID == "" {
			dfs.log("Exporting snapshots for %s (%s)", svc.Name, svc.ID)
			_, err := dfs.exportSnapshots(filepath.Join(dirpath, snapshotDir), &svc)
			if err != nil {
				glog.Errorf("Could not export snapshot for %s (%s): %s", svc.Name, svc.ID, err)
				return "", err
			}
			dfs.log("Exporting of %s (%s) snapshots successful", svc.Name, svc.ID)
		}
	}

	dfs.log("Writing backup file")
	if err := exportTGZ(dirpath, filename); err != nil {
		glog.Errorf("Could not write backup file %s: %s", filename, err)
		return "", err
	}
	dfs.log("Backup file created: %s", filename)

	return filename, nil
}

func (dfs *DistributedFilesystem) Restore(filename string) error {
	// fail if any services are running
	dfs.log("Checking running services")
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
			glog.Errorf("Cannot restore to %s: %s", filename, err)
			return err
		}
	}

	var reloadLogstashContainer bool
	defer func() {
		if reloadLogstashContainer {
			go facade.LogstashContainerReloader(datastore.Get(), dfs.facade) // don't block main thread
		}
	}()

	dirpath := filepath.Join(getHome(), "restore")
	if err := os.RemoveAll(dirpath); err != nil {
		glog.Errorf("Could not remove %s: %s", dirpath, err)
		return err
	}

	if err := mkdir(dirpath); err != nil {
		glog.Errorf("Could neither find nor create %s: %s", dirpath, err)
		return err
	}

	defer func() {
		if err := os.RemoveAll(dirpath); err != nil {
			glog.Warningf("Could not remove %s: %s", dirpath, err)
		}
	}()

	dfs.log("Extracting backup file %s", filename)
	if err := importTGZ(dirpath, filename); err != nil {
		glog.Errorf("Could not expand %s to %s: %s", filename, dirpath, err)
		return err
	}

	var templates map[string]servicetemplate.ServiceTemplate
	if err := importJSON(filepath.Join(dirpath, templateJSON), &templates); err != nil {
		glog.Errorf("Could not read templates from %s: %s", filename, err)
		return err
	}

	// restore the service templates
	dfs.log("Loading service templates")
	for templateID, template := range templates {
		glog.V(1).Infof("Restoring service template %s", templateID)
		template.ID = templateID
		if err := dfs.facade.UpdateServiceTemplate(datastore.Get(), template); err != nil {
			glog.Errorf("Could not restore template %s: %s", templateID, err)
			return err
		}
		reloadLogstashContainer = true
	}
	dfs.log("Service template load successful")

	// Get the tenant of all the services to be restored
	snapshotFiles, err := ls(filepath.Join(dirpath, snapshotDir))
	if err != nil {
		glog.Errorf("Could not list contents of %s: %s", filepath.Join(dirpath, snapshotDir), err)
		return err
	}
	tenantIDs := make(map[string]struct{})
	for _, f := range snapshotFiles {
		tenantID, _, err := parseLabel(strings.TrimSuffix(f, ".tgz"))
		if err != nil {
			glog.Errorf("Cannot restore %s: %s", f, err)
			return err
		}
		tenantIDs[tenantID] = struct{}{}
	}

	// restore docker images
	dfs.log("Loading docker images")
	var images [][]string
	if err := importJSON(filepath.Join(dirpath, imageJSON), &images); err != nil {
		glog.Errorf("Could not read images from %s: %s", filename, err)
		return err
	}

	if err := dfs.importImages(filepath.Join(dirpath, imageDir), images, tenantIDs); err != nil {
		glog.Errorf("Could not import images from %s: %s", filename, err)
		return err
	}
	dfs.log("Docker image load successful")

	// restore snapshots
	glog.V(1).Infof("Restoring services and snapshots")
	for _, f := range snapshotFiles {
		dfs.log("Loading %s", f)
		if err := dfs.importSnapshots(filepath.Join(dirpath, snapshotDir, f)); err != nil {
			glog.Errorf("Could not import snapshot from %s: %s", f, err)
			return err
		}
		dfs.log("Successfully loaded %s", f)
	}

	return nil
}

func (dfs *DistributedFilesystem) exportSnapshots(dirpath string, tenant *service.Service) (string, error) {
	glog.V(1).Infof("Exporting %s", tenant.ID)
	snapshotID, err := dfs.Snapshot(tenant.ID)
	if err != nil {
		glog.Errorf("Could not snapshot service %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}

	defer func() {
		// delete the snapshot
		if err := dfs.DeleteSnapshot(snapshotID); err != nil {
			glog.Warningf("Could not delete snapshot %s while backing up %s (%s): %s", snapshotID, tenant.Name, tenant.ID, err)
		}
	}()

	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not acquire the volume for %s (%s): %s", tenant.Name, tenant.ID, err)
		return "", err
	}
	src := snapshotVolume.SnapshotPath(snapshotID)
	tgz := filepath.Join(dirpath, fmt.Sprintf("%s.tgz", snapshotID))

	if err := exportTGZ(src, tgz); err != nil {
		glog.Errorf("Could not write %s to %s: %s", src, tgz, err)
		return "", err
	}

	return tgz, nil
}

func (dfs *DistributedFilesystem) importSnapshots(filename string) error {
	dirpath, name := filepath.Split(filename)
	snapshotID := strings.TrimSuffix(name, ".tgz")
	tenantID, _, err := parseLabel(snapshotID)
	if err != nil {
		glog.Errorf("Cannot restore snapshot %s: %s", filename, err)
		return err
	}

	glog.V(1).Infof("Importing %s", tenantID)
	if err := importTGZ(filepath.Join(dirpath, snapshotID), filename); err != nil {
		glog.Errorf("Could not extract %s: %s", filename, err)
		return err
	}

	// Load the services
	var svcs []*service.Service
	if err := importJSON(filepath.Join(dirpath, snapshotID, serviceJSON), &svcs); err != nil {
		glog.Errorf("Could not acquire services from %s: %s", snapshotID, err)
		return err
	}

	// Restore the service data
	if err := dfs.restoreServices(svcs); err != nil {
		glog.Errorf("Could not restore services from %s: %s", filename, err)
		return err
	}

	tenant, err := dfs.facade.GetService(datastore.Get(), tenantID)
	if err != nil {
		glog.Errorf("Could not find service %s for snapshot %s: %s", tenantID, snapshotID, err)
		return err
	}

	snapshotVolume, err := dfs.GetVolume(tenant)
	if err != nil {
		glog.Errorf("Could not acquire the volume for %s (%s): %s", tenant.Name, tenant.ID, err)
		return err
	}
	if err := os.Rename(filepath.Join(dirpath, snapshotID), snapshotVolume.SnapshotPath(snapshotID)); err != nil {
		glog.Errorf("Could not move snapshot volume: %s", err)
		return err
	}

	defer func() {
		if err := dfs.DeleteSnapshot(snapshotID); err != nil {
			glog.Warningf("Could not delete snapshot while restoring %s (%s): %s", snapshotID, tenant.Name, tenant.ID, err)
		}
	}()

	if err := dfs.Rollback(snapshotID); err != nil {
		glog.Errorf("Could not rollback to snapshot %s: %s", snapshotID, err)
		return err
	}

	//TODO: garbage collect (http://jimhoskins.com/2013/07/27/remove-untagged-docker-images.html)
	return nil
}
