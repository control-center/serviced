// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package btrfs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/volume"

	"fmt"
	"io"
	"os/exec"
	"os/user"
	"path"
	"strings"
)

const (
	DriverName = "btrfs"
)

type BtrfsDriver struct {
	volumes map[string]VolumeInfo
	sudoer  bool
}

type VolumeInfo struct {
	root string
}

var useSudo bool // use sudo to execute btrfs commands
var BtrfsCmd func(args ...string) MockableCmd

type MockableCmd interface {
	Run() error
	Start() error
	Output() ([]byte, error)
	CombinedOutput() ([]byte, error)
	StderrPipe() (io.ReadCloser, error)
	StdinPipe() (io.WriteCloser, error)
	StdoutPipe() (io.ReadCloser, error)
	Wait() error
}

func init() {
	// verify that btrfs is in the path
	if user, err := user.Current(); err == nil {
		if user.Uid != "0" {
			if err := exec.Command("sudo", "-n", "btrfs", "help").Run(); err == nil {
				useSudo = true
			}
		}
	}
	BtrfsCmd = execBtrfsCmd
}

func execBtrfsCmd(args ...string) (cmd MockableCmd) {
	var myargs []string
	if useSudo {
		myargs = append([]string{"sudo", "-n", "btrfs"}, args...)
	} else {
		myargs = append([]string{"btrfs"}, args...)
	}
	glog.Infof("About to execute: %v", myargs)
	return exec.Command(myargs[0], myargs[1:]...)
}

func New() (*BtrfsDriver, error) {
	if user, err := user.Current(); err != nil {
		return nil, err
	}

	if user.Uid != "0" {
		err := exec.Command("sudo", "-n", "btrfs", "help").Run()
		useSudo = err == nil
	}

	result := &BtrfsDriver{}
	result.volumes = make(map[string]volume.Volume)
	result.sudoer = useSudo

	return result, nil
}

func (d *BtrfsDriver) MkVolume(volumeName, rootDir string) (*Volume, error) {
	if dirp, err := isDir(baseDir); err != nil || dirp == false {
		return nil, err
	}

	volumeDir := path.Join(rootDir, name)
	if cmd := BtrfsCmd("subvolume", "list", "-apuc", volumeDir); cmd.Run() != nil {
		if err := BtrfsCmd("subvolume", "create", volumeDir).Run(); err != nil {
			glog.Errorf("Could not create volume at: %s", volumeDir)
			return nil, fmt.Errorf("could not create subvolume: %s (%v)", name, err)
		}
	}

	v := &Volume{driver: d, name: volumeName}
	d.volumes[volumeName] = VolumeInfo{root: rootDir}

	return v, nil
}

func (d *BtrfsDriver) New(baseDir, name string) (Volume, error) {
	return NewBtrfsVolume(baseDir, name)
}

func (d *BtrfsDriver) Name() (name string) {
	return d.name
}

func (d *BtrfsDriver) Dir() string {
	return path.Join(d.root, d.name)
}

// Snapshot performs a readonly snapshot on the subvolume
func (d *BtrfsDriver) Snapshot(label string) (err error) {
	return BtrfsCmd("subvolume", "snapshot", "-r", v.Dir(), path.Join(d.root, label)).Run()
}

// Snapshots() returns the current snapshots on the volume
func (d *BtrfsDriver) Snapshots() (labels []string, err error) {
	labels = make([]string, 0)
	glog.Info("about to execute subvolume list command")
	if output, err := BtrfsCmd("subvolume", "list", "-apucr", d.root).CombinedOutput(); err != nil {
		glog.Errorf("got an error with subvolume list: %s", string(output))
		return labels, err
	} else {
		glog.Info("btrfs subvolume list:, baseDir: %s", d.root)
		prefixedName := v.name + "_"
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
	return labels, err
}

func (d *BtrfsDriver) RemoveSnapshot(label string) error {
	if exists, err := d.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return errors.New("snapshot does not exist")
		}
	}
	return BtrfsCmd("subvolume", "delete", path.Join(d.root, label)).Run()
}

// Rollback() rolls back the volume to the given snapshot
func (d *BtrfsDriver) Rollback(label string) (err error) {
	if exists, err := d.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return errors.New("snapshot does not exist")
		}
	}
	if dir, err := isDir(d.Dir()); err != nil {
		return err
	} else {
		if dir {
			if err := BtrfsCmd("subvolume", "delete", d.Dir()).Run(); err != nil {
				return err
			}
		}
	}
	return BtrfsCmd("subvolume", "snapshot", path.Join(d.root, label), d.Dir()).Run()
}

// snapshotExists() rolls back the volume to the given snapshot
func (d *BtrfsDriver) snapshotExists(label string) (exists bool, err error) {
	if snapshots, err := d.Snapshots(); err != nil {
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

func (d *BtrfsDriver) RootDir() string {
	return d.root
}
