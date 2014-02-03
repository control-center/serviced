// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package btrfs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/volume"

	"fmt"
	"os/exec"
	"os/user"
	"path"
	"strings"
)

const (
	DriverName = "btrfs"
)

type BtrfsDriver struct {
	sudoer bool
}

type BtrfsConn struct {
	sudoer bool
	name   string
	root   string
}

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

func (d *BtrfsDriver) Mount(volumeName, rootDir string) (volume.Conn, error) {
	if dirp, err := volume.IsDir(rootDir); err != nil || dirp == false {
		return nil, err
	}

	vdir := path.Join(rootDir, volumeName)
	if _, err := runcmd(d.sudoer, "subvolume", "list", "-apuc", vdir); err != nil {
		if _, err = runcmd(d.sudoer, "subvolume", "create", vdir); err != nil {
			glog.Errorf("Could not create volume at: %s", vdir)
			return nil, fmt.Errorf("could not create subvolume: %s (%v)", volumeName, err)
		}
	}

	c := &BtrfsConn{sudoer: d.sudoer, name: volumeName, root: rootDir}
	return c, nil
}

// Snapshot performs a readonly snapshot on the subvolume
func (c *BtrfsConn) Snapshot(label string) error {
	_, err := runcmd(c.sudoer, "subvolume", "snapshot", "-r", c.root, path.Join(c.root, label))
	return err
}

// Snapshots() returns the current snapshots on the volume
func (c *BtrfsConn) Snapshots() ([]string, error) {
	labels := make([]string, 0)
	glog.V(4).Info("about to execute subvolume list command")
	if output, err := runcmd(c.sudoer, "subvolume", "list", "-apucr", c.root); err != nil {
		glog.Errorf("got an error with subvolume list: %s", string(output))
		return labels, err
	} else {
		glog.Info("btrfs subvolume list:, root: %s", c.root)
		prefixedName := c.name + "_"
		for _, line := range strings.Split(string(output), "\n") {
			glog.Infof("btrfs subvolume list: %s", line)
			fields := strings.Fields(line)
			for i, field := range fields {
				if field == "path" {
					fstree := fields[i+1]
					parts := strings.Split(fstree, "/")
					label := parts[len(parts)-1]
					if strings.HasPrefix(label, prefixedName) {
						labels = append(labels, label)
						break
					}
				}
			}
		}
	}
	return labels, nil
}

func (c *BtrfsConn) RemoveSnapshot(label string) error {
	if exists, err := c.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}
	_, err := runcmd(c.sudoer, "subvolume", "delete", path.Join(c.root, label))
	return err
}

// Rollback() rolls back the volume to the given snapshot
func (c *BtrfsConn) Rollback(label string) error {
	if exists, err := c.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return fmt.Errorf("snapshot %s does not exist", label)
		}
	}

	vd := path.Join(c.root, c.name)
	dirp, err := volume.IsDir(vd)
	if err != nil {
		return err
	}

	if dirp {
		if _, err := runcmd(c.sudoer, "subvolume", "delete", vd); err != nil {
			return err
		}
	}

	_, err = runcmd(c.sudoer, "subvolume", "snapshot", path.Join(c.root, label), vd)
	return err
}

// snapshotExists() rolls back the volume to the given snapshot
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

func runcmd(sudoer bool, args ...string) ([]byte, error) {
	cmd := append([]string{"btrfs"}, args...)
	if sudoer {
		cmd = append([]string{"sudo", "-n"}, cmd...)
	}
	glog.V(4).Infof("Executing: %v", cmd)
	return exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
}
