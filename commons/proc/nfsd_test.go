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
	. "gopkg.in/check.v1"
)

func setProcDir(dir string) {
	procDir = dir
}

func (s *TestProcSuite) TestGetProcNFSDExport(c *C) {

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
	c.Assert(err, IsNil)
	c.Assert(expected.Equals(actual), Equals, true)

	actualUUID := actual.ClientOptions["*"]["uuid"]
	expectedUUID := expected.ClientOptions["*"]["uuid"]
	c.Assert(actualUUID, Equals, expectedUUID)

	_, err = GetProcNFSDExport("/doesnotexist")
	c.Assert(err, Equals, ErrMountPointNotExported)
}

func (s *TestProcSuite) TestGetProcNFSDExports(c *C) {

	// mock up our proc dir
	defer setProcDir(procDir)
	procDir = "tstproc/"

	mounts, err := GetProcNFSDExports()
	c.Assert(err, IsNil)

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
	c.Assert(expected.Equals(&actual), Equals, true)

	actualUUID := actual.ClientOptions["*"]["uuid"]
	expectedUUID := expected.ClientOptions["*"]["uuid"]
	c.Assert(err, IsNil)
	c.Assert(actualUUID, Equals, expectedUUID)
}
