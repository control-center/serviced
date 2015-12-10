// Copyright 2015 The Serviced Authors.
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

// +build root,integration

package volume

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"sync"
	"syscall"

	. "gopkg.in/check.v1"
)

var (
	ramdisks   map[string]string = make(map[string]string)
	volumeLock sync.Mutex
)

func CreateRamdisk(c *C, size int64) string {
	// Make a ramdisk
	ramdiskDir, err := ioutil.TempDir("", "serviced-test-")
	c.Assert(err, IsNil)
	err = os.MkdirAll(ramdiskDir, 0700)
	c.Assert(err, IsNil)
	err = syscall.Mount("tmpfs", ramdiskDir, "tmpfs", syscall.MS_MGC_VAL, "")
	c.Assert(err, IsNil)
	return ramdiskDir
}

func DestroyRamdisk(c *C, path string) {
	// Unmount the ramdisk
	err := syscall.Unmount(path, syscall.MNT_DETACH)
	c.Check(err, IsNil)

	// Clean up the mount point
	err = os.RemoveAll(path)
	c.Check(err, IsNil)
}

func CreateTmpVolume(c *C, size int64, fs string) string {
	// Make a ramdisk
	ramdiskDir := CreateRamdisk(c, size)
	loopFile := filepath.Join(ramdiskDir, "loop")
	mountPath := filepath.Join(ramdiskDir, "mnt")
	err := os.MkdirAll(mountPath, 0700)
	c.Assert(err, IsNil)

	// Create a sparse file of <size> bytes to back the loop device
	file, err := os.OpenFile(loopFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		defer DestroyRamdisk(c, ramdiskDir)
		c.Fatal(err)
	}
	defer file.Close()
	if err = file.Truncate(size); err != nil {
		defer DestroyRamdisk(c, ramdiskDir)
		c.Fatal(err)
	}
	// Create a btrfs filesystem
	if err := exec.Command(fmt.Sprintf("mkfs.%s", fs), loopFile).Run(); err != nil {
		defer DestroyRamdisk(c, ramdiskDir)
		c.Fatal(err)
	}
	// Mount the loop device. System calls to get the next available loopback
	// device are nontrivial, so just shell out, like an animal
	if err := exec.Command("mount", "-t", fs, "-o", "loop", loopFile, mountPath).Run(); err != nil {
		defer DestroyRamdisk(c, ramdiskDir)
		c.Fatal(err)
	}
	volumeLock.Lock()
	defer volumeLock.Unlock()
	ramdisks[mountPath] = ramdiskDir
	return mountPath
}

// createBtrfsTmpVolume creates a btrfs volume of <size> bytes in a ramdisk,
// based on a loop device. Returns the path to the mounted filesystem.
func CreateBtrfsTmpVolume(c *C, size int64) string {
	return CreateTmpVolume(c, size, "btrfs")
}

func CleanupTmpVolume(c *C, fsPath string) {
	var (
		ramdisk string
		ok      bool
	)
	volumeLock.Lock()
	defer volumeLock.Unlock()

	ramdisk, ok = ramdisks[fsPath]
	c.Assert(ok, Equals, true)

	// First unmount the loop device
	err := syscall.Unmount(fsPath, syscall.MNT_DETACH)
	c.Check(err, IsNil)

	// Clean up the ramdisk
	DestroyRamdisk(c, ramdisk)

	// Remove the reference to the volume from our internal map
	delete(ramdisks, fsPath)
}
