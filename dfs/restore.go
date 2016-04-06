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

package dfs

import (
	"archive/tar"
	"encoding/json"
	"errors"
	"io"
	"os/exec"
	"strings"
	"sync"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var (
	ErrRestoreNoInfo        = errors.New("backup is missing metadata")
	ErrInvalidBackupVersion = errors.New("backup has an invalid version")
)

// ExtractBackupInfo extracts the backup metadata from a tarball on disk in as
// cheaply a manner as possible. The serialized BackupInfo is stored at the
// front of the tarball to facilitate this.
func ExtractBackupInfo(filename string) (*BackupInfo, error) {
	var info BackupInfo
	data, err := exec.Command("tar", "-O", "--occurrence", "-xzf", filename, BackupMetadataFile).CombinedOutput()
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

// Restore restores application data from a backup.
func (dfs *DistributedFilesystem) Restore(r io.Reader, info *BackupInfo) error {
	glog.Infof("Detected backup version %d", info.BackupVersion)
	if info.BackupVersion == 0 {
		return dfs.restoreVersion0(r, info)
	}
	if info.BackupVersion == 1 {
		return dfs.restoreVersion1(r, info)
	}
	return ErrInvalidBackupVersion
}

// restoreVersion0 restores application data from a pre-1.1.3 backup, where the
// backups are tgzs within tgzs
func (dfs *DistributedFilesystem) restoreVersion0(r io.Reader, data *BackupInfo) error {
	var registryImages []string
	glog.Infof("Loading backup data")
	tarfile := tar.NewReader(r)
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			glog.Infof("Finished reading backup")
			if err := dfs.loadRegistry(registryImages); err != nil {
				return err
			}
			return nil
		} else if err != nil {
			glog.Errorf("Could not read backup: %s", err)
			return err
		}
		switch {
		case header.Name == BackupMetadataFile:
			// Skip it, we've already got it
		case strings.HasPrefix(header.Name, SnapshotsMetadataDir):
			parts := strings.Split(header.Name, "/")
			if len(parts) != 3 {
				continue
			}
			tenant, label := parts[1], parts[2]
			imgs, err := dfs.restoreVolume(tenant, label, tarfile)
			if err != nil {
				return err
			}
			registryImages = append(registryImages, imgs...)
		case header.Name == DockerImagesFile:
			if err := dfs.docker.LoadImage(tarfile); err != nil {
				glog.Errorf("Could not load images: %s", err)
				return err
			}
			glog.Infof("Loaded images")
		default:
			glog.Warningf("Unrecognized file %s", header.Name)
		}
	}
}

// restoreVersion1 restores application data from a >=1.1.3 backup, where the
// data is contained in a single tar
func (dfs *DistributedFilesystem) restoreVersion1(r io.Reader, data *BackupInfo) error {

	var (
		registryImages []string
		imagesr        *io.PipeReader
		imagesw        *io.PipeWriter
		wg             sync.WaitGroup
	)

	backupTar := tar.NewReader(r)

	snapshotStreams := make(map[string]*tar.Writer)

	// Get the image-loading pipe going
	imagesr, imagesw = io.Pipe()
	imageTar := tar.NewWriter(imagesw)
	go func() {
		wg.Add(1)
		defer wg.Done()
		dfs.restoreImages(imagesr)
	}()

	// Func to register streams for snapshots
	getSnapshotStream := func(tenant, label string) *tar.Writer {
		if _, ok := snapshotStreams[tenant]; !ok {
			glog.Infof("Loading snapshot data for %s", tenant)
			r, w := io.Pipe()
			snapshotStreams[tenant] = tar.NewWriter(w)
			go func() error {
				wg.Add(1)
				defer wg.Done()
				imgs, err := dfs.restoreVolume(tenant, label, r)
				if err != nil {
					return err
				}
				glog.Infof("Loaded %s data", tenant)
				registryImages = append(registryImages, imgs...)
				return nil
			}()
		}
		return snapshotStreams[tenant]
	}

	for {
		hdr, err := backupTar.Next()
		if err == io.EOF {
			// All done. Write terminators to our pipes to make them valid tars
			imageTar.Close()
			break
		} else if err != nil {
			glog.Errorf("Could not read backup: %s", err)
			return err
		}
		switch {
		case hdr.Name == BackupMetadataFile:
			// Skip it
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			// Find or create the snapshot pipe
			parts := strings.Split(hdr.Name, "/")
			if len(parts) < 3 {
				continue
			}
			tenant, label := parts[1], parts[2]
			w := getSnapshotStream(tenant, label)
			hdr.Name = strings.TrimPrefix(hdr.Name, strings.Join(parts[:3], "/"))
			w.WriteHeader(hdr)
			io.Copy(w, backupTar)
		case strings.HasPrefix(hdr.Name, DockerImagesFile):
			// Remove the directory prefix, so Docker sees the tar it
			// originally gave us
			hdr.Name = strings.TrimPrefix(hdr.Name, DockerImagesFile+"/")
			imageTar.WriteHeader(hdr)
			io.Copy(imageTar, backupTar)
		default:
			glog.Warningf("Unrecognized file %s", hdr.Name)
		}
	}

	wg.Wait()
	return nil
}

