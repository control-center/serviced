// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package rsync

import (
	"github.com/zenoss/glog"

	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path"
	"reflect"
	"testing"
)

var rsyncTestVolumePath = os.TempDir()

const rsyncTestVolumePathEnv = "SERVICED_RSYNC_TEST_VOLUME_PATH"

func init() {
	testVolumePathEnv := os.Getenv(rsyncTestVolumePathEnv)
	if len(testVolumePathEnv) > 0 {
		rsyncTestVolumePath = testVolumePathEnv
	}
}

func TestRsyncVolume(t *testing.T) {

	if _, err := exec.LookPath("rsync"); err != nil {
		t.Skip("Skipping rsync volume test, rsync not found in path")
	}

	glog.Infof("Using '%s' as rsync test volume, use env '%s' to override.",
		rsyncTestVolumePath, rsyncTestVolumePathEnv)

	if output, err := exec.Command("sh", "rm", "-Rf", path.Join(rsyncTestVolumePath, "unittest*")).CombinedOutput(); err != nil {
		log.Printf("Could not delete previous test volume: %s", string(output))
	}

	if err := os.MkdirAll(rsyncTestVolumePath, 0775); err != nil {
		log.Printf("Could not create test volume path: %s : %s", rsyncTestVolumePath, err)
		t.FailNow()
	}

	if v, err := (&RsyncVolume{}).New(rsyncTestVolumePath, "unittest"); err != nil {
		log.Printf("Could not create volume object :%s", err)
		t.FailNow()
	} else {
		testFile := path.Join(rsyncTestVolumePath, "unittest", "test.txt")
		testData := []byte("testData\n")
		testData2 := []byte("testData2\n")

		if err := ioutil.WriteFile(testFile, testData, 0664); err != nil {
			log.Printf("Could not write out test file: %s", err)
			t.FailNow()
		}

		label := "unittest_foo"
		if err := v.Snapshot(label); err != nil {
			log.Printf("Could not snapshot: %s", err)
			t.FailNow()
		}

		if err := ioutil.WriteFile(testFile, testData2, 0664); err != nil {
			log.Printf("Could not write out test file 2: %s", err)
			t.FailNow()
		}

		snapshots, _ := v.Snapshots()
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
