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

package btrfs_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/control-center/serviced/volume/drivertest"
	// Register the btrfs driver
	_ "github.com/control-center/serviced/volume/btrfs"
)

var btrfsVolumes map[string]string = make(map[string]string)

// createBtrfsTmpVolume creates a btrfs volume of <size> bytes in a ramdisk,
// based on a loop device. Returns the path to the mounted filesystem.
func createBtrfsTmpVolume(t *testing.T, size int64) string {
	// Make a ramdisk
	ramdiskDir, err := ioutil.TempDir("", "btrfs-ramdisk-")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(ramdiskDir, 0700); err != nil {
		t.Fatal(err)
	}
	if err := syscall.Mount("tmpfs", ramdiskDir, "tmpfs", syscall.MS_MGC_VAL, ""); err != nil {
		t.Fatal(err)
	}
	loopFile := filepath.Join(ramdiskDir, "loop")
	mountPath := filepath.Join(ramdiskDir, "mnt")
	if err := os.MkdirAll(mountPath, 0700); err != nil {
		t.Fatal(err)
	}
	// Create a sparse file of <size> bytes to back the loop device
	file, err := os.OpenFile(loopFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		t.Fatal(err)
	}
	defer file.Close()
	if err = file.Truncate(size); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		t.Fatal(err)
	}
	// Create a btrfs filesystem
	if err := exec.Command("mkfs.btrfs", loopFile).Run(); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		t.Fatal(err)
	}
	// Mount the loop device. System calls to get the next available loopback
	// device are nontrivial, so just shell out, like an animal
	if err := exec.Command("mount", "-o", "loop", loopFile, mountPath).Run(); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		t.Fatal(err)
	}
	btrfsVolumes[mountPath] = ramdiskDir
	return mountPath
}

func cleanupBtrfsTmpVolume(t *testing.T, fsPath string) {
	var (
		ramdisk string
		ok      bool
	)
	if ramdisk, ok = btrfsVolumes[fsPath]; !ok {
		t.Fatal("Tried to clean up a btrfs volume we don't know about")
	}
	// First unmount the loop device
	if err := syscall.Unmount(fsPath, syscall.MNT_DETACH); err != nil {
		t.Error(err)
	}
	// Unmount the ramdisk
	if err := syscall.Unmount(ramdisk, syscall.MNT_DETACH); err != nil {
		t.Fatal(err)
	}
	// Clean up the mount point
	os.RemoveAll(ramdisk)
}

func TestBtrfsCreateEmpty(t *testing.T) {
	root := createBtrfsTmpVolume(t, 32*1024*1024)
	defer cleanupBtrfsTmpVolume(t, root)
	drivertest.DriverTestCreateEmpty(t, "btrfs", root)
}

func TestBtrfsCreateBase(t *testing.T) {
	root := createBtrfsTmpVolume(t, 32*1024*1024)
	defer cleanupBtrfsTmpVolume(t, root)
	drivertest.DriverTestCreateBase(t, "btrfs", root)
}

func TestBtrfsSnapshots(t *testing.T) {
	root := createBtrfsTmpVolume(t, 32*1024*1024)
	defer cleanupBtrfsTmpVolume(t, root)
	drivertest.DriverTestSnapshots(t, "btrfs", root)
}
