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

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/statistics"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/volume"
	dockerclient "github.com/fsouza/go-dockerclient"
	"strings"
	"github.com/dustin/go-humanize"
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
func (f *Facade) Backup(ctx datastore.Context, w io.Writer, excludes []string, snapshotSpacePercent int) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.Backup"))
	// Do not DFSLock here, ControlPlaneDao does that

	stime := time.Now()
	message := fmt.Sprintf("started backup at %s", stime.UTC())
	plog.WithField("excludes", excludes).Info("Started backup")
	templates, images, err := f.GetServiceTemplatesAndImages(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get service templates and images")
		return err
	}
	plog.WithField("elapsed", time.Since(stime)).Info("Loaded templates and their images")
	pools, err := f.GetResourcePools(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get resource pools")
		return err
	}
	plog.WithField("elapsed", time.Since(stime)).Info("Loaded resource pools")
	tenants, err := f.GetTenantIDs(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get tenants")
		return err
	}
	snapshots := make([]string, len(tenants))
	snapshotExcludes := map[string][]string{}
	for i, tenant := range tenants {
		tenantLogger := plog.WithField("tenant", tenant)
		tag := fmt.Sprintf("backup-%s-%s", tenant, stime)
		snapshot, err := f.Snapshot(ctx, tenant, message, []string{tag}, snapshotSpacePercent)
		if err != nil {
			tenantLogger.WithError(err).Debug("Could not snapshot tenant")
			return err
		}
		defer f.DeleteSnapshot(ctx, snapshot)
		snapshots[i] = snapshot
		snapshotExcludes[snapshot] = append(excludes, f.getExcludedVolumes(ctx, tenant)...)
		tenantLogger.WithField("snapshot", snapshot).Info("Created a snapshot for tenant")
	}
	plog.WithField("elapsed", time.Since(stime)).Info("Loaded tenants")
	data := dfs.BackupInfo{
		Templates:        templates,
		BaseImages:       images,
		Pools:            pools,
		Snapshots:        snapshots,
		SnapshotExcludes: snapshotExcludes,
		Timestamp:        stime,
		BackupVersion:    1,
	}
	plog.WithField("data", data).Info("Calling dfs.Backup")
	if err := f.dfs.Backup(data, w); err != nil {
		plog.WithError(err).Debug("Could not backup")
		return err
	}
	plog.WithField("duration", time.Since(stime)).Info("Completed backup")
	return nil
}

// TODO: find a more suitable place for this
func DfPath(path string, excludes []string) (uint64, error) {
	plog.WithField("path", path).WithField("excludes", excludes).Info("Begin DfPath")
	var size uint64
	var fqexcludes []string
	for _, exc := range(excludes) {
		fqexcludes = append(fqexcludes, filepath.Join(path, exc))
	}
	err := filepath.Walk(path, func(walkpath string, info os.FileInfo, err error) error {
		for _, exclude := range(fqexcludes) {
			if strings.HasPrefix(walkpath,exclude)  {
				plog.WithField("walkpath", walkpath).WithField("info", info).Info("Excluding path from size count.")
				return filepath.SkipDir
			}
		}
		if !info.IsDir() {
			size += uint64(info.Size())
		}
		return err
	})
	return size, err
}


