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
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

const (
	meta         = "METADATA"
	imageDir     = "images"
	snapshotDir  = "snapshots"
	poolJSON     = "resourcepools.json"
	hostJSON     = "hosts.json"
	templateJSON = "templates.json"
	serviceJSON  = "services.json"
	imageJSON    = "images.json"
)

type metadata struct {
	FSType string
}

// ListBackups lists all the backups in a given directory
func (dfs *DistributedFilesystem) ListBackups(dirpath string) ([]dao.BackupFile, error) {
	backups := make([]dao.BackupFile, 0)

	if dirpath = strings.TrimSpace(dirpath); dirpath == "" {
		dirpath = utils.BackupDir(dfs.varpath)
	} else {
		dirpath = filepath.Clean(dirpath)
	}

	// Validate the path
	if fp, err := os.Stat(dirpath); os.IsNotExist(err) {
		// Directory not found means no backups yet
		return backups, nil
	} else if err != nil {
		glog.Errorf("Error looking up path %s: %s", dirpath, err)
		return nil, err
	} else if !fp.IsDir() {
		return nil, fmt.Errorf("path is not a directory")
	}

	// Get the list of files
	files, err := ioutil.ReadDir(dirpath)
	if err != nil {
		return nil, err
	}

	// Get the ip address
	hostIP, err := utils.GetIPAddress()
	if err != nil {
		return nil, err
	}

	// Filter backup files
	filemap := make(map[string]dao.BackupFile)
	for _, file := range files {
		if file.IsDir() {
			// If it is a directory, it could be the directory file for a
			// backup tgz, so mark the file as InProgress
			backup := filemap[file.Name()]
			backup.InProgress = true
			filemap[file.Name()] = backup
		} else if ext := filepath.Ext(file.Name()); ext == ".tgz" {
			// Create the backup file object
			abspath := filepath.Join(dirpath, file.Name())
			name := strings.TrimSuffix(file.Name(), ext)
			backup := filemap[name]
			backup.FullPath = fmt.Sprintf("%s:%s", hostIP, abspath)
			backup.Name = abspath
			backup.Size = file.Size()
			backup.Mode = file.Mode()
			backup.ModTime = file.ModTime()
			filemap[name] = backup
		}

	}

	// Clean up non-backups
	for _, backup := range filemap {
		if backup.FullPath != "" {
			// Directories without a related backup file get filtered
			backups = append(backups, backup)
		}
	}

	return backups, nil
}

