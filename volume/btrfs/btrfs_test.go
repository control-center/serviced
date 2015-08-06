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

// +build root,integration

package btrfs_test

import (
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"testing"

	. "gopkg.in/check.v1"

	"github.com/control-center/serviced/volume/drivertest"
	// Register the btrfs driver
	_ "github.com/control-center/serviced/volume/btrfs"
)

var (
	btrfsVolumes map[string]string = make(map[string]string)
	_                              = Suite(&BtrfsSuite{})
	btrfsArgs    []string          = []string{}
)

// Wire in gocheck
func Test(t *testing.T) { TestingT(t) }

type BtrfsSuite struct {
	root string
}

// createBtrfsTmpVolume creates a btrfs volume of <size> bytes in a ramdisk,
// based on a loop device. Returns the path to the mounted filesystem.
func createBtrfsTmpVolume(c *C, size int64) string {
	// Make a ramdisk
	ramdiskDir, err := ioutil.TempDir("", "btrfs-ramdisk-")
	c.Assert(err, IsNil)
	err = os.MkdirAll(ramdiskDir, 0700)
	c.Assert(err, IsNil)
	err = syscall.Mount("tmpfs", ramdiskDir, "tmpfs", syscall.MS_MGC_VAL, "")
	loopFile := filepath.Join(ramdiskDir, "loop")
	mountPath := filepath.Join(ramdiskDir, "mnt")
	err = os.MkdirAll(mountPath, 0700)
	c.Assert(err, IsNil)

	// Create a sparse file of <size> bytes to back the loop device
	file, err := os.OpenFile(loopFile, os.O_RDWR|os.O_CREATE, 0600)
	if err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		c.Fatal(err)
	}
	defer file.Close()
	if err = file.Truncate(size); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		c.Fatal(err)
	}
	// Create a btrfs filesystem
	if err := exec.Command("mkfs.btrfs", loopFile).Run(); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		c.Fatal(err)
	}
	// Mount the loop device. System calls to get the next available loopback
	// device are nontrivial, so just shell out, like an animal
	if err := exec.Command("mount", "-t", "btrfs", "-o", "loop", loopFile, mountPath).Run(); err != nil {
		defer syscall.Unmount(ramdiskDir, syscall.MNT_DETACH)
		c.Fatal(err)
	}
	btrfsVolumes[mountPath] = ramdiskDir
	return mountPath
}

func cleanupBtrfsTmpVolume(c *C, fsPath string) {
	var (
		ramdisk string
		ok      bool
	)
	ramdisk, ok = btrfsVolumes[fsPath]
	c.Assert(ok, Equals, true)

	// First unmount the loop device
	err := syscall.Unmount(fsPath, syscall.MNT_DETACH)
	c.Check(err, IsNil)

	// Unmount the ramdisk
	err = syscall.Unmount(ramdisk, syscall.MNT_DETACH)
	c.Check(err, IsNil)

	// Clean up the mount point
	os.RemoveAll(ramdisk)
}

func (s *BtrfsSuite) SetUpSuite(c *C) {
	root := createBtrfsTmpVolume(c, 32*1024*1024)
	s.root = root
}

func (s *BtrfsSuite) TearDownSuite(c *C) {
	cleanupBtrfsTmpVolume(c, s.root)
}

func (s *BtrfsSuite) TestBtrfsCreateEmpty(c *C) {
	drivertest.DriverTestCreateEmpty(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsCreateBase(c *C) {
	drivertest.DriverTestCreateBase(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsSnapshots(c *C) {
	drivertest.DriverTestSnapshots(c, "btrfs", s.root, btrfsArgs)
}

func (s *BtrfsSuite) TestBtrfsExportImport(c *C) {
	other_root := createBtrfsTmpVolume(c, 32*1024*1024)
	defer cleanupBtrfsTmpVolume(c, other_root)
	drivertest.DriverTestExportImport(c, "btrfs", s.root, other_root, btrfsArgs)
}
