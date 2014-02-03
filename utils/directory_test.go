/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/
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