// Backup backs up serviced, saving the last n snapshots
func (dfs *DistributedFilesystem) Backup(dirpath string, last int) (string, error) {
	dfs.log("Starting backup")

	// get the full path of the backup
	name := time.Now().Format("backup-2006-01-02-150405")
	if dirpath = strings.TrimSpace(dirpath); dirpath == "" {
		dirpath = utils.BackupDir(dfs.varpath)
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

	if err := exportJSON(filepath.Join(dirpath, meta), &metadata{dfs.fsType}); err != nil {
		glog.Errorf("Could not export %s: %s", meta, err)
		return "", err
	}

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

	// export pools (and virtual ips)
	dfs.log("Exporting resource pools")
	pools, err := dfs.facade.GetResourcePools(datastore.Get())
	if err != nil {
		glog.Errorf("Could not get resource pools: %s", err)
		return "", err
	}
	if err := exportJSON(filepath.Join(dirpath, poolJSON), pools); err != nil {
		glog.Errorf("Could not export resource pools: %s", err)
		return "", err
	}
	dfs.log("Resource pool export successful")

	// export hosts
	dfs.log("Exporting hosts")
	hosts, err := dfs.facade.GetHosts(datastore.Get())
	if err != nil {
		glog.Errorf("Could not get hosts: %s", err)
		return "", err
	}
	if err := exportJSON(filepath.Join(dirpath, hostJSON), hosts); err != nil {
		glog.Errorf("Could not export hosts: %s", err)
		return "", err
	}
	dfs.log("Host export successful")

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

	// export snapshots
	var snapshots []string
	for _, svc := range svcs {
		if svc.ParentServiceID == "" {
			dfs.log("Exporting snapshots for %s (%s)", svc.Name, svc.ID)
			_, labels, err := dfs.saveSnapshots(svc.ID, filepath.Join(dirpath, snapshotDir), last)

			if err != nil {
				glog.Errorf("Could not export snapshot for %s (%s): %s", svc.Name, svc.ID, err)
				return "", err
			}
			snapshots = append(snapshots, labels...)
			dfs.log("Exporting of %s (%s) snapshots successful", svc.Name, svc.ID)
		}
	}

	// export all of the docker images
	dfs.log("Exporting docker images")
	imageTags, err := dfs.exportImages(filepath.Join(dirpath, imageDir), templates, svcs, snapshots)
	if err != nil {
		glog.Errorf("Could not export docker images: %s", err)
		return "", err
	}
	if err := exportJSON(filepath.Join(dirpath, imageJSON), &imageTags); err != nil {
		glog.Errorf("Could not export images: %s", err)
		return "", err
	}
	dfs.log("Docker image export successful")

	dfs.log("Writing backup file")
	if err := exportTGZ(dirpath, filename); err != nil {
		glog.Errorf("Could not write backup file %s: %s", filename, err)
		return "", err
	}
	dfs.log("Backup file created: %s", filename)

	glog.Infof("Backup succeeded with fsType:%s saved to file:%s", dfs.fsType, filename)
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

	dirpath := filepath.Join(utils.BackupDir(dfs.varpath), "restore")
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

	var metadata metadata
	if err := importJSON(filepath.Join(dirpath, meta), &metadata); err != nil {
		glog.Errorf("Could not import %s: %s", meta, err)
		return err
	} else if metadata.FSType != dfs.fsType {
		err = fmt.Errorf("this backup can only be run on %s", metadata.FSType)
		glog.Errorf("Could not extract backup %s: %s", filename, err)
		return err
	}

	var pools []pool.ResourcePool
	if err := importJSON(filepath.Join(dirpath, poolJSON), &pools); err != nil {
		glog.Errorf("Could not read resource pools from %s: %s", filename, err)
		return err
	}

	// restore the resource pools (and virtual ips)
	dfs.log("Loading resource pools")
	for _, pool := range pools {
		pool.DatabaseVersion = 0

		glog.V(1).Infof("Restoring resource pool %s", pool.ID)
		var err error
		if err = dfs.facade.AddResourcePool(datastore.Get(), &pool); err == facade.ErrPoolExists {
			err = dfs.facade.UpdateResourcePool(datastore.Get(), &pool)
		}
		if err != nil {
			glog.Errorf("Could not restore resource pool %s: %s", pool.ID, err)
			return err
		}
	}
	dfs.log("Resource pool load successful")

	var hosts []host.Host
	if err := importJSON(filepath.Join(dirpath, hostJSON), &hosts); err != nil {
		glog.Errorf("Could not read hosts from %s: %s", filename, err)
		return err
	}

	// restore the hosts (add it if it doesn't exist; assume hosts don't need to be updated from backup)
	dfs.log("Loading static ips")
	for _, host := range hosts {
		host.DatabaseVersion = 0
		glog.V(1).Infof("Restoring host %s (%s)", host.ID, host.IPAddr)
		if exists, err := dfs.facade.HasIP(datastore.Get(), host.PoolID, host.IPAddr); err != nil {
			glog.Errorf("Could not check IP %s for pool %s: %s", host.IPAddr, host.PoolID, err)
			return err
		} else if exists {
			glog.Warningf("Could not restore host %s (%s): ip already exists", host.ID, host.IPAddr)
		} else if err := dfs.facade.AddHost(datastore.Get(), &host); err != nil {
			glog.Errorf("Could not add host %s: %s", host.ID, err)
			return err
		}
	}
	dfs.log("Static ip load successful")

	var templates map[string]servicetemplate.ServiceTemplate
	if err := importJSON(filepath.Join(dirpath, templateJSON), &templates); err != nil {
		glog.Errorf("Could not read templates from %s: %s", filename, err)
		return err
	}

	// restore the service templates
	dfs.log("Loading service templates")
	for templateID, template := range templates {
		template.DatabaseVersion = 0

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
	tenantIDs := make(map[string]struct{})
	for _, f := range snapshotFiles {
		tenantIDs[strings.TrimSuffix(f, ".tgz")] = struct{}{}
	}

	// restore docker images
	dfs.log("Loading docker images")
	var images []imagemeta
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
		if err := dfs.loadSnapshots(strings.TrimSuffix(f, ".tgz"), filepath.Join(dirpath, snapshotDir, f)); err != nil {
			glog.Errorf("Could not import snapshot from %s: %s", f, err)
			return err
		}
		dfs.log("Successfully loaded %s", f)
	}

	glog.Infof("Restore succeeded with fsType:%s from file:%s", dfs.fsType, filename)
	return nil
}

func (dfs *DistributedFilesystem) saveSnapshots(tenantID, directory string, last int) (string, []string, error) {
	tmpdir := filepath.Join(directory, tenantID)

	if err := mkdir(tmpdir); err != nil {
		glog.Errorf("Could neither find nor create %s: %v", tmpdir, err)
		return "", nil, err
	}

	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			glog.Errorf("Could not remove %s: %v", tmpdir, err)
		}
	}()

	volume, err := dfs.GetVolume(tenantID)
	if err != nil {
		glog.Errorf("Could not get volume for %s: %s", tenantID, err)
		return "", nil, err
	}

	label, err := dfs.Snapshot(tenantID)
	if err != nil {
		glog.Errorf("Could not snapshot service %s: %s", tenantID, err)
		return "", nil, err
	}

	defer func() {
		// delete the snapshot
		if err := dfs.DeleteSnapshot(label); err != nil {
			glog.Warningf("Could not delete snapshot %s while backing up %s: %s", label, tenantID, err)
		}
	}()

	// Save the last n+1 snapshots
	snapshots, err := volume.Snapshots()
	if err != nil {
		glog.Errorf("Could not retrieve snapshots for %s: %s", tenantID, err)
		return "", nil, err
	}
	if count := len(snapshots); count > last+1 {
		snapshots = snapshots[count-(last+1):]
	}

	var parent string
	for i, snapshot := range snapshots {
		outfile := filepath.Join(tmpdir, fmt.Sprintf("%s.%d", label, i))
		if err := volume.Export(label, parent, outfile); err != nil {
			glog.Errorf("Could not export snapshot %s to %s: %s", label, outfile, err)
			return "", nil, err
		}
		parent = snapshot
	}

	exportfile := fmt.Sprintf("%s.tgz", tmpdir)
	if err := exportTGZ(tmpdir, exportfile); err != nil {
		glog.Errorf("Could not write to tar file for %s: %s", tenantID, err)
		return "", nil, err
	}

	return exportfile, snapshots, nil
}

