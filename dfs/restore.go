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
	"strings"
	"sync"

	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var (
	ErrRestoreNoInfo        = errors.New("backup is missing metadata")
	ErrInvalidBackupVersion = errors.New("backup has an invalid version")
)

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
// data is contained in a single tar. It does this by partitioning the tar
// stream into multiple other streams: One for Docker images, which used to be
// an independent tar file within the tar stream (but is now included inline),
// and one for each DFS snapshot being restored.
func (dfs *DistributedFilesystem) restoreVersion1(r io.Reader, data *BackupInfo) error {
	var (
		errs           []error
		registryImages []string
		imagesr        *io.PipeReader
		imagesw        *io.PipeWriter
		wg             sync.WaitGroup
		startedDocker  bool
	)

	// Parse the incoming stream as a tar
	backupTar := tar.NewReader(r)

	// Create a pipe to handle the Docker image tar substream
	imagesr, imagesw = io.Pipe()
	imageTar := tar.NewWriter(imagesw)
	// Start restoring images from the substream. The read will block until
	// something writes to the pipe.
	wg.Add(1)
	go func() {
		defer func() {
			glog.Infof("Docker images finished loading from backup")
			wg.Done()
		}()
		if err := dfs.restoreImages(imagesr); err != nil {
			// Store the error for returning outside this func
			errs = append(errs, err)
		}
	}()

	// A function to create a pipe for each snapshot tar substream, to be
	// restored as a volume
	snapshotStreams := make(map[string]*tar.Writer)
	// pipeStreams track each of the PipeWriters, because they have to be
	// closed on exit.  *tar.Writer does not close pipes.
	pipeStreams := make(map[string]*io.PipeWriter)
	getSnapshotStream := func(tenant, label string) *tar.Writer {
		if _, ok := snapshotStreams[tenant]; !ok {
			// First check to see if the snapshot already exists, and if so,
			// return a nil writer
			fullLabel := volume.DefaultSnapshotLabel(tenant, label)
			glog.Infof("Looking for preexisting volume %s", fullLabel)
			if dfs.disk.Exists(fullLabel) {
				glog.Infof("Volume %s exists; skipping", fullLabel)
				snapshotStreams[tenant] = nil
				return nil
			}
			glog.Infof("Loading snapshot data for %s", tenant)
			r, w := io.Pipe()
			// Store this writer for later retrieval. This is all single-threaded
			// at this level, so no worries about concurrent access.
			snapshotStreams[tenant] = tar.NewWriter(w)
			pipeStreams[tenant] = w
			wg.Add(1)
			go func() {
				defer func() {
					wg.Done()
				}()
				// Restore the incoming tar substream as a snapshot volume
				imgs, err := dfs.restoreVolume(tenant, label, r)
				if err != nil {
					// Store the error for returning outside this func
					errs = append(errs, err)
					return
				}
				// Save off the images implicated by this volume, so we can tag
				// and push them to the registry after everything's restored
				registryImages = append(registryImages, imgs...)
				glog.Infof("Finished loading data for application %s", tenant)
			}()
		}
		return snapshotStreams[tenant]
	}

	cleanup := func() {
		// All done. Write terminators to our pipes to make them valid
		// tars, signaling the functions processing those streams that they
		// can finish up
		imageTar.Close()
		imagesw.Close()
		for k, w := range snapshotStreams {
			if w != nil {
				w.Close()
			}
			if pw, ok := pipeStreams[k]; ok {
				pw.Close()
			}
		}
	}

	for {
		// Short-circuit if any of the subpipes has produced an error so far
		if len(errs) > 0 {
			cleanup()
			return errs[0]
		}
		// Otherwise, move on to the next file in the stream
		hdr, err := backupTar.Next()
		if err == io.EOF {
			cleanup()
			break
		} else if err != nil {
			glog.Errorf("Could not read backup: %s", err)
			cleanup()
			return err
		}
		switch {
		case hdr.Name == BackupMetadataFile:
			// Skip it. We've already got it.
		case strings.HasPrefix(hdr.Name, SnapshotsMetadataDir):
			// This file is part of a volume snapshot. Find or create the pipe
			// responsible for restoring that volume, strip off the extra
			// parent path (so it resembles a tarfile containing the volume
			// data at the root), and write the tar entry to the subpipe.
			parts := strings.Split(hdr.Name, "/")
			if len(parts) < 3 {
				// This is a parent folder or something, not relevant
				continue
			}
			tenant, label := parts[1], parts[2]
			// Find or create the pipe that's got a restoreVolume for this
			// volume reading from the other end
			w := getSnapshotStream(tenant, label)
			if w == nil {
				// Snapshot already exists, so don't bother
				continue
			}
			// Strip off the parent path, so the subtar is a tar with the
			// volume data at the root (resembling a version 0 backup)
			hdr.Name = strings.TrimPrefix(hdr.Name, strings.Join(parts[:3], "/")+"/")
			// Write the entry down the pipe
			w.WriteHeader(hdr)
			io.Copy(w, backupTar)
		case strings.HasPrefix(hdr.Name, DockerImagesFile):
			if !startedDocker {
				glog.Infof("Loading Docker images from backup")
				startedDocker = true
			}
			// Remove the directory prefix, so Docker sees the tar it
			// originally gave us
			hdr.Name = strings.TrimPrefix(hdr.Name, DockerImagesFile+"/")
			// Feed it to the restoreImages goroutine on the other end
			imageTar.WriteHeader(hdr)
			io.Copy(imageTar, backupTar)
		default:
			glog.Warningf("Unrecognized file %s", hdr.Name)
		}
	}
	// Wait for the subpipes to finish up
	wg.Wait()
	// Return any errors, if produced
	if len(errs) > 0 {
		return errs[0]
	}
	// Everything's ok, so finalize by pushing images to the registry if necessary
	return dfs.loadRegistry(registryImages)
}

// restoreVolume restores the application data contained in the tarfile to the
// DFS volume denoted by tenant and label
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
