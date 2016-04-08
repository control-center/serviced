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
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/volume"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

const (
	currentRegistryVersion            = 2
	oldLocalRegistryPort              = "5001"
	oldLocalRegistryContainerNameBase = "cc-temp-registry-v%d"
	registryRootSubdir                = "docker-registry"
	upgradedMarkerFile                = "cc-upgraded"
)

type registryVersionInfo struct {
	version int
	rootDir string
	imageId string
}

var registryVersionInfos = map[int]registryVersionInfo{
	1: registryVersionInfo{
		1,
		"registry",
		"registry:0.9.1",
	},
	2: registryVersionInfo{
		2,
		"v2",
		"registry:2.2.0", // This is the registry we currently use--may change in the future (WILL USE ISVCS DOCKER REG)
	},
}

// Backup takes a backup of all installed applications
func (f *Facade) Backup(ctx datastore.Context, w io.Writer) error {
	// Do not DFSLock here, ControlPlaneDao does that

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
		Templates:     templates,
		BaseImages:    images,
		Pools:         pools,
		Hosts:         hosts,
		Snapshots:     snapshots,
		Timestamp:     stime,
		BackupVersion: 1,
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
	// Do not DFSLock here, ControlPlaneDao does that
	if err := f.dfs.Delete(snapshotID); err != nil {
		glog.Errorf("Could not delete snapshot %s: %s", snapshotID, err)
		return err
	}
	return nil
}

