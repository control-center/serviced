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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/volume"
)

var (
	ErrRestoreNoInfo        = errors.New("backup is missing metadata")
	ErrInvalidBackupVersion = errors.New("backup has an invalid version")
)

// Restore restores application data from a backup.
func (dfs *DistributedFilesystem) Restore(r io.Reader, version int) error {
	plog.WithField("version", version).Info("Detected backup version")
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

	// keep track of the snapshots that have been imported
	snapshots := make(map[string][]string)

	for {
		hdr, err := backuptar.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			plog.WithError(err).Error("Could not read backup file")
			return err
		}

		switch {
		case hdr.Name == BackupMetadataFile:
			// Skip it, we've already got it
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			// This is a snapshot volume
			parts := strings.Split(hdr.Name, "/")
			if len(parts) != 3 {
				// this is a parent directory and not the tar file of the
				// snapshot
				continue
			}

			// restore the snapshot
			tenant, label := parts[1], parts[2]
			if err := dfs.restoreSnapshot(tenant, label, backuptar); err != nil {
				plog.WithError(err).WithFields(log.Fields{
					"label":    label,
					"snapshot": tenant,
				}).Error("Could not restore snapshot")
				return err
			}

			// add the snapshot to the list of restored snapshots.
			snapshots[tenant] = append(snapshots[tenant], label)
		case hdr.Name == DockerImagesFile:

			// Load the images from the docker tar
			if err := dfs.docker.LoadImage(backuptar); err != nil {
				plog.WithError(err).Error("Could not load docker images")
				return err
			}
		default:
			plog.WithField("file", hdr.Name).Warn("Unrecognized file")
		}
	}

	// for each snapshot restored, add all the images to the registry
	for tenant, labels := range snapshots {
		for _, label := range labels {
			if err := dfs.loadSnapshotImages(tenant, label); err != nil {
				return err
			}
		}
	}

	return nil
}

// restoreV1 restores application data from a >=1.1.3 backup, where the
// data is contained in a single tar.  It does this by partitioning the tar
// stream into multiple other streams: One for Docker images, which used to be
// and independent tar file within the tar stream (but is now included inline),
// and one for each DFS snapshot being restored.
func (dfs *DistributedFilesystem) restoreV1(r io.Reader) error {
	backuptar := tar.NewReader(r)

	// Keep track of all the data pipes
	var dataError error
	type stream struct {
		tarwriter *tar.Writer
		writer    *io.PipeWriter
		errc      <-chan error
	}
	streamMap := make(map[string]*stream)
	defer func() {
		// close all the data pipes and make sure that all subroutines exit.
		if dataError == nil || dataError == io.EOF {
			dataError = errors.New("unexpected error reading backup")
		}
		for _, s := range streamMap {
			// we don't want to close the tarfile here, because that will send
			// an eof signal to the reader, which overrides the error on the
			// pipeWriter.
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
			plog.WithError(err).Error("Could not read backup file")
			dataError = err
			return err
		}

		switch {
		case hdr.Name == BackupMetadataFile:
			// Skip it, we've already got it
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			// This file is part of a volume snapshot.  Find or create the pipe
			// responsible for restoring that volume, strip off the extra
			// parent path (so it resembles a tarfile containing the volume
			// data at the root), and write the tar entry to the subpipe.
			parts := strings.SplitN(hdr.Name, "/", 4)
			if len(parts) <= 3 {
				// this is a parent directory, and there is nothing to pass
				// to the snapshot stream.
				continue
			}
			tenant, label := parts[1], parts[2]

			tenantLogger := plog.WithFields(log.Fields{
				"label":  label,
				"tenant": tenant,
			})

			id := path.Join(SnapshotsMetadataDir, tenant, label)
			// Find or create the pipe that's got a restoreSnapshot for this
			// volume reading from the other end
			s, ok := streamMap[id]
			if !ok {
				tenantLogger.Info("Loading snapshot for tenant from backup")

				writer, errc := dfs.snapshotLoadPipe(tenant, label)
				tarwriter := tar.NewWriter(writer)
				s = &stream{tarwriter: tarwriter, writer: writer, errc: errc}
				streamMap[id] = s
			}

			hdr.Name = parts[3]
			if err := s.tarwriter.WriteHeader(hdr); err == io.ErrClosedPipe {
				// Snapshot already exists, so don't bother
				continue
			} else if err != nil {
				tenantLogger.WithError(err).WithField("header", hdr.Name).
					Error("Could not write header for snapshot on tenant")
				dataError = err
				return err
			}

			if _, err := io.Copy(s.tarwriter, backuptar); err != nil {
				tenantLogger.WithError(err).WithField("header", hdr.Name).
					Error("Could not write snapshot for tenant with header")
				dataError = err
				return err
			}
		case strings.HasPrefix(hdr.Name, DockerImagesFile):
			// Find or create the pipe that's got a LoadImages for this path
			// reading from the other end.
			parts := strings.SplitN(hdr.Name, "/", 2)
			if len(parts) <= 1 {
				continue
			}

			id := parts[0]
			s, ok := streamMap[id]
			if !ok {
				plog.Info("Loading docker images from backup")
				writer, errc := dfs.imageLoadPipe()
				tarwriter := tar.NewWriter(writer)
				s = &stream{tarwriter: tarwriter, writer: writer, errc: errc}
				streamMap[DockerImagesFile] = s
			}
			hdr.Name = parts[1]
			if err := s.tarwriter.WriteHeader(hdr); err != nil {
				plog.WithError(err).WithField("header", hdr.Name).
					Error("Could not write image header")
				dataError = err
				return err
			} else if _, err := io.Copy(s.tarwriter, backuptar); err != nil {
				plog.WithError(err).WithField("header", hdr.Name).
					Error("Could not write image data with header")
				dataError = err
				return err
			}
		default:
			plog.WithField("name", hdr.Name).Warn("Unrecognized file")
		}
	}

	// make sure the image load finishes first
	s, ok := streamMap[DockerImagesFile]
	if ok {
		delete(streamMap, DockerImagesFile)
		s.tarwriter.Close()
		s.writer.Close()
		if err := <-s.errc; err != nil {
			plog.WithError(err).Error("Could not load docker images from backup")
			dataError = err
			return err
		}
	} else {
		plog.Warn("Backup missing docker image data")
	}

	// load the snapshots and update the images in the registry
	for id, s := range streamMap {
		delete(streamMap, id)
		s.tarwriter.Close()
		s.writer.Close()
		if err := <-s.errc; err != nil {
			// this snapshot is no good, but maybe the other snapshots are
			// better.
			plog.WithError(err).WithField("id", id).Error("Error trying to import")
			dataError = err
			continue
		}

		parts := strings.Split(id, "/")
		if len(parts) != 3 {
			dataError = errors.New("this should never happen")
			return dataError
		}

		if err := dfs.loadSnapshotImages(parts[1], parts[2]); err != nil {
			// could not load images for this snapshot, but maybe other
			// snapshots are better.
			dataError = err
			continue
		}
	}

	return dataError
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
	tenantLogger := plog.WithField("tenant", tenant)

	vol, err := dfs.disk.Create(tenant)
	if err == volume.ErrVolumeExists {
		if vol, err = dfs.disk.Get(tenant); err != nil {
			tenantLogger.WithError(err).Error("Could not get volume for tenant")
			return err
		}
	} else if err != nil {
		tenantLogger.WithError(err).Error("Could not create volume for tenant")
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
		err = nil // volume.ErrSnapshotExists is an error we can ignore
	} else if err != nil {
		tenantLogger.WithError(err).WithField("label", label).
			Error("Could not import snapshot for tenant")
		return err
	}

	return nil
}

