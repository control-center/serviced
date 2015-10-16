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
	"bytes"
	"encoding/json"
	"io"
	"path"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dfsnew/utils"
	"github.com/zenoss/glog"
)

const (
	BackupMetadataFile   = ".BACKUPINFO"
	SnapshotsMetadataDir = "SNAPSHOTS/"
	DockerImagesFile     = "IMAGES.dkr"
)

// Backup writes all application data into an export stream
func (dfs *DistributedFilesystem) Backup(data BackupInfo, w io.Writer) error {
	tarfile := tar.NewWriter(w)
	defer tarfile.Close()
	buffer := bytes.NewBufferString("")
	spool, err := utils.NewSpool("")
	if err != nil {
		glog.Errorf("Could not create spool: %s", err)
		return err
	}
	defer spool.Close()
	// write the backup metadata
	glog.Infof("Writing backup metadata")
	if err := json.NewEncoder(buffer).Encode(data); err != nil {
		glog.Errorf("Could not encode backup metadata: %s", err)
		return err
	}
	header := &tar.Header{Name: BackupMetadataFile, Size: int64(buffer.Len())}
	if err := tarfile.WriteHeader(header); err != nil {
		glog.Errorf("Could not create metadata header for backup: %s", err)
		return err
	}
	if _, err := buffer.WriteTo(tarfile); err != nil {
		glog.Errorf("Could not write backup metadata: %s", err)
		return err
	}
	// download the base images
	for _, image := range data.BaseImages {
		if _, err := dfs.docker.FindImage(image); docker.IsImageNotFound(err) {
			if err := dfs.docker.PullImage(image); err != nil {
				glog.Errorf("Could not pull image %s: %s", image, err)
				return err
			}
		} else if err != nil {
			glog.Errorf("Could not find image %s: %s", image, err)
			return err
		}
	}
	images := data.BaseImages
	// export the snapshots
	for _, snapshot := range data.Snapshots {
		vol, info, err := dfs.getSnapshotVolumeAndInfo(snapshot)
		if err != nil {
			return err
		}
		// load the images from this snapshot
		glog.Infof("Preparing images for tenant %s", info.TenantID)
		r, err := vol.ReadMetadata(info.Label, ImagesMetadataFile)
		if err != nil {
			glog.Errorf("Could not receive images metadata for tenant %s: %s", info.TenantID, err)
			return err
		}
		var imgs []string
		if err := importJSON(r, &imgs); err != nil {
			glog.Errorf("Could not interpret images metadata for tenant %s: %s", info.TenantID, err)
			return err
		}
		for _, img := range imgs {
			if err := dfs.reg.PullImage(img); err != nil {
				glog.Errorf("Could not pull image %s from registry: %s", img, err)
				return err
			}
			image, err := dfs.reg.ImagePath(img)
			if err != nil {
				glog.Errorf("Could not get the image path from registry %s: %s", img, err)
				return err
			}
			images = append(images, image)
		}
		// export the snapshot
		if err := vol.Export(info.Label, "", spool); err != nil {
			glog.Errorf("Could not export tenant %s: %s", info.TenantID, err)
			return err
		}
		glog.Infof("Exporting tenant volume %s", info.TenantID)
		header := &tar.Header{Name: path.Join(SnapshotsMetadataDir, info.TenantID, info.Label), Size: spool.Size()}
		if err := tarfile.WriteHeader(header); err != nil {
			glog.Errorf("Could not create header for tenant %s: %s", info.TenantID, err)
			return err
		}
		if _, err := spool.WriteTo(tarfile); err != nil {
			glog.Errorf("Could not write tenant %s to backup: %s", info.TenantID, err)
			return err
		}
		glog.Infof("Finished exporting tenant volume %s", info.TenantID)
	}
	// export the images
	glog.Infof("Saving images to backup")
	if err := dfs.docker.SaveImages(images, spool); err != nil {
		glog.Errorf("Could not save images to backup: %s", err)
		return err
	}
	header = &tar.Header{Name: DockerImagesFile, Size: spool.Size()}
	if err := tarfile.WriteHeader(header); err != nil {
		glog.Errorf("Could not create docker images header for backup: %s", err)
		return err
	}
	if _, err := spool.WriteTo(tarfile); err != nil {
		glog.Errorf("Could not write docker images: %s", err)
		return err
	}
	glog.Infof("Successfully completed backup")
	return nil
}
