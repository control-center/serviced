/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"testing"
)

// Test getMemorySize()
func TestGetMemorySize(t *testing.T) {

	// alter the file getMemorySize() is looking at
	meminfoFile = "testfiles/meminfo"
	size, err := getMemorySize()
	if err != nil {
		t.Errorf("Failed to parse memory file: %s", err)
		t.Fail()
	}
	expectedSize := uint64(33660776448)
	if size != expectedSize {
		t.Errorf("expected %d, received %d ", expectedSize, size)
		t.Fail()
	}
}

// Test parsing docker version
func Test_parseDockerVersion(t *testing.T) {

	const exampleOutput = `Client version: 0.6.6
Go version (client): go1.2rc3
Git commit (client): 6d42040
Server version: 0.6.6
Git commit (server): 6d42040
Go version (server): go1.2rc3
Last stable version: 0.6.6
`
	exampleVersion := DockerVersion{
		Client: []int{0, 6, 6},
		Server: []int{0, 6, 6},
	}

	version, err := parseDockerVersion(exampleOutput)
	if err != nil {
		t.Fatalf("Problem parsing example docker version: %s", err)
	}
	if !version.equals(&exampleVersion) {
		t.Fatalf("unexpected version: %s vs %s", version, exampleVersion)
	}
}
