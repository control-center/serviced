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
	"strings"
	"sync"
	"syscall"

	. "gopkg.in/check.v1"
)

var (
	ramdisks   map[string]string = make(map[string]string)
	loopdevs   map[string]string = make(map[string]string)
	volumeLock sync.Mutex
)

func CreateRamdisk(size int64) (string, error) {
	// Make a ramdisk
	ramdiskDir, err := ioutil.TempDir("", "serviced-test-")
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(ramdiskDir, 0700); err != nil {
		return "", err
	}
	if err := syscall.Mount("tmpfs", ramdiskDir, "tmpfs", syscall.MS_MGC_VAL, ""); err != nil {
		return "", err
	}
	return ramdiskDir, nil
}

func DestroyRamdisk(path string) {
	// Unmount the ramdisk
	syscall.Unmount(path, syscall.MNT_DETACH)
	// Clean up the mount point
	os.RemoveAll(path)
}

// AllocateLoopFile creates a file of a given size to back a loop device
func AllocateLoopFile(path, prefix string, size int64) (string, error) {
	loopFile := filepath.Join(path, fmt.Sprintf("%s-loop", prefix))
	file, err := os.OpenFile(loopFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		return "", err
	}
	defer file.Close()
	if err = file.Truncate(size); err != nil {
		return "", err
	}
	return loopFile, nil
}

func CreateLoopDevice(path string) (string, error) {
	nextLoop, err := exec.Command("losetup", "--find").CombinedOutput()
	if err != nil {
		return "", err
	}
	trimmed := strings.TrimSpace(string(nextLoop))
	if err := exec.Command("losetup", trimmed, path).Run(); err != nil {
		return "", err
	}
	return trimmed, nil
}

func DestroyLoopDevice(device string) error {
	syscall.Unmount(device, syscall.MNT_DETACH)
	return exec.Command("losetup", "-d", device).Run()
}

type TemporaryDevice struct {
	ramdisk    string
	loopfile   string
	loopdevice string
}

func CreateTemporaryDevice(size int64) (t *TemporaryDevice, err error) {
	t = &TemporaryDevice{}
	t.ramdisk, err = CreateRamdisk(size)
	if err != nil {
		return
	}
	t.loopfile, err = AllocateLoopFile(t.ramdisk, "test", size)
	if err != nil {
		defer DestroyRamdisk(t.ramdisk)
		return
	}
	t.loopdevice, err = CreateLoopDevice(t.loopfile)
	if err != nil {
		defer DestroyRamdisk(t.ramdisk)
		return
	}
	return
}

func (t *TemporaryDevice) Destroy() {
	DestroyLoopDevice(t.loopdevice)
	DestroyRamdisk(t.ramdisk)
}

func (t *TemporaryDevice) RamDisk() string {
	return t.ramdisk
}

func (t *TemporaryDevice) LoopFile() string {
	return t.loopfile
}

func (t *TemporaryDevice) LoopDevice() string {
	return t.loopdevice
}

func CreateTmpVolume(c *C, size int64, fs string) string {
	// Make a ramdisk
	ramdiskDir, err := CreateRamdisk(size)
	c.Assert(err, IsNil)
	mountPath := filepath.Join(ramdiskDir, "mnt")
	err = os.MkdirAll(mountPath, 0700)
	c.Assert(err, IsNil)

	// Create a sparse file of <size> bytes to back the loop device
	loopFile, err := AllocateLoopFile(ramdiskDir, "serviced", size)
	if err != nil {
		defer DestroyRamdisk(ramdiskDir)
		c.Fatal(err)
	}

	// Create a loop device against the file
	loopDevice, err := CreateLoopDevice(loopFile)
	if err != nil {
		defer DestroyRamdisk(ramdiskDir)
		c.Fatal(err)
	}

	// Create a filesystem
	if err := exec.Command(fmt.Sprintf("mkfs.%s", fs), loopDevice).Run(); err != nil {
		defer DestroyRamdisk(ramdiskDir)
		defer DestroyLoopDevice(loopDevice)
		c.Fatal(err)
	}

	// Mount the new filesystem
	if err := exec.Command("mount", "-t", fs, loopDevice, mountPath).Run(); err != nil {
		defer DestroyRamdisk(ramdiskDir)
		defer DestroyLoopDevice(loopDevice)
		c.Fatal(err)
	}

	volumeLock.Lock()
	defer volumeLock.Unlock()
	ramdisks[mountPath] = ramdiskDir
	loopdevs[mountPath] = loopDevice
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

	// Next destroy the loop device
	loopdev := loopdevs[fsPath]
	DestroyLoopDevice(loopdev)

	// Clean up the ramdisk
	DestroyRamdisk(ramdisk)

	// Remove the reference to the volume from our internal map
	delete(ramdisks, fsPath)
	delete(loopdevs, fsPath)
}
