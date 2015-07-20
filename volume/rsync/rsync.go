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
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	// DriverName is the name of this rsync volume driver implementation
	DriverName = "rsync"
)

// RsyncDriver is a driver for the rsync volume
type RsyncDriver struct {
	sync.Mutex
	root string
}

type RsyncVolume struct {
	sync.Mutex
	timeout time.Duration
	name    string
	path    string
	tenant  string
	driver  volume.Driver
}

func init() {
	volume.Register(DriverName, Init)
}

func Init(root string) (volume.Driver, error) {
	driver := &RsyncDriver{
		root: root,
	}
	return driver, nil
}

func (d *RsyncDriver) Root() string {
	return d.root
}

func (d *RsyncDriver) Create(volumeName string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
	volumePath := filepath.Join(d.root, volumeName)
	if err := os.MkdirAll(volumePath, 0755); err != nil {
		return nil, err
	}
	return d.Get(volumeName)
}

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
		glog.Errorf("could not delete subvolume: %s", string(output))
		return fmt.Errorf("could not delete subvolume: %s", v.Path())
	}
	return nil
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

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

func (d *RsyncDriver) Release(volumeName string) error {
	// rsync volumes are just a directory; nothing to release
	return nil
}

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

func (d *RsyncDriver) Cleanup() error {
	// Rsync driver has no hold on system resources
	return nil
}

func (v *RsyncVolume) Name() string {
	return v.name
}

func (v *RsyncVolume) Path() string {
	return v.path
}

func (v *RsyncVolume) Driver() volume.Driver {
	return v.driver
}

func (v *RsyncVolume) Tenant() string {
	return v.tenant
}

func (v *RsyncVolume) SnapshotMetadataPath(label string) string {
	// Snapshot metadata is stored with the snapshot for this driver
	return v.snapshotPath(label)
}

func (v *RsyncVolume) getSnapshotPrefix() string {
	return v.Tenant() + "_"
}

func (v *RsyncVolume) rawSnapshotLabel(label string) string {
	prefix := v.getSnapshotPrefix()
	if !strings.HasPrefix(label, prefix) {
		return prefix + label
	}
	return label
}

func (v *RsyncVolume) prettySnapshotLabel(rawLabel string) string {
	return strings.TrimPrefix(rawLabel, v.getSnapshotPrefix())
}

func (v *RsyncVolume) snapshotPath(label string) string {
	root := v.Driver().Root()
	rawLabel := v.rawSnapshotLabel(label)
	return filepath.Join(root, rawLabel)
}

func (v *RsyncVolume) isSnapshot(rawLabel string) bool {
	return strings.HasPrefix(rawLabel, v.getSnapshotPrefix())
}

// Snapshot performs a writable snapshot on the subvolume
func (v *RsyncVolume) Snapshot(label string) (err error) {
	v.Lock()
	defer v.Unlock()
	dest := v.snapshotPath(label)
	if exists, err := volume.IsDir(dest); exists || err != nil {
		if exists {
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

// Snapshots returns the current snapshots on the volume
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
			labels = append(labels, v.prettySnapshotLabel(file.Name()))
		}
	}
	return labels, nil
}

// RemoveSnapshot removes the snapshot with the given label
func (v *RsyncVolume) RemoveSnapshot(label string) error {
	v.Lock()
	defer v.Unlock()
	dest := v.snapshotPath(label)
	if exists, _ := volume.IsDir(dest); !exists {
		return fmt.Errorf("snapshot %s doesn't exist for tenant %s", label, v.tenant)
	}
	sh := exec.Command("rm", "-Rf", dest)
	glog.V(4).Infof("About to execute: %s", sh)
	output, err := sh.CombinedOutput()
	if err != nil {
		glog.Errorf("could not remove snapshot: %s", string(output))
		return fmt.Errorf("could not remove snapshot: %s", label)
	}
	return nil
}

// Rollback rolls back the volume to the given snapshot
func (v *RsyncVolume) Rollback(label string) (err error) {
	v.Lock()
	defer v.Unlock()
	src := v.snapshotPath(label)
	if exists, err := volume.IsDir(src); !exists || err != nil {
		if !exists {
			return fmt.Errorf("snapshot %s does not exist", label)
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

// Export copys a snapshot
func (v *RsyncVolume) Export(label, parent, outdir string) (err error) {
	v.Lock()
	defer v.Unlock()
	src := v.snapshotPath(label)
	if exists, err := volume.IsDir(src); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("snapshot %s does not exist", label)
	}

	rsync := exec.Command("rsync", "-azh", src, outdir)
	glog.V(4).Infof("About ro execute %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}

// Import imports a snapshot
func (v *RsyncVolume) Import(rawlabel, indir string) (err error) {
	v.Lock()
	defer v.Unlock()
	path := v.snapshotPath(rawlabel)
	if exists, err := volume.IsDir(path); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("snapshot %s exists", rawlabel)
	}

	fmt.Println("indir: ", indir)
	fmt.Println("backup: ", filepath.Join(indir, rawlabel))
	fmt.Println("restoreto: ", v.Driver().Root())

	rsync := exec.Command("rsync", "-azh", filepath.Join(indir, rawlabel), v.Driver().Root())
	glog.V(4).Infof("About ro execute %s", rsync)
	if output, err := rsync.CombinedOutput(); err != nil {
		glog.V(2).Infof("Could not perform rsync: %s", string(output))
		return err
	}
	return nil
}