// EstimateBackup estimates storage requirements to take a backup of all installed applications
func (f *Facade) EstimateBackup(ctx datastore.Context, request dao.BackupRequest, estimate *dao.BackupEstimate) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.EstimateBackup"))

	stime := time.Now()
	plog.WithField("request", request).Info("Started backup estimate")

	options := config.GetOptions()

	// Do not DFSLock here, ControlPlaneDao does that

	estimate.BackupPath = request.Dirpath
	// Get Filesystem free space
	estimate.AvailableBytes = volume.FilesystemBytesAvailable(request.Dirpath)
	estimate.AvailableString = humanize.Bytes(estimate.AvailableBytes)

	plog.WithFields(logrus.Fields {
		"dirpath": request.Dirpath,
		"estimate": estimate,
	}).Info("Checked FilestystemSpaceAvailable")

	// Estimate bytes to backup from filesystem
	_, images, err := f.GetServiceTemplatesAndImages(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get service templates and images")
		return err
	}

	plog.WithField("elapsed", time.Since(stime)).Info("Loaded templates and their images")

	tenants, err := f.GetTenantIDs(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get tenants")
		return err
	}

	volumesPath := options.VolumesPath
	var FilesystemBytesRequired, DockerBytesRequired uint64

	plog.WithField("volpath", volumesPath).Info("Got VolumesPath")

	for _, tenant := range tenants {
		tenantPath := filepath.Join(volumesPath, tenant)
		tenantLogger := plog.WithFields(logrus.Fields{
			"tenant": tenant,
			"tenantPath": tenantPath,
		})
		tpsize, err := DfPath(tenantPath, request.Excludes)
		if err != nil {
			tenantLogger.WithError(err).Info("Could not get size for path.")
		}

		tenantLogger.WithFields(logrus.Fields{
			"elapsed":    time.Since(stime),
			"tpSize":     tpsize,
		}).Info("Tenant Size")
		FilesystemBytesRequired += tpsize

		tenantLogger.WithField("elapsed", time.Since(stime)).Info("path sized")
	}

	plog.WithField("elapsed", time.Since(stime)).Info("Estimated tenants")

	// Estimate Docker image bytes to backup
	size, err := f.dfs.GetImagePullSize(images)
	if err != nil {
		plog.WithError(err).Info("Could not get size for images.")
	}
	plog.WithField("elapsed", time.Since(stime)).Infof("Estimated Docker pull size at %d", size)
	DockerBytesRequired = size

	MinOverheadBytes, err := humanize.ParseBytes(options.BackupMinOverhead)
	if err != nil {
		plog.WithError(err).Info("Unable to get MinOverheadBytes")
	}
	CompressionEst := options.BackupEstimatedCompression
	TotalBytesRequired := FilesystemBytesRequired + DockerBytesRequired
	AdjustedBytesRequired := uint64(float64(TotalBytesRequired) / CompressionEst + 0.5) + MinOverheadBytes
	estimate.EstimatedBytes = AdjustedBytesRequired
	estimate.EstimatedString = humanize.Bytes(AdjustedBytesRequired)

	plog.WithFields(logrus.Fields {
		"elapsed":  time.Since(stime),
		"filesystembytes": FilesystemBytesRequired,
		"dockerbytes":    DockerBytesRequired,
		"totalbytes": estimate.EstimatedBytes,
		"BackupEstimatedCompression": CompressionEst,
		"BackupMinOverhead": options.BackupMinOverhead,
		"MinOverheadBytes": MinOverheadBytes,
	}).Info("Estimated sizes for images")


	plog.WithField("duration", time.Since(stime)).WithField("estimate", estimate).Info("Completed backup estimate")
	return nil
}

// BackupInfo returns metadata info about a backup
func (f *Facade) BackupInfo(ctx datastore.Context, r io.Reader) (*dfs.BackupInfo, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.BackupInfo"))
	info, err := f.dfs.BackupInfo(r)
	if err != nil {
		plog.WithError(err).Debug("Could not get info for backup")
		return nil, err
	}
	return info, nil
}

// Commit commits a container to the docker registry and takes a snapshot.
func (f *Facade) Commit(ctx datastore.Context, ctrID, message string, tags []string, snapshotSpacePercent int) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.Commit"))
	logger := plog.WithField("containerid", ctrID)
	tenantID, err := f.dfs.Commit(ctrID)
	if err != nil {
		logger.WithError(err).Debug("Could not commit container")
		return "", err
	}
	logger = logger.WithField("tenantid", tenantID)
	snapshotID, err := f.Snapshot(ctx, tenantID, message, tags, snapshotSpacePercent)
	if err != nil {
		logger.WithError(err).Debug("Could not snapshot tenant")
		return "", err
	}
	return snapshotID, nil
}

