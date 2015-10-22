// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package rsync

import (
	"archive/tar"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

var (
	ErrDeletingVolume    = errors.New("could not delete volume")
	ErrRsyncInvalidLabel = errors.New("invalid label")
	ErrRsyncDfCommand    = errors.New("error executing df command")
)

// RsyncDriver is a driver for the rsync volume
type RsyncDriver struct {
	sync.Mutex
	root string
}

// RsyncVolume is an rsync volume
type RsyncVolume struct {
	sync.Mutex
	timeout time.Duration
	name    string
	path    string
	tenant  string
	driver  *RsyncDriver
}

type RsyncDFStatus struct {
	Filesystem     string
	TotalBytes     uint64
	UsedBytes      uint64
	AvailableBytes uint64
}

func init() {
	volume.Register(volume.DriverTypeRsync, Init)
}

// Rsync driver intialization
func Init(root string, _ []string) (volume.Driver, error) {
	driver := &RsyncDriver{
		root: root,
	}
	if err := os.MkdirAll(driver.MetadataDir(), 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *RsyncDriver) Root() string {
	return d.root
}

// DriverType implements volume.Driver.DriverType
func (d *RsyncDriver) DriverType() volume.DriverType {
	return volume.DriverTypeRsync
}

// Create implements volume.Driver.Create
func (d *RsyncDriver) Create(volumeName string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
	if d.Exists(volumeName) {
		return nil, volume.ErrVolumeExists
	}
	if err := os.MkdirAll(filepath.Join(d.MetadataDir(), volumeName), 0755); err != nil && !os.IsExist(err) {
		return nil, err
	}
	volumePath := filepath.Join(d.root, volumeName)
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, err
	}
	return d.Get(volumeName)
}

// Remove implements volume.Driver.Remove
func (d *RsyncDriver) Remove(volumeName string) error {
	v, err := d.Get(volumeName)
	if err != nil {
		return err
	}
	// Delete all of the snapshots
	snapshots, err := v.Snapshots()
	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		if err := v.RemoveSnapshot(snapshot); err != nil {
			return err
		}
	}
	// Delete the volume
	v.(*RsyncVolume).Lock()
	defer v.(*RsyncVolume).Unlock()
	if err := os.RemoveAll(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return err
	} else if err := os.RemoveAll(v.Path()); err != nil {
		glog.Errorf("Could not delete volume %s: %s", volumeName, err)
		return ErrDeletingVolume
	}
	return nil
}

func (d *RsyncDriver) poolDir() string {
	return filepath.Join(d.root, ".rsync")
}

// MetadataDir returns the path to a volume's metadata directory
func (d *RsyncDriver) MetadataDir() string {
	return filepath.Join(d.poolDir(), "volumes")
}

// runcmd runs the command
func runcmd(args ...string) ([]byte, error) {
	cmd := append([]string{}, args...)
	glog.V(4).Infof("Executing: %v", cmd)
	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		glog.Errorf("unable to run cmd:%s  output:%s  error:%s", cmd, string(output), err)
		return output, ErrRsyncDfCommand
	}
	return output, err
}

func makeUint64(input string) (uint64, error) {
	return strconv.ParseUint(input, 10, 64)
}

// parseDFCommand parses output of df to get volume.Usage information for Status command
func parseDFCommand(volname string, bytes []byte) ([]volume.Usage, error) {
	outString := strings.TrimSpace(string(bytes))
	lines := strings.Split(outString, "\n")
	result := []volume.Usage{}
	first := true
	for _, line := range lines {
		// skip first (header) line
		if first {
			first = false
			continue
		}
		fields := strings.Fields(line)
		if len(fields) != 4 {
			glog.Errorf("Error parsing DF output (%s). Expected %d fields, but got %d.", line, 4, len(fields))
			return nil, ErrRsyncDfCommand
		}
		label := fmt.Sprintf("%s on %s", volname, string(fields[0]))
		result = appendIfConvertSucceeds(result, fields[1], label, "Total Bytes")
		result = appendIfConvertSucceeds(result, fields[2], label, "Used Bytes")
		result = appendIfConvertSucceeds(result, fields[3], label, "Available Bytes")
	}
	return result, nil
}

func appendIfConvertSucceeds(result []volume.Usage, field string, label string, usage string) []volume.Usage {
	totalBytes, err := makeUint64(field)
	if err != nil {
		glog.Warningf("could not convert string %s to Uint64 for %s", field, usage)
	} else {
		result = append(result, volume.Usage{Label: label, Type: usage, Value: totalBytes})
	}
	return result
}

