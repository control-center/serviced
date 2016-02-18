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