// DeleteSnapshot removes a snapshot from an application.
func (f *Facade) DeleteSnapshot(ctx datastore.Context, snapshotID string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DeleteSnapshot"))
	// Do not DFSLock here, ControlPlaneDao does that
	if err := f.dfs.Delete(snapshotID); err != nil {
		plog.WithField("snapshotid", snapshotID).WithError(err).Debug("Could not delete snapshot")
		return err
	}
	return nil
}

// DeleteSnapshots removes all snapshots for an application.
func (f *Facade) DeleteSnapshots(ctx datastore.Context, serviceID string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DeleteSnapshots"))
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DFSLock"))
	return f.dfs
}

// GetSnapshotInfo returns information about a snapshot.
func (f *Facade) GetSnapshotInfo(ctx datastore.Context, snapshotID string) (*dfs.SnapshotInfo, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetSnapshotInfo"))
	info, err := f.dfs.Info(snapshotID)
	if err != nil {
		plog.WithField("snapshotid", snapshotID).WithError(err).Debug("Could not get info for snapshot")
		return nil, err
	}
	return info, nil
}

// ListSnapshots returns a list of strings that describes the snapshots for the
// given application.
func (f *Facade) ListSnapshots(ctx datastore.Context, serviceID string) ([]string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ListSnapshots"))
	logger := plog.WithField("serviceid", serviceID)
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not find tenant for service")
		return nil, err
	}
	logger = logger.WithField("tenantid", tenantID)
	snapshots, err := f.dfs.List(tenantID)
	if err != nil {
		logger.WithError(err).Debug("Could not list snapshots for tenant")
		return nil, err
	}
	return snapshots, nil
}

// TagSnapshot adds tags to an existing snapshot
func (f *Facade) TagSnapshot(snapshotID string, tagName string) error {
	logger := plog.WithFields(logrus.Fields{
		"snapshotid": snapshotID,
		"tagname":    tagName,
	})
	if err := f.dfs.Tag(snapshotID, tagName); err != nil {
		logger.WithError(err).Debug("Could not add tag to snapshot")
		return err
	}
	return nil
}

// RemoveSnapshotTag removes a specific tag from an existing snapshot
func (f *Facade) RemoveSnapshotTag(ctx datastore.Context, serviceID, tagName string) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RemoveSnapshotTag"))
	logger := plog.WithFields(logrus.Fields{
		"serviceid": serviceID,
		"tagname":   tagName,
	})
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not find tenant for service")
		return "", err
	}
	logger = logger.WithField("tenantid", tenantID)
	snapshotID, err := f.dfs.Untag(tenantID, tagName)
	if err != nil {
		logger.WithError(err).Debug("Could not remove tag from tenant")
		return "", err
	}
	return snapshotID, nil
}

// GetSnapshotByServiceIDAndTag finds the existing snapshot for a given service with a specific tag
func (f *Facade) GetSnapshotByServiceIDAndTag(ctx datastore.Context, serviceID, tagName string) (*dfs.SnapshotInfo, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetSnapshotByServiceIDAndTag"))
	logger := plog.WithFields(logrus.Fields{
		"serviceid": serviceID,
		"tagname":   tagName,
	})

	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not find tenant for service")
		return nil, err
	}
	logger = logger.WithField("tenantid", tenantID)
	info, err := f.dfs.TagInfo(tenantID, tagName)
	if err != nil {
		logger.WithError(err).Debug("Could not get info for snapshot tag")
		return nil, err
	}
	return info, nil
}