// Status implements volume.Driver.Status
func (d *RsyncDriver) Status() (*volume.Status, error) {
	glog.V(2).Info("rsync.Status()")

	outBytes, err := runcmd("df", "--output=source,size,used,avail", "-B1", d.root)
	glog.V(2).Infof("Result of running df command: (%q,%q)", outBytes, err)

	dfResult, err2 := parseDFCommand(d.root, outBytes)
	if err2 != nil {
		return nil, err2
	}

	response := &volume.Status{
		Driver:     volume.DriverTypeRsync,
		UsageData:  dfResult,
		DriverData: map[string]string{"DataFile": d.root},
	}

	return response, nil
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

// GetTenant implements volume.Driver.GetTenant
func (d *RsyncDriver) GetTenant(volumeName string) (volume.Volume, error) {
	if !d.Exists(volumeName) {
		return nil, volume.ErrVolumeNotExists
	}
	return d.Get(getTenant(volumeName))
}

// Get implements volume.Driver.Get
func (d *RsyncDriver) Get(volumeName string) (volume.Volume, error) {
	volumePath := filepath.Join(d.root, volumeName)
	volume := &RsyncVolume{
		timeout: 30 * time.Second,
		name:    volumeName,
		path:    volumePath,
		driver:  d,
		tenant:  getTenant(volumeName),
	}
	return volume, nil
}

// Release implements volume.Driver.Release
func (d *RsyncDriver) Release(volumeName string) error {
	// rsync volumes are just a directory; nothing to release
	return nil
}

// List implements volume.Driver.List
func (d *RsyncDriver) List() (result []string) {
	if files, err := ioutil.ReadDir(d.MetadataDir()); err != nil {
		glog.Errorf("Error trying to read from root directory: %s", d.root)
		return
	} else {
		for _, fi := range files {
			if fi.IsDir() {
				result = append(result, fi.Name())
			}
		}
	}
	return
}

// Exists implements volume.Driver.Exists
func (d *RsyncDriver) Exists(volumeName string) bool {
	if finfo, err := os.Stat(filepath.Join(d.MetadataDir(), volumeName)); err != nil {
		return false
	} else {
		return finfo.IsDir()
	}
}

// Cleanup implements volume.Driver.Cleanup
func (d *RsyncDriver) Cleanup() error {
	// Rsync driver has no hold on system resources
	return nil
}

// Name implements volume.Volume.Name
func (v *RsyncVolume) Name() string {
	return v.name
}

// Path implements volume.Volume.Path
func (v *RsyncVolume) Path() string {
	return v.path
}

// Driver implements volume.Volume.Driver
func (v *RsyncVolume) Driver() volume.Driver {
	return v.driver
}

// Tenant implements volume.Volume.Tenant
func (v *RsyncVolume) Tenant() string {
	return v.tenant
}

// WriteMetadata writes the metadata info for a snapshot on the base volume.
func (v *RsyncVolume) WriteMetadata(label, name string) (io.WriteCloser, error) {
	label = v.rawSnapshotLabel(label)
	filePath := filepath.Join(v.driver.MetadataDir(), label, name)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil && !os.IsExist(err) {
		glog.Errorf("Could not create path for file %s: %s", name, err)
		return nil, err
	}
	return os.Create(filePath)
}

// ReadMetadata reads the metadata info from a snapshot.
func (v *RsyncVolume) ReadMetadata(label, name string) (io.ReadCloser, error) {
	// check the metadata directory first
	label = v.rawSnapshotLabel(label)
	filename := filepath.Join(v.driver.MetadataDir(), label, name)
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		// now check the snapshot volume and copy the file into the metadata dir
		snapshotfile := filepath.Join(v.snapshotPath(label), name)
		file, err := os.Open(snapshotfile)
		if err != nil {
			return nil, err
		}
		// if you can't create the new file return the snapshot file
		if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil && !os.IsExist(err) {
			return file, nil
		}
		outfile, err := os.Create(filename)
		if err != nil {
			return file, nil
		}
		// copy the file and seek to the beginning
		defer file.Close()
		if _, err := io.Copy(outfile, file); err != nil {
			return nil, err
		} else if _, err := outfile.Seek(0, 0); err != nil {
			return nil, err
		}
		return outfile, nil
	} else if err != nil {
		return nil, err
	}
	return os.Open(filename)
}

func (v *RsyncVolume) getSnapshotPrefix() string {
	return v.Tenant() + "_"
}

// rawSnapshotLabel ensures that <label> has the tenant prefix for this volume
func (v *RsyncVolume) rawSnapshotLabel(label string) string {
	prefix := v.getSnapshotPrefix()
	if !strings.HasPrefix(label, prefix) {
		return prefix + label
	}
	return label
}

