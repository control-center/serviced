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
