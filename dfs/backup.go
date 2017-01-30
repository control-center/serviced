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
	"io"
	"path"
	"path/filepath"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/volume"
)

const (
	BackupMetadataFile   = ".BACKUPINFO"
	SnapshotsMetadataDir = "SNAPSHOTS/"
	DockerImagesFile     = "IMAGES.dkr"
)

// Backup writes all application data into an export stream
func (dfs *DistributedFilesystem) Backup(data BackupInfo, w io.Writer) error {

	backupLogger := plog.WithFields(log.Fields{
		"backupversion": data.BackupVersion,
		"timestamp":     data.Timestamp,
	})

	progress := NewProgressCounter(300)
	progress.Log = func() { plog.Infof("Written %v bytes to archive for backup", progress.Total) }

	tarOut := tar.NewWriter(io.MultiWriter(w, progress))

	// write the backup metadata
	if err := dfs.writeBackupMetadata(data, tarOut); err != nil {
		plog.WithError(err).Error("Unable to write metadata for backup")
		return err
	}

	var images []string

	baseImageLogger := backupLogger.WithField("total", len(data.BaseImages))
	baseImageLogger.Info("Preparing docker images for backup")

	// download the base images
	for i, image := range data.BaseImages {
		baseImageLogger = baseImageLogger.WithFields(log.Fields{
			"image":          image,
			"numbercomplete": i + 1,
		})

		if _, err := dfs.docker.FindImage(image); docker.IsImageNotFound(err) {
			if err := dfs.docker.PullImage(image); docker.IsImageNotFound(err) {
				baseImageLogger.Warn("Could not pull base image for backup, skipping")
				continue
			} else if err != nil {
				baseImageLogger.WithError(err).Error("Could not pull image for backup")
				return err
			}
		} else if err != nil {
			baseImageLogger.WithError(err).Error("Could not find image for backup")
			return err
		}

		baseImageLogger.Info("Prepared Docker image for backup")

		images = append(images, image)
	}

	numberOfSnapshots := len(data.Snapshots)

	backupLogger.WithField("total", numberOfSnapshots).Info("Preparing snapshots for backup")

	// export the snapshots
	for i, snapshot := range data.Snapshots {
		vol, info, err := dfs.getSnapshotVolumeAndInfo(snapshot)
		if err != nil {
			return err
		}

		// load the images from this snapshot
		tenantLogger := backupLogger.WithField("tenant", info.TenantID)
		tenantLogger.Info("Preparing images for tenant")

		r, err := vol.ReadMetadata(info.Label, ImagesMetadataFile)
		if err != nil {
			tenantLogger.WithError(err).Error("Could not receive images metadata for tenant")
			return err
		}

		var imgs []string
		if err := importJSON(r, &imgs); err != nil {
			tenantLogger.WithError(err).Error("Could not interpret images metadata for tenant")
			return err
		}

		timer := time.NewTimer(0)
		for _, img := range imgs {
			tenantImageLogger := tenantLogger.WithField("image", img)

			timer.Reset(dfs.timeout)
			if err := dfs.reg.PullImage(timer.C, img); err != nil {
				tenantImageLogger.WithError(err).Error("Could not pull image from registry")
				return err
			}

			image, err := dfs.reg.ImagePath(img)
			if err != nil {
				tenantImageLogger.WithError(err).Error("Could not get the image path from registry")
				return err
			}

			plog.WithField("image", image).Info("Prepared Docker image for backup")

			images = append(images, image)
		}

		timer.Stop()

		snapshotLogger := backupLogger.WithField("snapshot", snapshot)

		// dump the snapshot into the backup
		prefix := path.Join(SnapshotsMetadataDir, info.TenantID, info.Label)
		snapReader, errchan := dfs.snapshotSavePipe(vol, info.Label, data.SnapshotExcludes[snapshot])
		if err := rewriteTar(prefix, tarOut, snapReader); err != nil {
			// be a good citizen and clean up any running threads
			<-errchan
			snapshotLogger.WithError(err).Error("Could not write snapshot to backup")
			return err
		} else if err := <-errchan; err != nil {
			snapshotLogger.WithError(err).Error("Could not export snapshot for backup")
			return err
		}

		snapshotLogger.WithFields(log.Fields{
			"numbercomplete": i + 1,
			"total":          numberOfSnapshots,
		}).Info("Exported snapshot to backup")
	}

	// dump the images from all the snapshots into the backup
	imageReader, errchan := dfs.dockerSavePipe(images...)
	imageLogger := backupLogger.WithField("images", images)
	imageLogger.Info("Starting export of images to backup")
	if err := rewriteTar(DockerImagesFile, tarOut, imageReader); err != nil {
		// be a good citizen and clean up any running threads
		<-errchan
		imageLogger.WithError(err).Error("Could not write images to backup")
		return err
	} else if err := <-errchan; err != nil {
		imageLogger.WithError(err).Error("Could not export images for backup")
		return err
	}
	tarOut.Close()

	imageLogger.Info("Exported images to backup")

	return nil
}

// savePipe is a generic io pipe that returns the reader
func savePipe(do func(w io.Writer) error) (*io.PipeReader, <-chan error) {
	r, w := io.Pipe()
	errchan := make(chan error)
	go func() {
		err := do(w)
		w.Close()
		errchan <- err
	}()
	return r, errchan
}

// dockerSavePipe streams the tar archive output by Docker to pipe's stdout
func (dfs *DistributedFilesystem) dockerSavePipe(images ...string) (*io.PipeReader, <-chan error) {
	return savePipe(func(w io.Writer) error {
		return dfs.docker.SaveImages(images, w)
	})
}

// snapshotSavePipe returns a pipe that exports a given volume to the pipe's stdout
func (dfs *DistributedFilesystem) snapshotSavePipe(vol volume.Volume, label string, excludes []string) (*io.PipeReader, <-chan error) {
	return savePipe(func(w io.Writer) error {
		return vol.Export(label, "", w, excludes)
	})
}

// rewriteTar interprets an pipe reader as a tar reader and rewrites the
// headers so they can get written to the outfile.
func rewriteTar(prefix string, tarWriter *tar.Writer, r *io.PipeReader) error {
	defer r.Close()
	tarReader := tar.NewReader(r)

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			return err
		}

		// Rewrite the header to include the prefix
		header.Name = filepath.Join(prefix, header.Name)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}
		if _, err := io.Copy(tarWriter, tarReader); err != nil {
			return err
		}
	}

	return nil
}

// writeBackupMetadata writes out a tar stream containing a file containing the
// JSON-serialized backup metdata passed in
func (dfs *DistributedFilesystem) writeBackupMetadata(data BackupInfo, w *tar.Writer) error {
	var (
		jsonData []byte
		err      error
	)

	backupLogger := plog.WithFields(log.Fields{
		"backupversion": data.BackupVersion,
		"timestamp":     data.Timestamp,
	})

	backupLogger.Debug("Writing backup metadata")
	if jsonData, err = json.Marshal(data); err != nil {
		return err
	}
	header := &tar.Header{Name: BackupMetadataFile, Size: int64(len(jsonData))}
	if err := w.WriteHeader(header); err != nil {
		backupLogger.WithError(err).Debug("Could not create metadata header for backup")
		return err
	}
	if _, err := w.Write(jsonData); err != nil {
		backupLogger.WithError(err).Debug("Could not write backup metadata")
		return err
	}
	return nil
}