func (dfs *DistributedFilesystem) loadSnapshots(tenantID, infile string) error {
	tmpdir := filepath.Join(filepath.Dir(infile), tenantID)

	if err := mkdir(tmpdir); err != nil {
		glog.Errorf("Could neither find nor create %s: %v", tmpdir, err)
		return err
	}

	defer func() {
		if err := os.RemoveAll(tmpdir); err != nil {
			glog.Errorf("Could not remove %s: %v", tmpdir, err)
		}
	}()

	volume, err := dfs.GetVolume(tenantID)
	if err != nil {
		glog.Errorf("Could not load volume for service %s: %s", tenantID, err)
		return err
	}

	if err := importTGZ(tmpdir, infile); err != nil {
		glog.Errorf("Could not read from tar file for %s: %s", tenantID, err)
		return err
	}

	snapshots, err := ioutil.ReadDir(tmpdir)
	if err != nil {
		glog.Errorf("Could not read snapshots for %s: %s", tenantID, err)
		return err
	} else if len(snapshots) == 0 {
		glog.Warningf("No snapshots to load")
		return nil
	}
	sort.Sort(FileInfoSlice(snapshots))

	// Import all of the snapshots
	var label string
	for _, snapshot := range snapshots {
		label = strings.TrimSuffix(snapshot.Name(), filepath.Ext(snapshot.Name()))
		if err := volume.Import(label, filepath.Join(tmpdir, snapshot.Name())); err != nil {
			glog.Warningf("Could not import snapshot %s: %s", label, err)
		}
	}

	defer func() {
		// delete the snapshot
		if err := dfs.DeleteSnapshot(label); err != nil {
			glog.Warningf("Could not delete snapshot %s while restoring %s: %s", label, tenantID, err)
		}
	}()

	// Load the services
	var svcs []*service.Service
	jsonfile := filepath.Join(volume.SnapshotPath(label), serviceJSON)
	if err := importJSON(jsonfile, &svcs); err != nil {
		glog.Errorf("Could not load services from %s: %s", label, err)
		return err
	}
	// Restore the service data
	if err := dfs.restoreServices(svcs); err != nil {
		glog.Errorf("Could not restore services from %s: %s", label, err)
		return err
	}
	// Rollback the snapshot
	if err := dfs.Rollback(label, false); err != nil {
		glog.Errorf("Could not rollback to snapshot %s: %s", label, err)
		return err
	}

	//TODO: garbage collect (http://jimhoskins.com/2013/07/27/remove-untagged-docker-images.html)
	return nil
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
	} else if tenant == nil {
		err = fmt.Errorf("service not found")
		glog.Errorf("Service %s not found", tenantID)
		return err
	}

	snapshotVolume, err := dfs.GetVolume(tenant.ID)
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

	if err := dfs.Rollback(snapshotID, false); err != nil {
		glog.Errorf("Could not rollback to snapshot %s: %s", snapshotID, err)
		return err
	}

	return nil
}

type FileInfoSlice []os.FileInfo

func (p FileInfoSlice) Len() int { return len(p) }
func (p FileInfoSlice) Less(i, j int) bool {
	return filepath.Ext(p[i].Name()) < filepath.Ext(p[j].Name())
}
func (p FileInfoSlice) Swap(i, j int) { p[i], p[j] = p[j], p[i] }