// prettySnapshotLabel ensures that <label> does not have the tenant prefix for
func (v *RsyncVolume) prettySnapshotLabel(rawLabel string) string {
	return strings.TrimPrefix(rawLabel, v.getSnapshotPrefix())
}

// snapshotPath gets the path to the btrfs subvolume representing the snapshot <label>
func (v *RsyncVolume) snapshotPath(label string) string {
	root := v.Driver().Root()
	rawLabel := v.rawSnapshotLabel(label)
	return filepath.Join(root, rawLabel)
}

// isSnapshot checks to see if <rawLabel> describes a snapshot (i.e., begins
// with the tenant prefix)
func (v *RsyncVolume) isSnapshot(rawLabel string) bool {
	return strings.HasPrefix(rawLabel, v.getSnapshotPrefix())
}

// writeSnapshotInfo writes metadata about a snapshot
func (v *RsyncVolume) writeSnapshotInfo(label string, info *volume.SnapshotInfo) error {
	writer, err := v.WriteMetadata(label, ".SNAPSHOTINFO")
	if err != nil {
		glog.Errorf("Could not write meta info for snapshot %s: %s", label, err)
		return err
	}
	defer writer.Close()
	encoder := json.NewEncoder(writer)
	if err := encoder.Encode(info); err != nil {
		glog.Errorf("Could not export meta info for snapshot %s: %s", label, err)
		return err
	}
	return nil
}

// SnapshotInfo returns the meta info for a snapshot
func (v *RsyncVolume) SnapshotInfo(label string) (*volume.SnapshotInfo, error) {
	reader, err := v.ReadMetadata(label, ".SNAPSHOTINFO")
	if err != nil {
		glog.Errorf("Could not get info for snapshot %s: %s", label, err)
		return nil, err
	}
	defer reader.Close()
	decoder := json.NewDecoder(reader)
	var info volume.SnapshotInfo
	if err := decoder.Decode(&info); err != nil {
		glog.Errorf("Could not decode snapshot info for %s: %s", label, err)
		return nil, err
	}
	return &info, err
}

// Snapshot implements volume.Volume.Snapshot
func (v *RsyncVolume) Snapshot(label, message string, tags []string) (err error) {
	v.Lock()
	defer v.Unlock()
	label = v.rawSnapshotLabel(label)
	dest := v.snapshotPath(label)
	if exists, err := volume.IsDir(dest); exists || err != nil {
		if exists {
			glog.Errorf("Snapshot exists: %s", v.rawSnapshotLabel(label))
			return volume.ErrSnapshotExists
		}
		return err
	}
	// write snapshot info
	info := volume.SnapshotInfo{
		Name:     v.rawSnapshotLabel(label),
		TenantID: v.Tenant(),
		Label:    v.prettySnapshotLabel(label),
		Tags:     tags,
		Message:  message,
		Created:  time.Now(),
	}
	if err := v.writeSnapshotInfo(label, &info); err != nil {
		return err
	}
	exe, err := exec.LookPath("rsync")
	if err != nil {
		return err
	}
	argv := []string{"-a", v.Path() + "/", dest + "/"}
	glog.Infof("Performing snapshot rsync command: %s %s", exe, argv)
	if err := os.MkdirAll(filepath.Join(v.driver.MetadataDir(), label), 0755); err != nil && !os.IsExist(err) {
		return err
	}
	var output []byte
	for i := 0; i < 3; i++ {
		rsync := exec.Command(exe, argv...)
		done := make(chan interface{})
		go func() {
			defer close(done)
			output, err = rsync.CombinedOutput()
		}()

		select {
		case <-time.After(v.timeout):
			glog.V(2).Infof("Received signal to kill rsync")
			rsync.Process.Kill()
			<-done
		case <-done:
		}
		if err == nil {
			return nil
		}
		if exitStatus, ok := utils.GetExitStatus(err); !ok || exitStatus != 24 {
			glog.Errorf("Could not perform rsync: %s", string(output))
			return err
		}
		glog.Infof("trying snapshot again: %s", label)
	}
	if exitStatus, _ := utils.GetExitStatus(err); exitStatus == 24 {
		glog.Warningf("snapshot completed with errors: Partial transfer due to vanished source files")
		return nil
	}
	glog.Errorf("Could not perform rsync: %s", string(output))
	return err
}

