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

package btrfs

import (
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"fmt"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	// DriverName is the name of this btrfs driver implementation
	DriverName = "btrfs"
)

// BtrfsDriver is a driver for the btrfs volume
type BtrfsDriver struct {
	sudoer bool
	root   string
	sync.Mutex
}

// BtrfsVolume is a connection to a btrfs volume
type BtrfsVolume struct {
	sudoer bool
	name   string
	path   string
	tenant string
	driver volume.Driver
	sync.Mutex
}

func init() {
	volume.Register(DriverName, Init)
}

func Init(root string) (volume.Driver, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}
	driver := &BtrfsDriver{
		sudoer: true,
		root:   root,
	}
	if user.Uid != "0" {
		err := exec.Command("sudo", "-n", "btrfs", "help").Run()
		driver.sudoer = err == nil
	}
	return driver, nil
}

func (d *BtrfsDriver) Root() string {
	return d.root
}

func (d *BtrfsDriver) Exists(volumeName string) bool {
	// TODO: More efficient version of this. This is stupid.
	for _, vol := range d.List() {
		if vol == volumeName {
			return true
		}
	}
	return false
}

func (d *BtrfsDriver) Cleanup() error {
	// TODO: Implement this. Btrfs definitely hangs on to stuff and needs to unmount.
	return nil
}

func (d *BtrfsDriver) Release(volumeName string) error {
	// TODO: Implement this. Btrfs definitely hangs on to stuff and needs to unmount.
	return nil
}

// Mount creates a new subvolume at given root dir
func (d *BtrfsDriver) Create(volumeName string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
	rootDir := d.root
	if _, err := runcmd(d.sudoer, "subvolume", "list", rootDir); err != nil {
		if _, err := runcmd(d.sudoer, "subvolume", "create", rootDir); err != nil {
			glog.Errorf("Could not create subvolume at: %s", rootDir)
			return nil, fmt.Errorf("could not create subvolume: %s (%v)", rootDir, err)
		}
	}
	vdir := path.Join(rootDir, volumeName)
	if _, err := runcmd(d.sudoer, "subvolume", "list", vdir); err != nil {
		if _, err = runcmd(d.sudoer, "subvolume", "create", vdir); err != nil {
			glog.Errorf("Could not create volume at: %s", vdir)
			return nil, fmt.Errorf("could not create subvolume: %s (%v)", volumeName, err)
		}
	}
	return d.Get(volumeName)
}

func (d *BtrfsDriver) Remove(volumeName string) error {
	return nil
}

func getTenant(from string) string {
	parts := strings.Split(from, "_")
	return parts[0]
}

func (d *BtrfsDriver) Get(volumeName string) (volume.Volume, error) {
	volumePath := filepath.Join(d.root, volumeName)
	v := &BtrfsVolume{
		sudoer: d.sudoer,
		name:   volumeName,
		path:   volumePath,
		driver: d,
		tenant: getTenant(volumeName),
	}
	return v, nil
}

// List returns a list of btrfs subvolumes at a given root dir
func (d *BtrfsDriver) List() (result []string) {
	if raw, err := runcmd(d.sudoer, "subvolume", "list", "-a", d.root); err != nil {
		glog.Errorf("Could not list subvolumes at: %s", d.root)
	} else {
		cleanraw := strings.TrimSpace(string(raw))
		rows := strings.Split(cleanraw, "\n")
		for _, row := range rows {
			if parts := strings.Split(row, "path"); len(parts) != 2 {
				glog.Errorf("Bad format parsing subvolume row: %s", row)
			} else {
				result = append(result, strings.TrimSpace(parts[1]))
			}
		}
	}
	return
}

// Name provides the name of the subvolume
func (v *BtrfsVolume) Name() string {
	return v.name
}

// Path provides the full path to the subvolume
func (v *BtrfsVolume) Path() string {
	return v.path
}

// Driver returns the driver managing this volume
func (v *BtrfsVolume) Driver() volume.Driver {
	return v.driver
}

func (v *BtrfsVolume) Tenant() string {
	return v.tenant
}

func (v *BtrfsVolume) getSnapshotPrefix() string {
	return v.Tenant() + "_"
}

