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

// +build unit

package proc

import (
	"testing"
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
