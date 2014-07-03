// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package utils

import (
	"testing"
)

// Test getMemorySize()
func TestGetMemorySize(t *testing.T) {

	// alter the file getMemorySize() is looking at
	meminfoFile = "testfiles/meminfo"
	size, err := GetMemorySize()
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

// Test GetIPv4Addresses()
func TestGetIPv4Addresses(t *testing.T) {
	ips, err := GetIPv4Addresses()
	if err != nil {
		t.Errorf("Failed to get ipv4 addresses: %s", err)
		t.Fail()
	}

	expectedMinimumLen := 1
	if len(ips) < expectedMinimumLen {
		t.Errorf("minimum IPs expected %d > retrieved %d  ips:%v", expectedMinimumLen, len(ips), ips)
		t.Fail()
	}
}
