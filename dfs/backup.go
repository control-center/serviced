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

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

const (
	BackupMetadataFile   = ".BACKUPINFO"
	SnapshotsMetadataDir = "SNAPSHOTS/"
	DockerImagesFile     = "IMAGES.dkr"
)

// Backup writes all application data into an export stream
func (dfs *DistributedFilesystem) Backup(data BackupInfo, w io.Writer) error {

	tarOut := tar.NewWriter(w)

	// write the backup metadata
	if err := dfs.writeBackupMetadata(data, tarOut); err != nil {
		glog.Errorf("Unable to write backup metadata: %s", err)
		return err
	}

	var images []string
	// download the base images
	for _, image := range data.BaseImages {
		if _, err := dfs.docker.FindImage(image); docker.IsImageNotFound(err) {
			if err := dfs.docker.PullImage(image); docker.IsImageNotFound(err) {
				glog.Warningf("Could not pull base image %s, skipping", image)
				continue
			} else if err != nil {
				glog.Errorf("Could not pull image %s: %s", image, err)
				return err
			}
		} else if err != nil {
			glog.Errorf("Could not find image %s: %s", image, err)
			return err
		}
		glog.Infof("Prepared Docker image %s for backup", image)
		images = append(images, image)
	}

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
		timer := time.NewTimer(0)
		for _, img := range imgs {
			timer.Reset(30 * time.Minute)
			if err := dfs.reg.PullImage(timer.C, img); err != nil {
				glog.Errorf("Could not pull image %s from registry: %s", img, err)
				return err
			}
			image, err := dfs.reg.ImagePath(img)
			if err != nil {
				glog.Errorf("Could not get the image path from registry %s: %s", img, err)
				return err
			}
			glog.Infof("Prepared Docker image %s for backup", image)
			images = append(images, image)
		}
		timer.Stop()
		// dump the snapshot into the backup
		prefix := path.Join(SnapshotsMetadataDir, info.TenantID, info.Label)
		snapReader, errchan := dfs.snapshotSavePipe(vol, info.Label)
		if err := rewriteTar(prefix, tarOut, snapReader); err != nil {
			// be a good citizen and clean up any running threads
			<-errchan
			glog.Errorf("Could not write snapshot %s to backup: %s", snapshot, err)
			return err
		} else if err := <-errchan; err != nil {
			glog.Errorf("Could not export snapshot %s for backup: %s", snapshot, err)
			return err
		}
		glog.Infof("Exported snapshot %s to backup", snapshot)
	}
	// dump the images from all the snapshots into the backup
	imageReader, errchan := dfs.dockerSavePipe(images...)
	if err := rewriteTar(DockerImagesFile, tarOut, imageReader); err != nil {
		// be a good citizen and clean up any running threads
		<-errchan
		glog.Errorf("Could not write images %v to backup: %s", images, err)
		return err
	} else if err := <-errchan; err != nil {
		glog.Errorf("Could not export images %v for backup: %s", images, err)
		return err
	}
	glog.Infof("Exported images to backup")
	tarOut.Close()
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
func (dfs *DistributedFilesystem) snapshotSavePipe(vol volume.Volume, label string) (*io.PipeReader, <-chan error) {
	return savePipe(func(w io.Writer) error {
		return vol.Export(label, "", w)
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
	glog.V(2).Infof("Writing backup metadata")
	if jsonData, err = json.Marshal(data); err != nil {
		return err
	}
	header := &tar.Header{Name: BackupMetadataFile, Size: int64(len(jsonData))}
	if err := w.WriteHeader(header); err != nil {
		glog.V(2).Infof("Could not create metadata header for backup: %s", err)
		return err
	}
	if _, err := w.Write(jsonData); err != nil {
		glog.V(2).Infof("Could not write backup metadata: %s", err)
		return err
	}
	return nil
}
