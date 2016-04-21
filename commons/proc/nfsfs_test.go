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
	"fmt"
	. "gopkg.in/check.v1"
)

func (s *TestProcSuite) TestGetProcNFSFSServers(c *C) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = "tstproc/"

	servers, err := GetProcNFSFSServers()
	c.Assert(err, IsNil)

	expectedServer := &ProcNFSFSServer{
		Version:  "v4",
		ServerID: "0a57d1a8",
		Port:     "801",
		Hostname: "10.87.209.168",
	}
	key := fmt.Sprintf("%s:%s:%s", expectedServer.Version, expectedServer.ServerID, expectedServer.Port)
	actualServer := servers[key]
	c.Assert(actualServer, Equals, *expectedServer)
}

func (s *TestProcSuite) TestGetMountInfo(c *C) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = "tstproc/"

	// mock up our findmnt command
	defer func(s string) {
		procFindmntCommand = s
	}(procFindmntCommand)
	procFindmntCommand = "grep %s tstproc/self/mountinfo | awk '{print $3, $9, $10, $5, $NF}'"

	actual, err := GetMountInfo("/tmp/serviced/var")
	c.Assert(err, IsNil)

	expected := MountInfo{
		DeviceID:   "0:137",
		FSType:     "nfs4",
		RemotePath: "10.87.209.168:/serviced_var",
		LocalPath:  "/tmp/serviced/var",
		ServerIP:   "10.87.209.168",
	}
	c.Assert(*actual, Equals, expected)
}

func (s *TestProcSuite) TestGetNFSVolumeInfo(c *C) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = "tstproc/"

	// mock up our findmnt command
	defer func(s string) {
		procFindmntCommand = s
	}(procFindmntCommand)
	procFindmntCommand = "grep %s tstproc/self/mountinfo | awk '{print $3, $9, $10, $5, $NF}'"

	// mock up our ReadFSIDFromMount command
	actual, err := GetNFSVolumeInfo("/tmp/serviced/var")
	c.Assert(err, IsNil)

	minfo := MountInfo{
		DeviceID:   "0:137",
		FSType:     "nfs4",
		RemotePath: "10.87.209.168:/serviced_var",
		LocalPath:  "/tmp/serviced/var",
		ServerIP:   "10.87.209.168",
	}

	expected := NFSMountInfo{
		MountInfo: minfo,
	}
	c.Assert(*actual, Equals, expected)
}
