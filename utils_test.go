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

// Test NewUuid()
func TestNewUuid(t *testing.T) {

	urandomFilename = "testfiles/urandom_bytes"

	uuid, err := NewUuid()
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
	expectedUuid := "1102c395-e94b-0a08-d1e9-307e31a5213e"
	if uuid != expectedUuid {
		t.Errorf("uuid: expected %s, got %s", expectedUuid, uuid)
		t.Fail()
	}
}

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