// loadSnapshotImages adds images to the registry based on the information
// provided by the loaded snapshot.
func (dfs *DistributedFilesystem) loadSnapshotImages(tenant, label string) error {
	vol, err := dfs.disk.Get(tenant)
	if err != nil {
		plog.WithError(err).WithField("tenant", tenant).Error("Could not get volume for tenant")
		return err
	}

	tenantLogger := plog.WithFields(log.Fields{
		"label":  label,
		"tenant": tenant,
	})

	// get the list of images for this snapshot
	images, err := func() ([]string, error) {
		r, err := vol.ReadMetadata(label, ImagesMetadataFile)
		defer r.Close()
		if err != nil {
			tenantLogger.WithError(err).Error("Could not read images metadata from snapshot for tenant")
			return nil, err
		}

		images := []string{}
		if err := importJSON(r, &images); err != nil {
			tenantLogger.WithError(err).Error("Could not interpret images metadata from snapshot for tenant")
			return nil, err
		}
		return images, nil
	}()

	// if the image data is incomplete, this is a bad snapshot, so remove it
	if err != nil {
		vol.RemoveSnapshot(label)
		return err
	}

	// try to load the image into the registry
	for _, image := range images {
		imageLogger := plog.WithField("image", image)

		img, err := dfs.docker.FindImage(image)
		if err != nil {
			imageLogger.WithError(err).Warn("Missing image for import to registry")
			continue
		}

		hash, err := dfs.docker.GetImageHash(img.ID)
		if err != nil {
			imageLogger.WithField("imageid", img.ID).WithError(err).
				Error("Could not get hash for image")
			return err
		}

		if err := dfs.index.PushImage(image, img.ID, hash); err != nil {
			imageLogger.WithError(err).Error("Could not push image into the registry")
			return err
		}

		imageLogger.Info("Loaded image into the registry")
	}

	tenantLogger.Info("Loaded images from snapshot for tenant")

	return nil
}

// loadPipe sets up an async pipe for writing snapshot and docker information.
func loadPipe(do func(io.Reader) error) (*io.PipeWriter, <-chan error) {
	r, w := io.Pipe()
	errc := make(chan error)
	go func() {
		progress := NewProgressCounter(300)
		progress.Log = func() { plog.Infof("Read %v bytes from archive for restore", progress.Total) }

		tr := io.TeeReader(r, progress)
		err := do(tr)
		r.CloseWithError(err)
		errc <- err
	}()
	return w, errc
}
