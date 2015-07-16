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
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"reflect"
	"syscall"
	"testing"
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
	if err := syscall.Unmount(ramdisk, syscall.MNT_DETACH); err != nil {
		t.Fatal(err)
	}
}

func TestBtrfsVolume(t *testing.T) {

	if user, err := user.Current(); err != nil {
		panic(err)
	} else {
		if user.Uid != "0" {
			t.Skip("Skipping BTRFS tests because we are not running as root")
		}
	}

	if _, err := exec.LookPath("btrfs"); err != nil {
		t.Skip("Skipping BTRFS tests because btrfs-tools were not found in the path")
	}

	// Create a 64MB btrfs test volume
	btrfsTestVolumePath := createBtrfsTmpVolume(t, 256*1024*1024)
	defer cleanupBtrfsTmpVolume(t, btrfsTestVolumePath)

	t.Logf("Using '%s' as btrfs test volume", btrfsTestVolumePath)

	if err := os.MkdirAll(btrfsTestVolumePath, 0775); err != nil {
		t.Fatalf("Could not create test volume path: %s : %s", btrfsTestVolumePath, err)
	}

	btrfsd, err := New()
	if err != nil {
		t.Fatalf("Unable to create btrfs driver: %v", err)
	}

	if c, err := btrfsd.Mount("unittest", btrfsTestVolumePath); err != nil {
		t.Fatalf("Could not create volume object :%s", err)
	} else {
		testFile := filepath.Join(btrfsTestVolumePath, "unittest", "test.txt")
		testData := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		testData2 := []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

		if err := ioutil.WriteFile(testFile, testData, 0664); err != nil {
			t.Fatalf("Could not write out test file: %s", err)
		}

		label := "unittest_foo"
		if err := c.Snapshot(label); err != nil {
			t.Fatalf("Could not snapshot: %s", err)
		}

		if err := ioutil.WriteFile(testFile, testData2, 0664); err != nil {
			t.Errorf("Could not write out test file 2: %s", err)
		}

		snapshots, _ := c.Snapshots()
		t.Logf("Found %v", snapshots)

		t.Logf("About to rollback %s", label)
		if err := c.Rollback(label); err != nil {
			t.Fatalf("Could not roll back: %s", err)
		}

		if output, err := ioutil.ReadFile(testFile); err != nil {
			t.Fatalf("Could not read back test file: %s", err)
		} else {
			if !reflect.DeepEqual(output, testData) {
				t.Logf("testdata: %v", testData)
				t.Logf("readdata: %v", output)
				t.FailNow()
			}
		}

		log.Printf("About to remove snapshot %s", label)
		if err := c.RemoveSnapshot(label); err != nil {
			t.Fatalf("Could not remove %s: %s", label, err)
		}

	}
}
