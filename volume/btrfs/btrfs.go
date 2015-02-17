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
	sync.Mutex
}

// BtrfsConn is a connection to a btrfs volume
type BtrfsConn struct {
	sudoer bool
	name   string
	root   string
	sync.Mutex
}

func init() {
	btrfsdriver, err := New()
	if err != nil {
		glog.Errorf("Can't create btrfs driver", err)
		return
	}

	volume.Register(DriverName, btrfsdriver)
}

// New creates a new BtrfsDriver
func New() (*BtrfsDriver, error) {
	user, err := user.Current()
	if err != nil {
		return nil, err
	}

	result := &BtrfsDriver{}
	if user.Uid != "0" {
		err := exec.Command("sudo", "-n", "btrfs", "help").Run()
		result.sudoer = err == nil
	}

	return result, nil
}

// Mount creates a new subvolume at given root dir
func (d *BtrfsDriver) Mount(volumeName, rootDir string) (volume.Volume, error) {
	d.Lock()
	defer d.Unlock()
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

	c := &BtrfsConn{sudoer: d.sudoer, name: volumeName, root: rootDir}
	return c, nil
}

// List returns a list of btrfs subvolumes at a given root dir
func (d *BtrfsDriver) List(rootDir string) (result []string) {
	if raw, err := runcmd(d.sudoer, "subvolume", "list", "-a", rootDir); err != nil {
		glog.Errorf("Could not list subvolumes at: %s", rootDir)
	} else {
		rows := strings.Split(string(raw), "\n")
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
func (c *BtrfsConn) Name() string {
	return c.name
}

// Path provides the full path to the subvolume
func (c *BtrfsConn) Path() string {
	return path.Join(c.root, c.name)
}

func (c *BtrfsConn) SnapshotPath(label string) string {
	return path.Join(c.root, label)
}

// Snapshot performs a readonly snapshot on the subvolume
func (c *BtrfsConn) Snapshot(label string) error {
	c.Lock()
	defer c.Unlock()
	_, err := runcmd(c.sudoer, "subvolume", "snapshot", "-r", c.Path(), c.SnapshotPath(label))
	return err
}

// Snapshots returns the current snapshots on the volume (sorted by date)
func (c *BtrfsConn) Snapshots() ([]string, error) {
	c.Lock()
	defer c.Unlock()

	glog.V(2).Infof("listing snapshots of volume:%v and c.name:%s ", c.root, c.name)
	output, err := runcmd(c.sudoer, "subvolume", "list", "-s", c.root)
	if err != nil {
		glog.Errorf("Could not list subvolumes of %s: %s", c.root, err)
		return nil, err
	}

	var files []os.FileInfo
	for _, line := range strings.Split(string(output), "\n") {
		glog.V(2).Infof("line: %s", line)
		if parts := strings.Split(line, "path"); len(parts) == 2 {
			label := strings.TrimSpace(parts[1])
			label = strings.TrimPrefix(label, "volumes/")
			glog.V(2).Infof("looking for tenant:%s in label:'%s'", c.name, label)
			if strings.HasPrefix(label, c.name+"_") {
				file, err := os.Stat(filepath.Join(c.root, label))
				if err != nil {
					glog.Errorf("Could not stat snapshot %s: %s", label, err)
					return nil, err
				}
				files = append(files, file)
				glog.V(2).Infof("found snapshot:%s", label)
			}
		}
	}

	return volume.FileInfoSlice(files).Labels(), nil
}

// RemoveSnapshot removes the snapshot with the given label
func (c *BtrfsConn) RemoveSnapshot(label string) error {
	if exists, err := c.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}

	c.Lock()
	defer c.Unlock()
	_, err := runcmd(c.sudoer, "subvolume", "delete", c.SnapshotPath(label))
	return err
}

// Unmount removes the subvolume that houses all of the snapshots
func (c *BtrfsConn) Unmount() error {
	snapshots, err := c.Snapshots()
	if err != nil {
		return err
	}

	for _, snapshot := range snapshots {
		if err := c.RemoveSnapshot(snapshot); err != nil {
			return err
		}
	}

	c.Lock()
	defer c.Unlock()
	_, err = runcmd(c.sudoer, "subvolume", "delete", c.Path())
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
func (c *BtrfsConn) Rollback(label string) error {
	if exists, err := c.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}

	c.Lock()
	defer c.Unlock()
	vd := path.Join(c.root, c.name)
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
			output, deleteError := runcmd(c.sudoer, cmd...)
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

	cmd := []string{"subvolume", "snapshot", c.SnapshotPath(label), vd}
	_, err = runcmd(c.sudoer, cmd...)
	if err != nil {
		glog.Errorf("rollback of snapshot %s failed for cmd:%s", label, cmd)
	} else {
		duration := time.Now().Sub(start)
		glog.Infof("rollback of snapshot %s took %s", label, duration)
	}
	return err
}

// Export saves a snapshot to an outfile
func (c *BtrfsConn) Export(label, parent, outfile string) error {
	if label == "" {
		return fmt.Errorf("%s: label cannot be empty", DriverName)
	} else if exists, err := c.snapshotExists(label); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("%s: snapshot %s not found", DriverName, label)
	}

	if parent == "" {
		_, err := runcmd(c.sudoer, "send", c.SnapshotPath(label), "-f", outfile)
		return err
	} else if exists, err := c.snapshotExists(label); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("%s: snapshot %s not found", DriverName, parent)
	}

	_, err := runcmd(c.sudoer, "send", c.SnapshotPath(label), "-p", parent, "-f", outfile)
	return err
}

// Import loads a snapshot from an infile
func (c *BtrfsConn) Import(label, infile string) error {
	if exists, err := c.snapshotExists(label); err != nil {
		return err
	} else if exists {
		return fmt.Errorf("%s: snapshot %s exists", DriverName, label)
	}

	// create a tmp path to load the volume
	tmpdir := filepath.Join(c.root, "tmp")
	runcmd(c.sudoer, "subvolume", "create", tmpdir)
	defer runcmd(c.sudoer, "subvolume", "delete", tmpdir)

	if _, err := runcmd(c.sudoer, "receive", tmpdir, "-f", infile); err != nil {
		return err
	}
	defer runcmd(c.sudoer, "subvolume", "delete", filepath.Join(tmpdir, label))

	_, err := runcmd(c.sudoer, "subvolume", "snapshot", "-r", filepath.Join(tmpdir, label), c.root)
	return err
}

// snapshotExists queries the snapshot existence for the given label
func (c *BtrfsConn) snapshotExists(label string) (exists bool, err error) {
	if snapshots, err := c.Snapshots(); err != nil {
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
		glog.Errorf("%s", e)
		return output, e
	}
	return output, err
}