// ResetLock resets locks for a specific tenant
func (f *Facade) ResetLock(ctx datastore.Context, serviceID string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ResetLock"))
	logger := plog.WithField("serviceid", serviceID)
	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not find tenant for service")
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.ResetLocks"))
	tenantIDs, err := f.GetTenantIDs(ctx)
	if err != nil {
		plog.WithError(err).Debug("Could not get tenants")
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.RepairRegistry"))
	if err := f.DFSLock(ctx).LockWithTimeout("reset registry", userLockTimeout); err != nil {
		plog.WithError(err).Debug("Cannot reset registry")
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	tenantIDs, err := f.GetTenantIDs(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServiceDetailsByTenantID(ctx, tenantID)
		if err != nil {
			return err
		}
		var imagesMap = make(map[string]struct{})
		for _, svc := range svcs {
			if _, ok := imagesMap[svc.ImageID]; !ok {
				if _, err := f.dfs.Download(svc.ImageID, tenantID, true); err != nil {
					plog.WithField("imageid", svc.ImageID).WithError(err).Debug("Could not download image from registry")
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.UpgradeRegistry"))
	logger := plog.WithFields(logrus.Fields{
		"fromregistryhost": fromRegistryHost,
		"force":            force,
	})
	if err := f.DFSLock(ctx).LockWithTimeout("migrate registry", userLockTimeout); err != nil {
		logger.WithError(err).Debug("Cannot migrate registry")
		return err
	}
	defer f.DFSLock(ctx).Unlock()

	success := true // indicates a successful migration
	if fromRegistryHost == "" {
		// check if a local docker migration is needed
		isMigrated, previousVersion, err := f.getPreviousRegistryVersion()
		if err != nil {
			logger.WithError(err).Debug("Could not determine the previous docker registry to migrate")
			return err
		}
		if previousVersion == currentRegistryVersion {
			logger.Info("No previous version of the docker registry exists; nothing to migrate")
			return nil
		}
		logger := logger.WithField("previousversion", previousVersion)
		if !isMigrated || force {
			logger.Info("Starting local docker registry")
			oldRegistryCtr, err := f.startDockerRegistry(previousVersion, oldLocalRegistryPort)
			if err != nil {
				logger.WithField("port", oldLocalRegistryPort).WithError(err).Debug("Could not start old docker registry")
				return err
			}
			logger = logger.WithField("oldregistrycontainer", oldRegistryCtr.Name)
			defer func() {
				if success {
					f.markLocalDockerRegistryUpgraded(previousVersion)
				}
				logger.Infof("Stopping docker registry container")
				if err := oldRegistryCtr.Stop(5 * time.Minute); err != nil {
					logger.WithError(err).Error("Could not stop old docker registry container")
				}
			}()
			fromRegistryHost = fmt.Sprintf("localhost:%s", oldLocalRegistryPort)
		} else {
			logger.Info("Registry already migrated; no action required")
			return nil
		}
	}
	tenantIDs, err := f.GetTenantIDs(ctx)
	if err != nil {
		return err
	}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServiceDetailsByTenantID(ctx, tenantID)
		if err != nil {
			return err
		}
		if err := f.dfs.UpgradeRegistry(svcs, tenantID, fromRegistryHost, force); err != nil {
			success = false
			logger.WithField("tenantid", tenantID).WithError(err).Warning("Could not upgrade registry for tenant")
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
		logger := plog.WithField("version", i)
		plog.Info("Checking docker registry")
		versionInfo := registryVersionInfos[i]
		registryPath := versionInfo.getStoragePath(f.isvcsPath)
		logger = logger.WithField("registrypath", registryPath)
		if _, err := os.Stat(registryPath); os.IsNotExist(err) {
			continue
		} else if err != nil {
			logger.WithError(err).Debug("Could not stat registry path")
			return false, 0, err
		}
		markerFilePath := versionInfo.getUpgradedMarkerPath(f.isvcsPath)
		logger = logger.WithField("markerfilepath", markerFilePath)
		if _, err := os.Stat(markerFilePath); os.IsNotExist(err) {
			return false, i, nil
		} else if err != nil {
			logger.WithError(err).Debug("Could not stat registry marker file")
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
	logger := plog.WithFields(logrus.Fields{
		"version":        version,
		"markerfilepath": markerFilePath,
	})
	if err := ioutil.WriteFile(markerFilePath, []byte{}, 0644); err != nil {
		logger.WithError(err).Debug("Could not write marker file")
		return err
	}
	return nil
}

// Restore restores application data from a backup.
func (f *Facade) Restore(ctx datastore.Context, r io.Reader, backupInfo *dfs.BackupInfo) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.Restore"))
	// Do not DFSLock here, ControlPlaneDao does that
	plog.Info("Started restore from backup")
	if err := f.dfs.Restore(r, backupInfo.BackupVersion); err != nil {
		plog.WithError(err).Debug("Could not restore from backup")
		return err
	}
	if err := f.RestoreServiceTemplates(ctx, backupInfo.Templates); err != nil {
		plog.WithError(err).Debug("Could not restore service templates from backup")
		return err
	}
	plog.Infof("Restored service templates")
	if err := f.RestoreResourcePools(ctx, backupInfo.Pools); err != nil {
		plog.WithError(err).Debug("Could not restore resource pools from backup")
		return err
	}
	plog.Info("Restored resource pools")
	for _, snapshot := range backupInfo.Snapshots {
		logger := plog.WithField("snapshot", snapshot)
		if err := f.Rollback(ctx, snapshot, false); err != nil {
			logger.WithError(err).Debug("Could not rollback snapshot")
			return err
		}
		logger.Info("Rolled back snapshot")
	}
	plog.Info("Completed restore from backup")
	return nil
}

// Rollback rolls back an application to state described in the provided
// snapshot.
func (f *Facade) Rollback(ctx datastore.Context, snapshotID string, force bool) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.Rollback"))
	// Do not DFSLock here, ControlPlaneDao does that
	logger := plog.WithField("snapshotid", snapshotID)
	logger.Infof("Started rollback of snapshot")
	info, err := f.dfs.Info(snapshotID)
	if err != nil {
		logger.WithError(err).Debug("Could not get info for snapshot")
		return err
	}
	logger = logger.WithField("tenantid", info.TenantID)
	if err := f.lockTenant(ctx, info.TenantID); err != nil {
		logger.WithError(err).Debug("Could not lock tenant")
		return err
	}
	defer f.retryUnlockTenant(ctx, info.TenantID, nil, time.Second)
	logger.Info("Got tenant lock")
	svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: info.TenantID})
	if err != nil {
		logger.WithError(err).Debug("Could not get services under tenant")
		return err
	}
	serviceids := make([]string, len(svcs))
	servicesToStop := []*service.Service{}
	for i, _ := range svcs {
		svc := &svcs[i]
		if svc.DesiredState == int(service.SVCRun) {
			servicesToStop = append(servicesToStop, svc)
			if !force {
				logger.WithFields(logrus.Fields{
					"servicename": svc.Name,
					"serviceid":   svc.ID,
				}).Debug("Could not rollback to snapshot; service is running")
				return errors.New("service is running")
			}
		}
		serviceids[i] = svc.ID
	}

	// Stop the services that need stopping in a batch
	if _, err := scheduleServices(f, servicesToStop, ctx, info.TenantID, service.SVCStop, false); err != nil {
		logger.WithError(err).Debug("Could not stop services for rollback")
		return err
	}

	defer scheduleServices(f, servicesToStop, ctx, info.TenantID, service.SVCRun, false)

	if err := f.WaitService(ctx, service.SVCStop, f.dfs.Timeout(), false, serviceids...); err != nil {
		logger.WithError(err).Debug("Could not wait for services to stop during rollback of snapshot")
		return err
	}
	logger.Info("Services are all stopped")
	if err := f.RestoreServices(ctx, info.TenantID, info.Services); err != nil {
		logger.WithError(err).Debug("Could not restore services")
		return err
	}
	logger.Infof("Service data is rolled back")
	if err := f.dfs.Rollback(snapshotID); err != nil {
		logger.WithError(err).Debug("Could not rollback snapshot")
		return err
	}
	logger.Info("Successfully restored application data from snapshot")
	return nil
}

// Snapshot takes a snapshot for a particular application.
func (f *Facade) Snapshot(ctx datastore.Context, serviceID, message string, tags []string, snapshotSpacePercent int) (string, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.Snapshot"))
	// Do not DFSLock here, ControlPlaneDao does that

	logger := plog.WithFields(logrus.Fields{
		"serviceid": serviceID,
	})

	tenantID, err := f.GetTenantID(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not get tenant id of service")
		return "", err
	}
	logger = logger.WithField("tenantid", tenantID)
	if err := f.lockTenant(ctx, tenantID); err != nil {
		logger.WithError(err).Debug("Could not lock tenant")
		return "", err
	}
	defer f.retryUnlockTenant(ctx, tenantID, nil, time.Second)
	logger.Info("Got tenant lock")
	svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
	if err != nil {
		logger.WithError(err).Debug("Could not get services under tenant")
		return "", err
	}
	imagesMap := make(map[string]struct{})
	images := make([]string, 0)
	serviceids := make([]string, len(svcs))
	servicesToPause := []*service.Service{}
	var pausedServiceIds []string
	for i, _ := range svcs {
		svc := &svcs[i]
		if svc.DesiredState == int(service.SVCRun) {
			servicesToPause = append(servicesToPause, svc)
			pausedServiceIds = append(pausedServiceIds, svc.ID)
		}
		serviceids[i] = svc.ID
		if svc.ImageID != "" {
			if _, ok := imagesMap[svc.ImageID]; !ok {
				imagesMap[svc.ImageID] = struct{}{}
				images = append(images, svc.ImageID)
			}
		}
	}
	// Pause the services that need pausing in a batch
	if _, err := scheduleServices(f, servicesToPause, ctx, tenantID, service.SVCPause, false); err != nil {
		logger.WithError(err).Debug("Could not pause services for snapshot")
		return "", err
	}

	defer func() {
		// Refresh service objects in case something has changed (like current state)
		servicesToPause = f.GetServicesForScheduling(ctx, pausedServiceIds)
		scheduleServices(f, servicesToPause, ctx, tenantID, service.SVCRun, false)
	}()

	// Wait for the paused services to reach the paused state (and other services to reach stopped)
	if err := f.WaitService(ctx, service.SVCPause, f.dfs.Timeout(), false, serviceids...); err != nil {
		logger.WithError(err).Debug("Could not wait for services to pause during snapshot")
		return "", err
	}
	logger.Infof("Services are now paused for snapshot")
	data := dfs.SnapshotInfo{
		SnapshotInfo: &volume.SnapshotInfo{
			TenantID: tenantID,
			Message:  message,
			Tags:     tags,
		},
		Services: svcs,
		Images:   images,
	}
	snapshotID, err := f.dfs.Snapshot(data, snapshotSpacePercent)
	if err != nil {
		logger.WithError(err).Debug("Could not snapshot disk and images for tenant")
		return "", err
	}
	logger.WithField("snapshotid", snapshotID).Info("Successfully captured application data and created snapshot")
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
			plog.WithField("containername", containerName).WithError(err).Debug("Could not look up container")
			return nil, err
		}

		// Not found, so make a new one
		containerDefinition := &dockerclient.CreateContainerOptions{
			Name: containerName,
			Config: &dockerclient.Config{
				User:       "root",
				WorkingDir: "/tmp/registry",
				Image:      info.imageId,
				Env:        []string{"SETTINGS_FLAVOR=local"},
			},
			HostConfig: &dockerclient.HostConfig{
				Binds:        []string{bindMount},
				PortBindings: portBindings,
			},
		}
		cclogger := plog.WithFields(logrus.Fields{
			"containername": containerDefinition.Name,
			"image":         containerDefinition.Config.Image,
		})
		container, err = docker.NewContainer(containerDefinition, false, 0, nil, nil)
		if err != nil {
			cclogger.WithError(err).Debug("Error trying to create container")
			return nil, err
		}
		cclogger.Info("Created container")
	}

	os.MkdirAll(storagePath, 0755)

	logger := plog.WithFields(logrus.Fields{
		"containername": container.Name,
		"version":       info.version,
	})

	// Make sure container is running

	if err = container.Start(); err != nil {
		logger.WithError(err).Debug("Could not start container")
		return nil, err
	}
	logger.Infof("Started container for Docker registry")
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
			logger.Debug("Waiting for Docker registry to accept connections...")
		}

		select {
		case <-timeout:
			logger.Warning("Timed out waiting for Docker registry to accept connections")
			if err := container.Stop(5 * time.Minute); err != nil {
				logger.WithError(err).Error("After timeout, could not stop Docker registry container")
			}
			return nil, errors.New(fmt.Sprintf("Timed out waiting for Docker registry v%d to accept connections", info.version))
		case <-time.After(time.Second):
		}
	}

	return container, nil
}

