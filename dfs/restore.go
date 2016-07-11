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
	"errors"
	"io"
	"path"
	"strings"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var (
	ErrRestoreNoInfo        = errors.New("backup is missing metadata")
	ErrInvalidBackupVersion = errors.New("backup has an invalid version")
)

// Restore restores application data from a backup.
func (dfs *DistributedFilesystem) Restore(r io.Reader, version int) error {
	glog.Infof("Detected backup version %d", version)
	switch version {
	case 0:
		return dfs.restoreV0(r)
	case 1:
		return dfs.restoreV1(r)
	default:
		return ErrInvalidBackupVersion
	}
}

// restoreV0 restores a pre-1.1.3 backup
func (dfs *DistributedFilesystem) restoreV0(r io.Reader) error {
	backuptar := tar.NewReader(r)

	snapshots := make(map[string][]string)
	for {
		hdr, err := backuptar.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("Could not read backup file: %s", err)
			return err
		}

		switch {
		case hdr.Name == BackupMetadataFile:
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			parts := strings.SplitN(hdr.Name, "/", 4)
			if len(parts) != 3 {
				continue
			}
			tenant, label := parts[1], parts[2]
			if err := dfs.restoreSnapshot(tenant, label, backuptar); err != nil {
				glog.Errorf("Could not restore snapshot %s for tenant %s: %s", label, tenant, err)
				return err
			}
			snapshots[tenant] = append(snapshots[tenant], label)
		case hdr.Name == DockerImagesFile:
			if err := dfs.docker.LoadImage(backuptar); err != nil {
				glog.Errorf("Could not load docker images: %s", err)
				return err
			}
		default:
			glog.Warningf("Unrecognized file %s", hdr.Name)
		}
	}

	for tenant, labels := range snapshots {
		for _, label := range labels {
			if err := dfs.loadSnapshotImages(tenant, label); err != nil {
				return err
			}
		}
	}

	return nil
}

// restoreV1 restores a backup created >= 1.1.3
func (dfs *DistributedFilesystem) restoreV1(r io.Reader) error {
	backuptar := tar.NewReader(r)

	// Keep track of all the data pipes
	var dataError error
	type stream struct {
		writer *io.PipeWriter
		errc   <-chan error
	}
	streamMap := make(map[string]*stream)
	defer func() {
		for _, s := range streamMap {
			s.writer.CloseWithError(dataError)
			<-s.errc
		}
	}()

	// Distribute the tar contents into the correct reader pipes
	for {
		hdr, err := backuptar.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("Could not read backup file: %s", err)
			return err
		}

		switch {
		case hdr.Name == BackupMetadataFile:
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			parts := strings.SplitN(hdr.Name, "/", 4)
			if len(parts) <= 3 {
				continue
			}
			tenant, label := parts[1], parts[2]

			id := path.Join(SnapshotsMetadataDir, tenant, label)
			s, ok := streamMap[id]
			if !ok {
				glog.Infof("Loading snapshot %s for tenant %s from backup", label, tenant)
				writer, errc := dfs.snapshotLoadPipe(tenant, label)
				s = &stream{writer: writer, errc: errc}
				streamMap[id] = s
			}
			hdr.Name = parts[3]
			tarWriter := tar.NewWriter(s.writer)
			tarWriter.WriteHeader(hdr)
			if _, err := io.Copy(tarWriter, backuptar); err != nil {
				glog.Errorf("Could not write snapshot %s for tenant %s: %s", label, tenant, err)
				dataError = err
				return err
			}
		case strings.HasPrefix(hdr.Name, DockerImagesFile):
			parts := strings.SplitN(hdr.Name, "/", 2)
			if len(parts) <= 1 {
				continue
			}

			id := parts[0]
			s, ok := streamMap[id]
			if !ok {
				glog.Infof("Loading docker images from backup")
				writer, errc := dfs.imageLoadPipe()
				s = &stream{writer: writer, errc: errc}
				streamMap[DockerImagesFile] = s
			}
			hdr.Name = parts[1]
			tarWriter := tar.NewWriter(s.writer)
			tarWriter.WriteHeader(hdr)
			if _, err := io.Copy(tarWriter, backuptar); err != nil {
				glog.Errorf("Could not write docker data: %s", err)
				dataError = err
				return err
			}
		default:
			glog.Warningf("Unrecognized file %s", hdr.Name)
		}
	}

	// make sure the image load finishes first
	s, ok := streamMap[DockerImagesFile]
	if ok {
		delete(streamMap, DockerImagesFile)
		s.writer.Close()
		if err := <-s.errc; err != nil {
			glog.Errorf("Could not load docker images from backup: %s", err)
			dataError = err
			return err
		}
	} else {
		glog.Warningf("Backup missing docker image data")
	}

	// load the snapshots and update the images in the registry
	for id, s := range streamMap {
		delete(streamMap, id)
		s.writer.Close()
		if err := <-s.errc; err != nil {
			glog.Errorf("Error trying to import %s: %s", id, err)
			dataError = err
			return err
		}
		parts := strings.Split(id, "/")
		if len(parts) != 3 {
			dataError = errors.New("this should never happen")
			return dataError
		}
		if err := dfs.loadSnapshotImages(parts[1], parts[2]); err != nil {
			dataError = err
			return err
		}
	}

	return nil
}

