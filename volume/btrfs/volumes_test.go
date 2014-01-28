/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*******************************************************************************/

package btrfs

import (
	"github.com/zenoss/glog"

	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"os/user"
	"reflect"
	"testing"
)

var testVolumePath = "/var/lib/serviced"

const btrfsTestVolumePathEnv = "SERVICED_BTRFS_TEST_VOLUME_PATH"

func init() {
	testVolumePathEnv := os.Getenv(btrfsTestVolumePathEnv)
	if len(testVolumePathEnv) > 0 {
		testVolumePath = testVolumePathEnv
	}
}

func TestVolumes(t *testing.T) {

	if user, err := user.Current(); err != nil {
		panic(err)
	} else {
		if user.Uid != "0" {
			t.Skip("Skipping BTRFS tests because we are not running as root")
		}
	}

	if _, err := exec.LookPath("btrfs"); err != nil {
		t.Skip("Skipping BTRFS tests because the btrfs-tools we not found in the path")
	}

	glog.Infof("Using '%s' as btrfs test volume, use env '%s' to override.",
		testVolumePath, btrfsTestVolumePathEnv)

	if err := os.MkdirAll(testVolumePath, 0775); err != nil {
		log.Printf("Could not create test volume path: %s : %s", testVolumePath, err)
		t.FailNow()
	}

	if v, err := NewVolume(testVolumePath, "unittest"); err != nil {
		log.Printf("Could not create volume object :%s", err)
		t.FailNow()
	} else {
		testFile := "/var/lib/serviced/unittest/test.txt"
		testData := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}
		testData2 := []byte{0, 1, 2, 3, 4, 5, 6, 7, 8, 9}

		if err := ioutil.WriteFile(testFile, testData, 0664); err != nil {
			log.Printf("Could not write out test file: %s", err)
			t.FailNow()
		}

		label, err := v.Snapshot()
		if err != nil {
			log.Printf("Could not snapshot: %s", err)
			t.FailNow()
		}

		if err := ioutil.WriteFile(testFile, testData2, 0664); err != nil {
			log.Printf("Could not write out test file 2: %s", err)
			t.FailNow()
		}

		snapshots, err := v.Snapshots()
		log.Printf("Found %v", snapshots)

		log.Printf("About to rollback %s", label)
		if err := v.Rollback(label); err != nil {
			log.Printf("Could not roll back: %s", err)
			t.FailNow()
		}

		if output, err := ioutil.ReadFile(testFile); err != nil {
			log.Printf("Could not read back test file: %s", err)
			t.FailNow()
		} else {
			if !reflect.DeepEqual(output, testData) {
				log.Printf("testdata: %v", testData)
				log.Printf("readdata: %v", output)
				t.FailNow()
			}
		}

		log.Printf("About to remove snapshot %s", label)
		if err := v.RemoveSnapshot(label); err != nil {
			log.Printf("Could not remove %s: %s", label, err)
			t.FailNow()
		}

	}
}