func (dfs *DistributedFilesystem) restoreVolume(tenant, label string, tarfile io.Reader) ([]string, error) {
	vol, err := dfs.disk.Create(tenant)
	if err == volume.ErrVolumeExists {
		if vol, err = dfs.disk.Get(tenant); err != nil {
			glog.Errorf("Could not get volume for tenant %s: %s", tenant, err)
			return nil, err
		}
	} else if err != nil {
		glog.Errorf("Could not create volume for tenant %s: %s", tenant, err)
		return nil, err
	}
	// Lets expedite this if this restore had already imported the snapshot
	// But delete the snapshot if it doesn't have the right information
	var snapshotErr error
	defer func() {
		if snapshotErr != nil {
			vol.RemoveSnapshot(label)
		}
	}()
	if snapshotErr = vol.Import(label, tarfile); snapshotErr != nil && snapshotErr != volume.ErrSnapshotExists {
		glog.Errorf("Could not import volume for tenant %s: %s", tenant, snapshotErr)
		return nil, snapshotErr
	}
	// Get all the images for this snapshot for the docker registry
	r, snapshotErr := vol.ReadMetadata(label, ImagesMetadataFile)
	if snapshotErr != nil {
		glog.Errorf("Could not receive images metadata from snapshot %s: %s", label, snapshotErr)
		return nil, snapshotErr
	}
	var images []string
	if snapshotErr = importJSON(r, &images); snapshotErr != nil {
		glog.Errorf("Could not interpret images metadata file from snapshot %s: %s", label, snapshotErr)
		return nil, snapshotErr
	}
	glog.Infof("Loaded volume for tenant %s", tenant)
	return images, nil
}

func (dfs *DistributedFilesystem) restoreImages(r io.Reader) error {
	glog.V(2).Infof("Restoring backup images")
	if err := dfs.docker.LoadImage(r); err != nil {
		glog.Errorf("Unable to restore backup images: %s", err)
		return err
	}
	glog.V(2).Infof("Backup images restored")
	return nil
}

// loadRegistry reads snapshot images and pushes them into the registry with
// the correct registry labeling
func (dfs *DistributedFilesystem) loadRegistry(images []string) error {
	for _, image := range images {
		img, err := dfs.docker.FindImage(image)
		if err != nil {
			glog.Errorf("Could not load image %s into the registry: %s", image, err)
			return err
		}

		hash, err := dfs.docker.GetImageHash(img.ID)
		if err != nil {
			glog.Errorf("Could not get hash for image %s: %s", img.ID, err)
			return err
		}

		if err := dfs.index.PushImage(image, img.ID, hash); err != nil {
			glog.Errorf("Could not push image %s into the registry: %s", image, err)
			return err
		}
	}
	return nil
}