// DeleteSnapshots removes all snapshots for an application.
func (f *Facade) DeleteSnapshots(ctx datastore.Context, serviceID string) error {
	// Do not DFSLock here, ControlPlaneDao does that
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
func (f *Facade) DFSLock(ctx datastore.Context) dfs.DFSLocker {
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

// TagSnapshot adds tags to an existing snapshot
func (f *Facade) TagSnapshot(snapshotID string, tagName string) error {
	if err := f.dfs.Tag(snapshotID, tagName); err != nil {
		glog.Errorf("Could not add tag to snapshot %s: %s", snapshotID, err)
		return err
	}
	return nil
}

// RemoveSnapshotTag removes a specific tag from an existing snapshot
func (f *Facade) RemoveSnapshotTag(ctx datastore.Context, serviceID, tagName string) (string, error) {
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not find tenant for service %s: %s", serviceID, err)
		return "", err
	}
	snapshotID, err := f.dfs.Untag(tenantID, tagName)
	if err != nil {
		glog.Errorf("Could not remove tag %s from tenant %s: %s", tagName, snapshotID, err)
		return "", err
	}
	return snapshotID, nil
}

// GetSnapshotByServiceIDAndTag finds the existing snapshot for a given service with a specific tag
func (f *Facade) GetSnapshotByServiceIDAndTag(ctx datastore.Context, serviceID, tagName string) (*dfs.SnapshotInfo, error) {
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not find tenant for service %s: %s", serviceID, err)
		return nil, err
	}
	info, err := f.dfs.TagInfo(tenantID, tagName)
	if err != nil {
		glog.Errorf("Could not get info for snapshot tag %s: %s", tagName, err)
		return nil, err
	}
	return info, nil
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

// Download will push a specified image into the registry for the specified
// tenant
func (f *Facade) Download(imageID, tenantID string) error {
	if _, err := f.dfs.Download(imageID, tenantID, true); err != nil {
		return err
	}
	return nil
}

// RepairRegistry will load "latest" from the docker registry and save it to the
// database.
func (f *Facade) RepairRegistry(ctx datastore.Context) error {
	if err := f.DFSLock(ctx).LockWithTimeout("reset registry", userLockTimeout); err != nil {
		glog.Warningf("Cannot reset registry: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

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

// UpgradeRegistry adds the images to the registry index so that they will be pushed into the registry.
// If fromRegistryHost is not set, search for an old registry on the local host to upgrade.
// If force is true for a local registry, upgrade again even if previous upgrade was successful.
// (For a remote registry, the upgrade is always performed regardless of the value of the force parameter.)
func (f *Facade) UpgradeRegistry(ctx datastore.Context, fromRegistryHost string, force bool) error {
	if err := f.DFSLock(ctx).LockWithTimeout("migrate registry", userLockTimeout); err != nil {
		glog.Warningf("Cannot migrate registry: %s", err)
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	success := true // indicates a successful migration
	if fromRegistryHost == "" {
		// check if a local docker migration is needed
		isMigrated, previousVersion, err := f.getPreviousRegistryVersion()
		if err != nil {
			glog.Errorf("Could not determine the previous docker registry to migrate: %s", err)
			return err
		}
		if previousVersion == currentRegistryVersion {
			glog.Infof("No previous version of the docker registry exists; nothing to migrate")
			return nil
		}
		if !isMigrated || force {
			glog.Infof("Starting local docker registry v%d", previousVersion)
			oldRegistryCtr, err := f.startDockerRegistry(previousVersion, oldLocalRegistryPort)
			if err != nil {
				glog.Errorf("Could not start v%d docker registry at port %s: %s", previousVersion, oldLocalRegistryPort, err)
				return err
			}
			defer func() {
				if success {
					f.markLocalDockerRegistryUpgraded(previousVersion)
				}
				glog.Infof("Stopping docker registry v%d container %s", previousVersion, oldRegistryCtr.Name)
				if err := oldRegistryCtr.Stop(5 * time.Minute); err != nil {
					glog.Errorf("Could not stop docker registry v%d container %s: %s", previousVersion, oldRegistryCtr.Name, err)
				}
			}()
			fromRegistryHost = fmt.Sprintf("localhost:%s", oldLocalRegistryPort)
		} else {
			glog.Infof("Registry already migrated from v%d; no action required", previousVersion)
			return nil
		}
	}
	tenantIDs, err := f.getTenantIDs(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
		if err != nil {
			return err
		}
		if err := f.dfs.UpgradeRegistry(svcs, tenantID, fromRegistryHost, force); err != nil {
			success = false
			glog.Warningf("Could not upgrade registry for tenant %s: %s", tenantID, err)
		}
	}
	return nil
}

// startDockerRegistry returns the old docker registry container.
func (f *Facade) startDockerRegistry(version int, port string) (*docker.Container, error) {
	versionInfo := registryVersionInfos[version]
	return versionInfo.start(f.isvcsPath, port)
}

// getPreviousRegistryVersion returns the next previous version of the docker
// registry that needs migration and whether it has been previously migrated.
func (f *Facade) getPreviousRegistryVersion() (isMigrated bool, version int, err error) {
	for i := currentRegistryVersion - 1; i > 0; i-- {
		glog.Infof("Checking v%d docker registry", i)
		versionInfo := registryVersionInfos[i]
		registryPath := versionInfo.getStoragePath(f.isvcsPath)
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			glog.Errorf("Could not stat v%d registry path at %s: %s", i, registryPath, err)
			return false, 0, err
		}
		markerFilePath := versionInfo.getUpgradedMarkerPath(f.isvcsPath)
		if _, err := os.Stat(markerFilePath); os.IsNotExist(err) {
			return false, i, nil
		} else if err != nil {
			glog.Errorf("Could not stat v%d registry marker file at %s: %s", i, markerFilePath, err)
			return false, 0, err
		}
		return true, i, nil
	}
	return false, currentRegistryVersion, nil
}

// markLocalDockerRegistryUpgraded sets a marker file that will indicate
// whether a registry has been previously migrated.
func (f *Facade) markLocalDockerRegistryUpgraded(version int) error {
	versionInfo := registryVersionInfos[version]
	markerFilePath := versionInfo.getUpgradedMarkerPath(f.isvcsPath)
	if err := ioutil.WriteFile(markerFilePath, []byte{}, 0644); err != nil {
		glog.Errorf("Could not write marker file %s: %s", markerFilePath, err)
		return err
	}
	return nil
}

// Restore restores application data from a backup.
func (f *Facade) Restore(ctx datastore.Context, r io.Reader, backupInfo *dfs.BackupInfo) error {
	// Do not DFSLock here, ControlPlaneDao does that
	glog.Infof("Beginning restore from backup")
	if err := f.dfs.Restore(r, backupInfo); err != nil {
		glog.Errorf("Could not restore from backup: %s", err)
		return err
	}
	if err := f.RestoreServiceTemplates(ctx, backupInfo.Templates); err != nil {
		glog.Errorf("Could not restore service templates from backup: %s", err)
		return err
	}
	glog.Infof("Restored service templates")
	if err := f.RestoreResourcePools(ctx, backupInfo.Pools); err != nil {
		glog.Errorf("Could not restore resource pools from backup: %s", err)
		return err
	}
	glog.Infof("Restored resource pools")
	if err := f.RestoreHosts(ctx, backupInfo.Hosts); err != nil {
		glog.Errorf("Could not restore hosts from backup: %s", err)
		return err
	}
	glog.Infof("Loaded hosts")
	for _, snapshot := range backupInfo.Snapshots {
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
	// Do not DFSLock here, ControlPlaneDao does that
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
				defer f.scheduleService(ctx, svc.ID, false, service.DesiredState(svc.DesiredState), true)
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
	if err := f.WaitService(ctx, service.SVCStop, f.dfs.Timeout(), false, serviceids...); err != nil {
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
	// Do not DFSLock here, ControlPlaneDao does that
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
			defer f.scheduleService(ctx, svc.ID, false, service.DesiredState(svc.DesiredState), true)
			if _, err := f.scheduleService(ctx, svc.ID, false, service.SVCPause, true); err != nil {
				glog.Errorf("Could not %s service %s (%s): %s", service.SVCPause, svc.Name, svc.ID, err)
				return "", err
			}
		}
		serviceids[i] = svc.ID
		if svc.ImageID != "" {
			if _, ok := imagesMap[svc.ImageID]; !ok {
				imagesMap[svc.ImageID] = struct{}{}
				images = append(images, svc.ImageID)
			}
		}
	}
	if err := f.WaitService(ctx, service.SVCPause, f.dfs.Timeout(), false, serviceids...); err != nil {
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

func (info *registryVersionInfo) getStoragePath(isvcsRoot string) string {
	return filepath.Join(isvcsRoot, registryRootSubdir, info.rootDir)
}

func (info *registryVersionInfo) getUpgradedMarkerPath(isvcsRoot string) string {
	return filepath.Join(info.getStoragePath(isvcsRoot), upgradedMarkerFile)
}

func (info *registryVersionInfo) start(isvcsRoot string, hostPort string) (*docker.Container, error) {
	var err error

	containerName := fmt.Sprintf(oldLocalRegistryContainerNameBase, info.version)
	storagePath := info.getStoragePath(isvcsRoot)
	bindMount := fmt.Sprintf("%s:/tmp/registry", storagePath)
	portBindings := make(map[dockerclient.Port][]dockerclient.PortBinding)
	portBindings["5000/tcp"] = []dockerclient.PortBinding{dockerclient.PortBinding{HostPort: hostPort}}
	url := fmt.Sprintf("http://localhost:%s/", hostPort)

	// See if container for old registry already exists
	container, err := docker.FindContainer(containerName)
	if err != nil {
		if err != docker.ErrNoSuchContainer {
			glog.Errorf("Could not look up container %s: %s", containerName, err)
			return nil, err
		}

		// Not found, so make a new one
		containerDefinition := &docker.ContainerDefinition{
			dockerclient.CreateContainerOptions{
				Name: containerName,
				Config: &dockerclient.Config{
					User:       "root",
					WorkingDir: "/tmp/registry",
					Image:      info.imageId,
					Env:        []string{"SETTINGS_FLAVOR=local"},
				},
				// HostConfig: &dockerclient.HostConfig{
				// 	Binds:        []string{bindMount},
				// 	PortBindings: portBindings,
				// },
			},
			dockerclient.HostConfig{},
		}

		glog.Infof("Creating container %s from image %s", containerDefinition.Name, containerDefinition.Config.Image)
		container, err = docker.NewContainer(containerDefinition, false, 0, nil, nil)
		if err != nil {
			glog.Errorf("Error trying to create container %s: %s", containerDefinition.Name, err)
			return nil, err
		}
	}

	os.MkdirAll(storagePath, 0755)

	// Make sure container is running
	container.HostConfig.Binds = []string{bindMount}
	container.HostConfig.PortBindings = portBindings
	glog.Infof("Starting container %s for Docker registry v%d at %s", container.Name, info.version, url)
	if err = container.Start(); err != nil {
		glog.Errorf("Could not start container %s: %s", container.Name, err)
		return nil, err
	}

	// Make sure registry is up and running (accepting connections)
	timeout := time.After(5 * time.Minute)
	for {
		resp, err := http.Get(url)
		if resp != nil && resp.Body != nil {
			resp.Body.Close()
		}
		if err == nil {
			break
		} else {
			glog.V(1).Infof("Waiting for Docker registry v%d to accept connections...", info.version)
		}

		select {
		case <-timeout:
			glog.Warningf("Timed out waiting for Docker registry v%d to accept connections", info.version)
			if err := container.Stop(5 * time.Minute); err != nil {
				glog.Errorf("After timeout, could not stop Docker registry v%d container %s: %s", info.version, container.Name, err)
			}
			return nil, errors.New(fmt.Sprintf("Timed out waiting for Docker registry v%d to accept connections", info.version))
		case <-time.After(time.Second):
		}
	}

	return container, nil
}

// DockerOverride will replace a docker image in the registry with a new image
func (f *Facade) DockerOverride(ctx datastore.Context, newImageName, oldImageName string) error {
	return f.dfs.Override(newImageName, oldImageName)
}
