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
	"reflect"
	"testing"
)

var btrfsTestVolumePath = "/var/lib/serviced"

const btrfsTestVolumePathEnv = "SERVICED_BTRFS_TEST_VOLUME_PATH"

func init() {
	testVolumePathEnv := os.Getenv(btrfsTestVolumePathEnv)
	if len(testVolumePathEnv) > 0 {
		btrfsTestVolumePath = testVolumePathEnv
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

	t.Logf("Using '%s' as btrfs test volume, use env '%s' to override.",
		btrfsTestVolumePath, btrfsTestVolumePathEnv)

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
		testFile := "/var/lib/serviced/unittest/test.txt"
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
