/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*******************************************************************************/

package btrfs

import (
	"github.com/zenoss/glog"

	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"time"
)

type Volume struct {
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
			if err := exec.Command("sudo", "-n", "btrfs", "help").Run(); err != nil {
				glog.Fatalf("could not execute sudo -n btrfs help: %s", err)
			}
			useSudo = true
		}
	}
	BtrfsCmd = execBtrfsCmd
}

func execBtrfsCmd(args ...string) (cmd MockableCmd) {
	if useSudo {
		myargs := []string{"-n", " "}
		myargs = append(myargs, args...)
		return exec.Command("sudo", myargs...)
	}
	return exec.Command("btrfs", args...)
}

// NewVolume() create a BTRFS volume admin object. If a subvolume does not exist
// it is created.
func NewVolume(baseDir, name string) (*Volume, error) {
	if lstat, err := os.Lstat(baseDir); err != nil {
		return nil, err
	} else {
		if !lstat.IsDir() {
			return nil, errors.New("baseDir is not a directory")
		}
	}

	volumeDir := path.Join(baseDir, name)
	if cmd := BtrfsCmd("subvolume", "list", "-apuc", volumeDir); cmd.Run() != nil {
		if err := BtrfsCmd("subvolume", "create", volumeDir).Run(); err != nil {
			return nil, errors.New("could not create subvolume")
		}
	}
	v := &Volume{
		baseDir: baseDir,
		name:    name,
	}
	return v, nil
}

func (v *Volume) Dir() string {
	return path.Join(v.baseDir, v.name)
}
func (v *Volume) snapshotName(baseDir string) string {
	return path.Join(baseDir, fmt.Sprintf("%s_%d", v.name, time.Now().UnixNano()))
}

// Snapshot performs a readonly snapshot on the subvolume
func (v *Volume) Snapshot() (label string, err error) {
	name := v.snapshotName("")
	return name, BtrfsCmd("subvolume", "snapshot", "-r", v.Dir(), path.Join(v.baseDir, name)).Run()
}

// Snapshots() returns the current snapshots on the volume
func (v *Volume) Snapshots() (labels []string, err error) {
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

func (v *Volume) RemoveSnapshot(label string) error {
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
func (v *Volume) Rollback(label string) (err error) {
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

// Rollback() rolls back the volume to the given snapshot
func (v *Volume) snapshotExists(label string) (exists bool, err error) {
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

func (v *Volume) BaseDir() string {
	return v.baseDir
}

// check if the given path is a directory
func isDir(path string) (bool, error) {
	stat, err := os.Stat(path)
	if err == nil {
		return stat.IsDir(), nil
	} else {
		if os.IsNotExist(err) {
			return false, nil
		}
	}
	return false, err
}