// imageLoadPipe returns a pipe writer and error channel for restoring docker
// images.
func (dfs *DistributedFilesystem) imageLoadPipe() (*io.PipeWriter, <-chan error) {
	return loadPipe(dfs.docker.LoadImage)
}

// snapshotLoadPipe returns a pipe writer and error channel for restoring
// a snapshot from a backup
func (dfs *DistributedFilesystem) snapshotLoadPipe(tenant, label string) (*io.PipeWriter, <-chan error) {
	return loadPipe(func(r io.Reader) error {
		return dfs.restoreSnapshot(tenant, label, r)
	})
}

// restoreSnapshot restores snapshot volume data from a backup for a specific
// tenant.
func (dfs *DistributedFilesystem) restoreSnapshot(tenant, label string, r io.Reader) error {
	vol, err := dfs.disk.Create(tenant)
	if err == volume.ErrVolumeExists {
		if vol, err = dfs.disk.Get(tenant); err != nil {
			glog.Errorf("Could not get volume for tenant %s: %s", tenant, err)
			return err
		}
	} else if err != nil {
		glog.Errorf("Could not create volume for tenant %s: %s", tenant, err)
		return err
	} else {
		defer func() {
			if err != nil {
				dfs.disk.Remove(tenant)
			}
		}()
	}

	err = vol.Import(label, r)
	if err == volume.ErrSnapshotExists {
		err = nil
	} else if err != nil {
		glog.Errorf("Could not import snapshot %s for tenant %s: %s", label, tenant, err)
		return err
	}

	return nil
}

// loadSnapshotImages adds images to the registry based on the information
// provided by the loaded snapshot.
func (dfs *DistributedFilesystem) loadSnapshotImages(tenant, label string) error {
	vol, err := dfs.disk.Get(tenant)
	if err != nil {
		glog.Errorf("Could not get volume for tenant %s: %s", tenant, err)
		return err
	}

	images, err := func() ([]string, error) {
		r, err := vol.ReadMetadata(label, ImagesMetadataFile)
		if err != nil {
			glog.Errorf("Could not read images metadta from snapshot %s for tenant %s: %s", label, tenant, err)
			return nil, err
		}

		images := []string{}
		if err := importJSON(r, &images); err != nil {
			glog.Errorf("Could not interpret images metadata from snapshot %s for tenant %s: %s", label, tenant, err)
			return nil, err
		}
		return images, nil
	}()

	if err != nil {
		vol.RemoveSnapshot(label)
		return err
	}

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
		glog.V(2).Infof("Loaded image %s into the registry", image)
	}

	glog.V(2).Infof("Loaded images from snapshot %s for tenant %s", label, tenant)
	return nil
}

// loadPipe sets up an async pipe for writing snapshot and docker information.
func loadPipe(do func(io.Reader) error) (*io.PipeWriter, <-chan error) {
	r, w := io.Pipe()
	errc := make(chan error)
	go func() {
		err := do(r)
		r.CloseWithError(err)
		errc <- err
	}()
	return w, errc
}
