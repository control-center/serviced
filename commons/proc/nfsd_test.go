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

package proc

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
	"time"

	"github.com/zenoss/glog"
)

func setProcDir(dir string) {
	procDir = dir
}

func TestGetProcNFSDExport(t *testing.T) {

	// mock up our proc dir
	defer setProcDir(procDir)
	procDir = "tstproc/"

	expected := &ProcNFSDExports{
		MountPoint: "/exports/serviced_var",
		ClientOptions: map[string]NFSDExportOptions{
			"*": map[string]string{
				"wdelay":           "",
				"nohide":           "",
				"sec":              "1",
				"rw":               "",
				"insecure":         "",
				"no_root_squash":   "",
				"async":            "",
				"no_subtree_check": "",
				"uuid":             "45a148e9:89326106:00000000:00000000",
			},
		},
	}

	actual, err := GetProcNFSDExport(expected.MountPoint)
	if err != nil {
		t.Fatalf("could not get nfsd export: %s", err)
	}

	if !expected.Equals(actual) {
		t.Fatalf("expected: %+v != actual: %+v", expected, actual)
	}

	actualUUID := actual.ClientOptions["*"]["uuid"]
	expectedUUID := expected.ClientOptions["*"]["uuid"]
	if expectedUUID != actualUUID {
		t.Fatalf("expectedUUID: %+v != actualUUID: %+v", expectedUUID, actualUUID)
	}

	if _, err := GetProcNFSDExport("/doesnotexist"); err != ErrMountPointNotExported {
		t.Fatalf("expected could not get nfsd export: %s", err)
	}
}

func TestGetProcNFSDExports(t *testing.T) {

	// mock up our proc dir
	defer setProcDir(procDir)
	procDir = "tstproc/"

	mounts, err := GetProcNFSDExports()
	if err != nil {
		t.Fatalf("could not get nfsd exports: %s", err)
	}

	expected := &ProcNFSDExports{
		MountPoint: "/exports/serviced_var",
		ClientOptions: map[string]NFSDExportOptions{
			"*": map[string]string{
				"wdelay":           "",
				"nohide":           "",
				"sec":              "1",
				"rw":               "",
				"insecure":         "",
				"no_root_squash":   "",
				"async":            "",
				"no_subtree_check": "",
				"uuid":             "45a148e9:89326106:00000000:00000000",
			},
		},
	}
	actual := mounts[expected.MountPoint]
	if !expected.Equals(&actual) {
		t.Fatalf("expected: %+v != actual: %+v", expected, actual)
	}

	actualUUID := actual.ClientOptions["*"]["uuid"]
	expectedUUID := expected.ClientOptions["*"]["uuid"]
	if expectedUUID != actualUUID {
		t.Fatalf("expectedUUID: %+v != actualUUID: %+v", expectedUUID, actualUUID)
	}
}

func TestMonitorExportedVolume(t *testing.T) {
	// create temporary proc dir
	tmpVar, err := ioutil.TempDir("", "tstproc")
	if err != nil {
		t.Fatalf("could not create tempdir %+v: %s", tmpVar, err)
	}
	defer os.RemoveAll(tmpVar)

	// mock up our proc dir
	defer setProcDir(procDir)
	procDir = tmpVar + "/"

	nfsdDir := path.Join(procDir, path.Dir(procNFSDExportsFile))
	if err := os.MkdirAll(nfsdDir, 0755); err != nil {
		t.Fatalf("unable to mkdir %+v: %s", nfsdDir, err)
	}

	// populate the mocked up export file
	exportsLine := "#\n/exports/serviced_var   *(rw,insecure,no_root_squash,async,wdelay,nohide,no_subtree_check,uuid=45a148e9:89326106:00000000:00000000,sec=1)\n"
	exportsFile := path.Join(nfsdDir, path.Base(procNFSDExportsFile))
	glog.Infof("==== writing to exports %s: %s", exportsFile, exportsLine)
	if err := ioutil.WriteFile(exportsFile, []byte(exportsLine), 0600); err != nil {
		t.Fatalf("unable to write file %+v: %s", exportsFile, err)
	}

	expectedMountPoint := "/exports/serviced_var"
	actual, err := GetProcNFSDExport(expectedMountPoint)
	if err != nil {
		t.Fatalf("could not find expected mountpoint %s from file %s: %s", expectedMountPoint, exportsFile, err)
	}

	actualUUID := actual.ClientOptions["*"]["uuid"]
	expectedUUID := "45a148e9:89326106:00000000:00000000"
	if expectedUUID != actualUUID {
		t.Fatalf("expectedUUID: %+v != actualUUID: %+v", expectedUUID, actualUUID)
	}

	// monitor
	expectedIsExported := true
	processExportedVolumeChangeFunc := func(mountpoint string, actualIsExported bool) {
		glog.Infof("==== received exported volume info changed message - isExported:%s", actualIsExported)
		if expectedIsExported != actualIsExported {
			t.Fatalf("expectedIsExported: %+v != actualIsExported: %+v", expectedIsExported, actualIsExported)
		}
	}

	shutdown := make(chan interface{})
	defer close(shutdown)
	go MonitorExportedVolume(expectedMountPoint, time.Duration(2*time.Second), shutdown, processExportedVolumeChangeFunc)

	// wait some time
	waitTime := time.Second * 6
	time.Sleep(waitTime)

	// update the file
	glog.Infof("==== writing to exports %s: %s", exportsFile, exportsLine)
	if err := ioutil.WriteFile(exportsFile, []byte(exportsLine), 0600); err != nil {
		t.Fatalf("unable to write file %+v: %s", exportsFile, err)
	}

	// wait some time
	time.Sleep(waitTime)

	// remove the export
	expectedIsExported = false
	noEntries := "#\n"
	glog.Infof("==== clearing exports %s: %s", exportsFile, noEntries)
	if err := ioutil.WriteFile(exportsFile, []byte(noEntries), 0600); err != nil {
		t.Fatalf("unable to write file %+v: %s", exportsFile, err)
	}

	actual, err = GetProcNFSDExport(expectedMountPoint)
	if err != ErrMountPointNotExported {
		t.Fatalf("should not have found mountpoint %s from cleared file %s: %s", expectedMountPoint, exportsFile, err)
	}

	// wait some time
	time.Sleep(waitTime)
}
