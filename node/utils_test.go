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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// +build integration

package node

import (
	"testing"

	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/zenoss/glog"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	. "gopkg.in/check.v1"
)

var _ = Suite(&ZZKTest{})

type ZZKTest struct {
	zzk.ZZKTestSuite
}

func Test(t *testing.T) {
	TestingT(t)
}

// Test validOwnerSpec
func TestValidOwnerSpec(t *testing.T) {

	invalidSpecs := []string{
		"",
		":",
		".test:test",
		"test:.test",
		"test,test",
	}
	for _, spec := range invalidSpecs {
		if validOwnerSpec(spec) {
			t.Logf("%s should NOT be a valid owner spec", spec)
			t.Fail()
		}
	}
	validSpecs := []string{
		"mysql:mysql",
		"root:root",
		"user.name:group.name",
		"user-name:group-name",
	}
	for _, spec := range validSpecs {
		if !validOwnerSpec(spec) {
			t.Logf("%s should be a valid owner spec", spec)
			t.Fail()
		}
	}
}

func getTestConn(c *C, path string) client.Connection {
	root, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)
	err = root.CreateDir(path)
	c.Assert(err, IsNil)
	conn, err := zzk.GetLocalConnection(path)
	c.Assert(err, IsNil)
	return conn
}

// Test createVolumeDir
func (ts *ZZKTest) TestCreateVolumeDir(t *C) {
	// create temporary proc dir
	tmpPath, err := ioutil.TempDir("", "node_util")
	if err != nil {
		t.Fatalf("could not create tempdir %+v: %s", tmpPath, err)
	}
	defer os.RemoveAll(tmpPath)

	if err := os.MkdirAll(tmpPath, 0755); err != nil {
		t.Fatalf("unable to mkdir %+v: %s", tmpPath, err)
	}

	type volumeSpec struct {
		hostPath      string
		containerPath string
		image         string
		user          string
		perm          string
	}
	v := volumeSpec{
		hostPath:      path.Join(tmpPath, "actual_share_doc"),
		containerPath: "/usr/share/dpkg", // do not use a dir with symlinks that point outside the path
		image:         "ubuntu:latest",
		user:          "games:games",
		perm:          "755",
	}

	conn := getTestConn(t, "/vinit")
	if err := createVolumeDir(conn, v.hostPath, v.containerPath, v.image, v.user, v.perm); err != nil {
		t.Fatalf("unable to create volume %+v: %s", tmpPath, err)
	}

	// retrieve containerPath from image
	expectedPath := path.Join(tmpPath, "expected_share_doc")
	containerMount := "/mnt/dfs"
	copyCommand := [...]string{
		"docker", "run",
		"--rm",
		"-v", expectedPath + ":" + containerMount,
		v.image,
		"bash", "-c", fmt.Sprintf("shopt -s nullglob && shopt -s dotglob && cp -pr %s/* %s/\n", v.containerPath, containerMount),

		// FIXME: use rsync instead of cp to use a different command to copy
		// "bash", "-c", fmt.Sprintf("apt-get -y install rsync; rsync -a %s/ %s/\n", v.containerPath, containerMount),
	}

	glog.V(2).Infof("copy command: %s", copyCommand)
	docker := exec.Command(copyCommand[0], copyCommand[1:]...)
	if output, err := docker.CombinedOutput(); err != nil {
		t.Fatalf("could not create host volume: %+v, %s", copyCommand, string(output))
	}

	// compare rsync'ed path from image against DFS volume
	compareCmd := [...]string{"diff", "-qr", v.hostPath, expectedPath}
	glog.V(2).Infof("compare command: %s", compareCmd)
	docker = exec.Command(compareCmd[0], compareCmd[1:]...)
	if output, err := docker.CombinedOutput(); err != nil {
		t.Fatalf("could not compare paths: %+v, %s", compareCmd, string(output))
	}

	// compare user:group perms
	getUidGidCmd := [...]string{"docker", "run", "--rm", v.image, "getent", "passwd", "games"}
	glog.V(2).Infof("get command: %s", getUidGidCmd)
	docker = exec.Command(getUidGidCmd[0], getUidGidCmd[1:]...)
	if output, err := docker.CombinedOutput(); err != nil {
		t.Fatalf("could not get uid/gid: %+v, %s", getUidGidCmd, string(output))
	} else {
		parts := strings.Split(string(output), ":")
		expectedUID := parts[2]
		expectedGID := parts[3]

		fileinfo, err := os.Stat(v.hostPath)
		if err != nil {
			t.Fatalf("could not stat dir: %+v, %s", v.hostPath, err)
		}
		actualUID := fileinfo.Sys().(*syscall.Stat_t).Uid
		actualGID := fileinfo.Sys().(*syscall.Stat_t).Gid

		if expectedUID != fmt.Sprintf("%d", actualUID) {
			t.Fatalf("actualUID:%+v != expectedUID:%+v", actualUID, expectedUID)
		}
		if expectedGID != fmt.Sprintf("%d", actualGID) {
			t.Fatalf("actualGID:%+v != expectedGID:%+v", actualGID, expectedGID)
		}
	}

	// make sure initialized dotfile exists
	dotfileHostPath := path.Join(filepath.Dir(v.hostPath), fmt.Sprintf(".%s.serviced.initialized", filepath.Base(v.hostPath)))
	if _, err := os.Stat(dotfileHostPath); err != nil {
		t.Fatalf("could not stat serviced initialized dotfile %s: %s", dotfileHostPath, err)
	}
}
