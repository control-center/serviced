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
	"fmt"
	"testing"
)

func TestGetProcNFSFSServers(t *testing.T) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = ""

	servers, err := GetProcNFSFSServers()
	if err != nil {
		t.Fatalf("could not get nfsfs/servers: %s", err)
	}

	expectedServer := &ProcNFSFSServer{
		Version:  "v4",
		ServerID: "0a57d1a8",
		Port:     "801",
		Hostname: "10.87.209.168",
	}
	key := fmt.Sprintf("%s:%s:%s", expectedServer.Version, expectedServer.ServerID, expectedServer.Port)
	actualServer := servers[key]
	if *expectedServer != actualServer {
		t.Fatalf("expected: %+v != actual: %+v", expectedServer, &actualServer)
	}
}

func TestGetProcNFSFSVolumes(t *testing.T) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = ""

	volumes, err := GetProcNFSFSVolumes()
	if err != nil {
		t.Fatalf("could not get nfsfs/volumes: %s", err)
	}

	expected := []ProcNFSFSVolume{
		{
			Version:  "v4",
			ServerID: "0a57d1a8",
			Port:     "801",
			DeviceID: "0:137",
			FSID:     "45a148e989326106",
			FSCache:  "no",
		},
		{
			Version:  "v3",
			ServerID: "0a57d1a8",
			Port:     "801",
			DeviceID: "0:138",
			FSID:     "45a148e989326106",
			FSCache:  "no",
		},
	}
	for idx := range expected {
		if expected[idx] != volumes[idx] {
			t.Fatalf("expected[%d]: %+v != actual[%d]: %+v", idx, expected[idx], idx, volumes[idx])
		}
	}
	if len(expected) != len(volumes) {
		t.Fatalf("len(expected): %+v != len(actual): %+v", len(expected), len(volumes))
	}
}

func TestGetMountInfo(t *testing.T) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = ""

	// mock up our findmnt command
	defer func(s string) {
		procFindmntCommand = s
	}(procFindmntCommand)
	procFindmntCommand = "BASH: grep %s self/mountinfo | awk '{n=split($NF,fields,\"=\"); print $3, $9, $10, $5, fields[n]}'"

	actual, err := GetMountInfo("/tmp/serviced/var")
	if err != nil {
		t.Fatalf("could not get mount info: %s", err)
	}

	expected := MountInfo{
		DeviceID:   "0:137",
		FSType:     "nfs4",
		RemotePath: "10.87.209.168:/serviced_var",
		LocalPath:  "/tmp/serviced/var",
		ServerIP:   "10.87.209.168",
	}
	if expected != *actual {
		t.Fatalf("expected: %+v != actual: %+v", expected, actual)
	}
}

func TestGetNFSVolumeInfo(t *testing.T) {

	// mock up our proc dir
	defer func(s string) {
		procDir = s
	}(procDir)
	procDir = ""

	// mock up our findmnt command
	defer func(s string) {
		procFindmntCommand = s
	}(procFindmntCommand)
	procFindmntCommand = "BASH: grep %s self/mountinfo | awk '{n=split($NF,fields,\"=\"); print $3, $9, $10, $5, fields[n]}'"

	actual, err := GetNFSVolumeInfo("/tmp/serviced/var")
	if err != nil {
		t.Fatalf("could not get mount info: %s", err)
	}

	expected := NFSMountInfo{
		DeviceID:   "0:137",
		FSType:     "nfs4",
		RemotePath: "10.87.209.168:/serviced_var",
		LocalPath:  "/tmp/serviced/var",
		ServerIP:   "10.87.209.168",

		Version:  "v4",
		ServerID: "0a57d1a8",
		Port:     "801",
		FSID:     "45a148e989326106",
		FSCache:  "no",
	}
	if expected != *actual {
		t.Fatalf("expected: %+v != actual: %+v", expected, actual)
	}
}
