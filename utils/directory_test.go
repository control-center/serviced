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
	"os"
	"strings"
	"testing"
)

func TestLocalDir(t *testing.T) {
	original := os.Getenv("SERVICED_HOME")
	// make sure we clean up after ourselves
	defer func() {
		os.Setenv("SERVICED_HOME", original)
	}()

	os.Setenv("SERVICED_HOME", "test")
	testDir := LocalDir("test")
	if testDir != "test/test" {
		t.Errorf("Expected test directory to be test/test instead it was %s", testDir)
	}
}

func TestResourcesDir(t *testing.T) {
	original := os.Getenv("SERVICED_HOME")
	// make sure we clean up after ourselves
	defer func() {
		os.Setenv("SERVICED_HOME", original)
	}()

	os.Setenv("SERVICED_HOME", "test")
	testDir := ResourcesDir()
	if testDir != "test/isvcs/resources" {
		t.Errorf("Resources directory was an unexpected value  %s", testDir)
	}
}

func TestDefaultDir(t *testing.T) {
	original := os.Getenv("SERVICED_HOME")
	// make sure we clean up after ourselves
	defer func() {
		os.Setenv("SERVICED_HOME", original)
	}()

	os.Setenv("SERVICED_HOME", "")
	testDir := LocalDir("test")
	if !strings.Contains(testDir, "/serviced/") {
		t.Errorf("Making sure the local directory includes serviced	 %s", testDir)
	}

	if strings.Contains(testDir, "utils") {
		t.Errorf("test %s should not have the string utils in it since it should be from the directory above utils", testDir)
	}
}
