// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package volume

import (
	"github.com/zenoss/glog"

	"errors"
	"io"
	"os/exec"
	"os/user"
	"path"
	"strings"
)

type BtrfsVolume struct {
	baseDir string
	name    string
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

// Supported() checks if the given path supports BTRFS. If any is encountered
// it is returned and supported will be set to false.
func Supported(path string) (supported bool, err error) {
	if supported, err = isDir(path); err != nil || supported == false {
		return supported, err
	}
	if _, err = exec.LookPath("btrfs"); err == nil {
		supported = true
	}
	return supported, err
}

// NewVolume() create a BTRFS volume admin object. If a subvolume does not exist
// it is created.
func NewVolume(baseDir, name string) (Volume, error) {
	if baseIsDir, err := isDir(baseDir); err != nil || baseIsDir == false {
		return nil, err
	}

	volumeDir := path.Join(baseDir, name)
	if cmd := BtrfsCmd("subvolume", "list", "-apuc", volumeDir); cmd.Run() != nil {
		if err := BtrfsCmd("subvolume", "create", volumeDir).Run(); err != nil {
			glog.Errorf("Could not create volume at: %s", volumeDir)
			return nil, errors.New("could not create subvolume")
		}
	}
	v := &BtrfsVolume{
		baseDir: baseDir,
		name:    name,
	}
	return v, nil
}

func (v *BtrfsVolume) New(baseDir, name string) (Volume, error) {
	return NewVolume(baseDir, name)
}

func (v *BtrfsVolume) Name() (name string) {
	return v.name
}

func (v *BtrfsVolume) Dir() string {
	return path.Join(v.baseDir, v.name)
}

// Snapshot performs a readonly snapshot on the subvolume
func (v *BtrfsVolume) Snapshot(label string) (err error) {
	return BtrfsCmd("subvolume", "snapshot", "-r", v.Dir(), path.Join(v.baseDir, label)).Run()
}

// Snapshots() returns the current snapshots on the volume
func (v *BtrfsVolume) Snapshots() (labels []string, err error) {
	labels = make([]string, 0)
	glog.Info("about to execute subvolume list command")
	if output, err := BtrfsCmd("subvolume", "list", "-apucr", v.baseDir).CombinedOutput(); err != nil {
		glog.Errorf("got an error with subvolume list: %s", string(output))
		return labels, err
	} else {
		glog.Info("btrfs subvolume list:, baseDir: %s", v.baseDir)
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

func (v *BtrfsVolume) RemoveSnapshot(label string) error {
	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return errors.New("snapshot does not exist")
		}
	}
	return BtrfsCmd("subvolume", "delete", path.Join(v.baseDir, label)).Run()
}

// Rollback() rolls back the volume to the given snapshot
func (v *BtrfsVolume) Rollback(label string) (err error) {
	if exists, err := v.snapshotExists(label); err != nil || !exists {
		if err != nil {
			return err
		} else {
			return errors.New("snapshot does not exist")
		}
	}
	if dir, err := isDir(v.Dir()); err != nil {
		return err
	} else {
		if dir {
			if err := BtrfsCmd("subvolume", "delete", v.Dir()).Run(); err != nil {
				return err
			}
		}
	}
	return BtrfsCmd("subvolume", "snapshot", path.Join(v.baseDir, label), v.Dir()).Run()
}

// snapshotExists() rolls back the volume to the given snapshot
func (v *BtrfsVolume) snapshotExists(label string) (exists bool, err error) {
	if snapshots, err := v.Snapshots(); err != nil {
		return false, errors.New("could not get current snapshot list: " + err.Error())
	} else {
		for _, snapLabel := range snapshots {
			if label == snapLabel {
				return true, nil
			}
		}
	}
	return false, nil
}

func (v *BtrfsVolume) BaseDir() string {
	return v.baseDir
}
