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
	"errors"
	"io"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ErrDeletingVolume = errors.New("could not delete volume")
	ErrRsyncNotImplemented = errors.New("function not implemented for rsync")
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
	driver  volume.Driver
}

func init() {
	volume.Register(volume.DriverRsync, Init)
}

// Rsync driver intialization
func Init(root string, _ []string) (volume.Driver, error) {
	driver := &RsyncDriver{
		root: root,
	}
	return driver, nil
}

// Root implements volume.Driver.Root
func (d *RsyncDriver) Root() string {
	return d.root
}

// DriverType implements volume.Driver.DriverType
func (d *RsyncDriver) DriverType() volume.DriverType {
	return volume.DriverRsync
}

// Create implements volume.Driver.Create
func (d *RsyncDriver) Create(volumeName string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
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
	sh := exec.Command("rm", "-Rf", v.Path())
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not delete volume: %s", string(output))
		return ErrDeletingVolume
	}
	return nil
}

func (d *RsyncDriver) Status() (*volume.Status, error) {
	glog.V(2).Info("rsync.Status()")
	response := &volume.Status{
		Driver: DriverName,
	}
	return response, nil
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
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
	if files, err := ioutil.ReadDir(d.root); err != nil {
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
	if files, err := ioutil.ReadDir(d.root); err != nil {
		glog.Errorf("Error trying to read from root directory: %s", d.root)
		return false
	} else {
		for _, fi := range files {
			if fi.IsDir() && fi.Name() == volumeName {
				return true
			}
		}
	}
	return false
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

// WriteMetadata writes the metadata info for a snapshot
func (v *RsyncVolume) WriteMetadata(label, name string) (io.WriteCloser, error) {
	filePath := filepath.Join(v.Path(), name)
	if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
		glog.Errorf("Could not create path for file %s: %s", name, err)
		return nil, err
	}
	return os.Create(filePath)
}

// ReadMetadata reads the metadata info from a snapshot
func (v *RsyncVolume) ReadMetadata(label, name string) (io.ReadCloser, error) {
	filePath := filepath.Join(v.Path(), name)
	return os.Open(filePath)
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

// Snapshot implements volume.Volume.Snapshot
func (v *RsyncVolume) Snapshot(label string) (err error) {
	v.Lock()
	defer v.Unlock()
	dest := v.snapshotPath(label)
	if exists, err := volume.IsDir(dest); exists || err != nil {
		if exists {
			glog.Errorf("Snapshot exists: %s", v.rawSnapshotLabel(label))
			return volume.ErrSnapshotExists
		}
		return err
	}

	exe, err := exec.LookPath("rsync")
	if err != nil {
		return err
	}
	argv := []string{"-a", v.Path() + "/", dest + "/"}
	glog.Infof("Performing snapshot rsync command: %s %s", exe, argv)

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
	files, err := ioutil.ReadDir(v.Driver().Root())
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
	dest := v.snapshotPath(label)
	if exists, _ := volume.IsDir(dest); !exists {
		return volume.ErrSnapshotDoesNotExist
	}
	sh := exec.Command("rm", "-Rf", dest)
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", string(output))
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
func (v *RsyncVolume) Export(label, parent, outdir string) (err error) {
	v.Lock()
	defer v.Unlock()
	src := v.snapshotPath(label)
	if exists, err := volume.IsDir(src); err != nil {
		return err
	} else if !exists {
		return volume.ErrSnapshotDoesNotExist
	}

	rsync := exec.Command("rsync", "-azh", src, outdir)
	glog.V(4).Infof("About ro execute %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

// Import implements volume.Volume.Import
func (v *RsyncVolume) Import(rawlabel, indir string) (err error) {
	v.Lock()
	defer v.Unlock()
	path := v.snapshotPath(rawlabel)
	if exists, err := volume.IsDir(path); err != nil {
		return err
	} else if exists {
		return volume.ErrSnapshotExists
	}

	rsync := exec.Command("rsync", "-azh", filepath.Join(indir, rawlabel), v.Driver().Root())
	glog.V(4).Infof("About ro execute %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}
