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

	size, err := GetMemorySize()
	if err != nil {
		t.Errorf("Failed to retrieve RAM value: %s", err)
		t.Fail()
	}
	if size < 1 {
		t.Errorf("expected non-zero value, received %d ", size)
		t.Fail()
	}
	t.Logf("memory size = %d", size)
}
