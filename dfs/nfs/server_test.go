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

package nfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"syscall"
	"testing"
)

func TestMntArgs(t *testing.T) {
	name, args := mntArgs("/opt/serviced/var", "/exports/serviced_var", "", "bind")
	if syscall.Getuid() == 0 {
		if name != "mount" {
			t.Fatalf("as root, expected name to be 'mount' got '%s'", name)
		}
		expectedArgs := []string{"-o", "bind", "/opt/serviced/var", "/exports/serviced_var"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("got %+v expected %+v", args, expectedArgs)
		}
	} else {
		if name != "sudo" {
			t.Fatalf("as non-root, expected name to be 'sudo' got '%s'", name)
		}
		expectedArgs := []string{"mount", "-o", "bind", "/opt/serviced/var", "/exports/serviced_var"}
		if !reflect.DeepEqual(args, expectedArgs) {
			t.Fatalf("got %+v expected %+v", args, expectedArgs)
		}
	}
}

func dirExists(path string) (bool, error) {
	s, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return s.IsDir(), err
}

var expectedExports = "%s\t192.168.1.0/24(rw,fsid=0,no_root_squash,insecure,no_subtree_check,async)\n%s/foo\t192.168.1.0/24(rw,no_root_squash,nohide,insecure,no_subtree_check,async)\n\n"

func TestNewServer(t *testing.T) {
	tempDir, err := ioutil.TempDir("", "nfs_unit_tests_")
	if err != nil {
		t.Fatalf("unexpected error: %s", err)
	}
	defer os.RemoveAll(tempDir)
	t.Logf("created temp dir: %s", tempDir)

	baseDir := path.Join(tempDir, "baseDir")

	// mock out the exports directory, use stack to hold old values
	defer func(e, hostsDeny, hostsAllow, exports, exportsd string) {
		// restore to original values
		exportsPath = e
		etcHostsDeny = hostsDeny
		etcHostsAllow = hostsAllow
		etcExports = exports
		exportsDir = exportsd
	}(exportsPath, etcHostsDeny, etcHostsAllow, etcExports, exportsDir)
	exportsPath = path.Join(tempDir, "exports")
	etcHostsDeny = path.Join(tempDir, "etc/hosts.deny")
	etcHostsAllow = path.Join(tempDir, "etc/hosts.allow")
	etcExports = path.Join(tempDir, "etc/exports")
	exportsDir = path.Join(tempDir, "exports")

	// neuter bindmount during tests
	bindMount = func(string, string) error {
		return nil
	}

	defer func(f func() error) {
		reload = f
	}(reload)
	reload = func() error {
		return nil
	}
	defer func(f func() error) {
		start = f
	}(start)
	start = reload

	// create our test server
	s, err := NewServer(baseDir, "foo", "192.168.1.0/24")
	if err != nil {
		t.Fatalf("unexpected error : %s ", err)
	}

	// check that the required directories were created
	if exists, err := dirExists(baseDir); err != nil || !exists {
		t.Fatalf("baseDir dir does not exist: %s, %s", baseDir, err)
	}
	exportDir := path.Join(exportsPath, "foo")
	if exists, err := dirExists(exportDir); err != nil || !exists {
		t.Fatalf("export dir does not exist: %s, %s", exportDir, err)
	}

	// we call .Sync() repeatedly, lets make a shortcut
	sync := func() {
		if err := s.Sync(); err != nil {
			t.Fatalf("unexpected error synching server: %s", err)
		}
	}
	sync()

	// assert that the defaults get written out
	assertFileContents(t, etcHostsDeny, []byte(hostDenyDefaults))
	assertFileContents(t, etcHostsAllow, []byte(hostAllowDefaults+" \n\n"))

	s.SetClients("192.168.1.21")
	sync()

	assertFileContents(t, etcHostsDeny, []byte(hostDenyDefaults))
	assertFileContents(t, etcHostsAllow, []byte(hostAllowDefaults+" 192.168.1.21\n\n"))

	s.SetClients("192.168.1.21", "192.168.1.20")
	sync()

	assertFileContents(t, etcHostsDeny, []byte(hostDenyDefaults))
	assertFileContents(t, etcHostsAllow, []byte(hostAllowDefaults+" 192.168.1.20 192.168.1.21\n\n"))

	assertFileContents(t, etcExports, []byte(fmt.Sprintf(expectedExports, exportsPath, exportsPath)))

}

func assertFileContents(t *testing.T, filename string, contents []byte) {
	bytes, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatalf("unexpected failure reading %s: %s", filename, err)
	}
	if string(bytes) != string(contents) {
		t.Fatalf("got [%d]:\n '%+v'' \n\n expected [%d]:\n '%+v'", len(bytes), string(bytes), len(contents), string(contents))
	}
}