func (v *BtrfsVolume) rawSnapshotLabel(label string) string {
	prefix := v.getSnapshotPrefix()
	if !strings.HasPrefix(label, prefix) {
		return prefix + label
	}
	return label
}

func (v *BtrfsVolume) prettySnapshotLabel(rawLabel string) string {
	return strings.TrimPrefix(rawLabel, v.getSnapshotPrefix())
}

func (v *BtrfsVolume) snapshotPath(label string) string {
	root := v.Driver().Root()
	rawLabel := v.rawSnapshotLabel(label)
	return filepath.Join(root, rawLabel)
}

func (v *BtrfsVolume) isSnapshot(rawLabel string) bool {
	return strings.HasPrefix(rawLabel, v.getSnapshotPrefix())
}

// Snapshot performs a readonly snapshot on the subvolume
func (v *BtrfsVolume) Snapshot(label string) error {
	v.Lock()
	defer v.Unlock()
	_, err := runcmd(v.sudoer, "subvolume", "snapshot", "-r", v.Path(), v.snapshotPath(label))
	return err
}

// Snapshots returns the current snapshots on the volume (sorted by date)
func (v *BtrfsVolume) Snapshots() ([]string, error) {
	v.Lock()
	defer v.Unlock()

	glog.V(2).Infof("listing snapshots of volume:%v and v.name:%s ", v.path, v.name)
	output, err := runcmd(v.sudoer, "subvolume", "list", "-s", v.path)
	if err != nil {
		glog.Errorf("Could not list subvolumes of %s: %s", v.path, err)
		return nil, err
	}

	var files []os.FileInfo
	for _, line := range strings.Split(string(output), "\n") {
		glog.V(0).Infof("line: %s", line)
		if parts := strings.Split(line, "path"); len(parts) == 2 {
			rawLabel := strings.TrimSpace(parts[1])
			rawLabel = strings.TrimPrefix(rawLabel, "volumes/")
			if v.isSnapshot(rawLabel) {
				label := v.prettySnapshotLabel(rawLabel)
				file, err := os.Stat(filepath.Join(v.Driver().Root(), rawLabel))
				if err != nil {
					glog.Errorf("Could not stat snapshot %s: %s", label, err)
					return nil, err
				}
				files = append(files, file)
				glog.V(2).Infof("found snapshot:%s", label)
			}
		}
	}
	labels := volume.FileInfoSlice(files).Labels()
	result := make([]string, len(labels))
	for i := 0; i < len(labels); i++ {
		result[i] = v.prettySnapshotLabel(labels[i])
	}
	return result, nil
}

// RemoveSnapshot removes the snapshot with the given label
func (v *BtrfsVolume) RemoveSnapshot(label string) error {
	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}

	v.Lock()
	defer v.Unlock()
	_, err := runcmd(v.sudoer, "subvolume", "delete", v.snapshotPath(label))
	return err
}

// getEnvMinDuration returns the time.Duration env var meeting minimum and default duration
func getEnvMinDuration(envvar string, def, min int32) time.Duration {
	duration := def
	envval := os.Getenv(envvar)
	if len(strings.TrimSpace(envval)) == 0 {
		// ignore unset envvar
	} else if intVal, intErr := strconv.ParseInt(envval, 0, 32); intErr != nil {
		glog.Warningf("ignoring invalid %s of '%s': %s", envvar, envval, intErr)
		duration = min
	} else if int32(intVal) < min {
		glog.Warningf("ignoring invalid %s of '%s' < minimum:%v seconds", envvar, envval, min)
	} else {
		duration = int32(intVal)
	}

	return time.Duration(duration) * time.Second
}