// Snapshots implements volume.Volume.Snapshots
func (v *RsyncVolume) Snapshots() ([]string, error) {
	v.Lock()
	defer v.Unlock()
	files, err := ioutil.ReadDir(v.driver.MetadataDir())
	if err != nil {
		return nil, err
	}
	var labels []string
	for _, file := range files {
		if file.IsDir() && v.isSnapshot(file.Name()) {
			labels = append(labels, file.Name())
		}
	}
	return labels, nil
}

// RemoveSnapshot implements volume.Volume.RemoveSnapshot
func (v *RsyncVolume) RemoveSnapshot(label string) error {
	v.Lock()
	defer v.Unlock()
	label = v.rawSnapshotLabel(label)
	dest := v.snapshotPath(label)
	if exists, _ := volume.IsDir(dest); !exists {
		return volume.ErrSnapshotDoesNotExist
	}
	if err := os.RemoveAll(filepath.Join(v.driver.MetadataDir(), label)); err != nil {
		return err
	} else if err := os.RemoveAll(dest); err != nil {
		glog.Errorf("Could not remove snapshot %s: %s", label, err)
		return volume.ErrRemovingSnapshot
	}
	return nil
}

// Rollback implements volume.Volume.Rollback
func (v *RsyncVolume) Rollback(label string) (err error) {
	v.Lock()
	defer v.Unlock()
	src := v.snapshotPath(label)
	if exists, err := volume.IsDir(src); !exists || err != nil {
		if !exists {
			return volume.ErrSnapshotDoesNotExist
		}
		return err
	}
	rsync := exec.Command("rsync", "-a", "--del", "--force", src+"/", v.Path()+"/")
	glog.V(4).Infof("About to execute: %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

// Export implements volume.Volume.Export
func (v *RsyncVolume) Export(label, parent string, writer io.Writer) error {
	v.Lock()
	defer v.Unlock()
	if label = strings.TrimSpace(label); label == "" {
		glog.Errorf("%s: label cannot be empty", volume.DriverTypeRsync)
		return ErrRsyncInvalidLabel
	}
	label = v.rawSnapshotLabel(label)
	tarfile := tar.NewWriter(writer)
	defer tarfile.Close()
	// Set the driver type
	header := &tar.Header{Name: fmt.Sprintf("%s-driver", label), Size: int64(len([]byte(v.Driver().DriverType())))}
	if err := tarfile.WriteHeader(header); err != nil {
		glog.Errorf("Could not export driver type header: %s", err)
		return err
	}
	if _, err := fmt.Fprint(tarfile, v.Driver().DriverType()); err != nil {
		glog.Errorf("Could not export driver type: %s", err)
		return err
	}
	// write metadata
	mdpath := filepath.Join(v.driver.MetadataDir(), label)
	if err := volume.ExportDirectory(tarfile, mdpath, fmt.Sprintf("%s-metadata", label)); err != nil {
		return err
	}
	// write volume
	volpath := v.snapshotPath(label)
	if err := volume.ExportDirectory(tarfile, volpath, fmt.Sprintf("%s-volume", label)); err != nil {
		return err
	}
	return nil
}

// Import implements volume.Volume.Import
func (v *RsyncVolume) Import(label string, reader io.Reader) error {
	v.Lock()
	defer v.Unlock()
	label = v.rawSnapshotLabel(label)
	if exists, err := volume.IsDir(v.snapshotPath(label)); err != nil {
		return err
	} else if exists {
		return volume.ErrSnapshotExists
	}
	driverfile := fmt.Sprintf("%s-driver", label)
	volumedir := fmt.Sprintf("%s-volume", label)
	metadatadir := fmt.Sprintf("%s-metadata", label)
	var drivertype string
	tarfile := tar.NewReader(reader)
	for {
		header, err := tarfile.Next()
		if err == io.EOF {
			break
		} else if err != nil {
			glog.Errorf("Could not import archive: %s", err)
			return err
		}
		if header.Name == driverfile {
			buf := bytes.NewBufferString("")
			if _, err := buf.ReadFrom(tarfile); err != nil {
				return err
			}
			drivertype = buf.String()
		} else if strings.HasPrefix(header.Name, volumedir) {
			header.Name = strings.Replace(header.Name, volumedir, label, 1)
			if err := volume.ImportArchiveHeader(header, tarfile, v.driver.Root()); err != nil {
				return err
			}
		} else if strings.HasPrefix(header.Name, metadatadir) {
			header.Name = strings.Replace(header.Name, metadatadir, label, 1)
			if err := volume.ImportArchiveHeader(header, tarfile, v.driver.MetadataDir()); err != nil {
				return err
			}
		}
	}
	if drivertype == "" {
		return errors.New("incompatible snapshot")
	}
	return nil
}