// DockerOverride will replace a docker image in the registry with a new image
func (f *Facade) DockerOverride(ctx datastore.Context, newImageName, oldImageName string) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.DockerOverride"))
	return f.dfs.Override(newImageName, oldImageName)
}

// PredictStorageAvailability returns the predicted available storage after
// a given period for the thin pool data device, the thin pool metadata device,
// and each tenant filesystem.
func (f *Facade) PredictStorageAvailability(ctx datastore.Context, lookahead time.Duration) (map[string]float64, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.PredictStorageAvailability"))
	options := config.GetOptions()

	// First, get a list of all tenant IDs
	tenantIDs, err := f.ListTenants(ctx)
	if err != nil {
		return nil, err
	}

	// Next, query metrics for our window
	window := time.Duration(options.StorageMetricMonitorWindow) * time.Second
	perfdata, err := f.metricsClient.GetAvailableStorage(window, tenantIDs...)
	if err != nil {
		return nil, err
	}
	result := make(map[string]float64)
	predict := func(series metrics.MetricSeries) (float64, error) {
		return statistics.LeastSquaresPredictor.Predict(lookahead, series.X(), series.Y())
	}
	if avail, err := predict(perfdata.PoolDataAvailable); err == nil {
		result[metrics.PoolDataAvailableName] = avail
	}
	if avail, err := predict(perfdata.PoolMetadataAvailable); err == nil {
		result[metrics.PoolMetadataAvailableName] = avail
	}
	for tenant, series := range perfdata.Tenants {
		if avail, err := predict(series); err == nil {
			result[tenant] = avail
		}
	}
	return result, nil
}

// Interface to allow filtering DFS clients
type DfsClientValidator interface {
	ValidateClient(string) bool
}

type clientValidator struct {
	context datastore.Context
	facade  *Facade
}

func NewDfsClientValidator(fac *Facade, ctx datastore.Context) DfsClientValidator {
	return &clientValidator{ctx, fac}
}

func (val *clientValidator) ValidateClient(hostIP string) bool {
	logger := plog.WithField("hostip", hostIP)
	host, err := val.facade.GetHostByIP(val.context, hostIP)
	if err != nil || host == nil {
		logger.Warningf("Unable to load host with given ip")
		return false
	}
	logger = logger.WithField("poolid", host.PoolID)
	pool, err := val.facade.GetResourcePool(val.context, host.PoolID)
	if err != nil || pool == nil {
		logger.Warningf("Unable to load pool")
		return false
	}
	return pool.HasDfsAccess()
}