// Rollback rolls back the volume to the given snapshot
func (v *BtrfsVolume) Rollback(label string) error {
	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}

	v.Lock()
	defer v.Unlock()
	vd := filepath.Join(v.Driver().Root(), v.name)
	dirp, err := volume.IsDir(vd)
	if err != nil {
		return err
	}

	glog.Infof("starting rollback of snapshot %s", label)

	start := time.Now()
	if dirp {
		timeout := getEnvMinDuration("SERVICED_BTRFS_ROLLBACK_TIMEOUT", 300, 120)
		glog.Infof("rollback using env var SERVICED_BTRFS_ROLLBACK_TIMEOUT:%s", timeout)

		for {
			cmd := []string{"subvolume", "delete", vd}
			output, deleteError := runcmd(v.sudoer, cmd...)
			if deleteError == nil {
				break
			}

			now := time.Now()
			if now.Sub(start) > timeout {
				glog.Errorf("rollback of snapshot %s failed - btrfs subvolume deletes took %s for cmd:%s", label, timeout, cmd)
				return deleteError
			} else if strings.Contains(string(output), "Device or resource busy") {
				waitTime := time.Duration(5 * time.Second)
				glog.Warningf("retrying rollback subvolume delete in %s - unable to run cmd:%s  output:%s  error:%s", waitTime, cmd, string(output), deleteError)
				time.Sleep(waitTime)
			} else {
				return deleteError
			}
		}
	}

	cmd := []string{"subvolume", "snapshot", v.snapshotPath(label), vd}
	_, err = runcmd(v.sudoer, cmd...)
	if err != nil {
		glog.Errorf("rollback of snapshot %s failed for cmd:%s", label, cmd)
	} else {
		duration := time.Now().Sub(start)
		glog.Infof("rollback of snapshot %s took %s", label, duration)
	}
	return err
}

// Export saves a snapshot to an outfile
func (v *BtrfsVolume) Export(label, parent, outfile string) error {
	if label == "" {
		return fmt.Errorf("%s: label cannot be empty", DriverName)
	} else if exists, err := v.snapshotExists(label); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("%s: snapshot %s not found", DriverName, label)
	}

	if parent == "" {
		_, err := runcmd(v.sudoer, "send", v.snapshotPath(label), "-f", outfile)
		return err
	} else if exists, err := v.snapshotExists(label); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("%s: snapshot %s not found", DriverName, parent)
	}

	_, err := runcmd(v.sudoer, "send", v.snapshotPath(label), "-p", parent, "-f", outfile)
	return err
}

// Import loads a snapshot from an infile
func (v *BtrfsVolume) Import(label, infile string) error {
	if exists, err := v.snapshotExists(label); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("%s: snapshot %s exists", DriverName, label)
	}

	// create a tmp path to load the volume
	tmpdir := filepath.Join(v.path, "tmp")
	runcmd(v.sudoer, "subvolume", "create", tmpdir)
	defer runcmd(v.sudoer, "subvolume", "delete", tmpdir)

	if _, err := runcmd(v.sudoer, "receive", tmpdir, "-f", infile); err != nil {
		return err
	}
	defer runcmd(v.sudoer, "subvolume", "delete", filepath.Join(tmpdir, label))

	_, err := runcmd(v.sudoer, "subvolume", "snapshot", "-r", filepath.Join(tmpdir, label), v.Driver().Root())
	return err
}

// snapshotExists queries the snapshot existence for the given label
func (v *BtrfsVolume) snapshotExists(label string) (exists bool, err error) {
	label = v.prettySnapshotLabel(label)
	if snapshots, err := v.Snapshots(); err != nil {
		return false, fmt.Errorf("could not get current snapshot list: %v", err)
	} else {
		for _, snapLabel := range snapshots {
			if label == snapLabel {
				return true, nil
			}
		}
	}
	return false, nil
}

// IsBtrfsFilesystem determines whether the path is a btrfs filesystem
func IsBtrfsFilesystem(thePath string) error {
	_, err := runcmd(false, "filesystem", "df", thePath)
	return err
}

// runcmd runs the btrfs command
func runcmd(sudoer bool, args ...string) ([]byte, error) {
	cmd := append([]string{"btrfs"}, args...)
	if sudoer {
		cmd = append([]string{"sudo", "-n"}, cmd...)
	}
	glog.V(4).Infof("Executing: %v", cmd)
	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		e := fmt.Errorf("unable to run cmd:%s  output:%s  error:%s", cmd, string(output), err)
		glog.V(0).Infof("%s", e)
		return output, e
	}
	return output, err
}
