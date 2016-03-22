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

package nfs

import (
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/control-center/serviced/commons/proc"
	"github.com/control-center/serviced/validation"
)

var (
	ErrTestMountError   = errors.New("Error mounting volume")
	ErrTestUnMountError = errors.New("Error un-mounting volume")
	ErrTestMountNotNFS  = errors.New("Mount not NFS")
)

type mockCommand struct {
	name   string
	args   []string
	output []byte
	err    error
}

func (c *mockCommand) CombinedOutput() ([]byte, error) {
	return c.output, c.err
}

type mountTestCaseT struct {
	remote   string
	local    string
	expected error
}

var mountTestCases = []mountTestCaseT{
	mountTestCaseT{"127.0.0.1:/tmp", "/test", nil},
	mountTestCaseT{"127.0sf1:/tmp", "/test", ErrMalformedNFSMountpoint},
	mountTestCaseT{"127.0.0.1:tmp", "/test", ErrMalformedNFSMountpoint},
}

type mockDriver struct {
	MountInfo    *proc.NFSMountInfo
	isInstalled  bool
	isMounted    bool
	mountError   error
	unMountError error
}

func (d *mockDriver) Installed() error {
	if !d.isInstalled {
		return ErrNfsMountingUnsupported
	}

	return nil
}

func (d *mockDriver) Info(localPath string, info *proc.NFSMountInfo) error {
	if !d.isMounted {
		return proc.ErrMountPointNotFound
	}

	if localPath != d.MountInfo.LocalPath {
		return proc.ErrMountPointNotFound
	}

	*info = *d.MountInfo

	if !strings.HasPrefix(info.FSType, "nfs") {
		return ErrTestMountNotNFS
	}

	return nil
}

func (d *mockDriver) Mount(_, _ string, _ time.Duration) error {
	d.isMounted = true
	return d.mountError
}

func (d *mockDriver) Unmount(_ string) error {
	d.isMounted = false
	return d.unMountError
}

func TestMount_NotInstalled(t *testing.T) {
	d := mockDriver{isInstalled: false}
	if err := Mount(&d, "remote", "local"); err != ErrNfsMountingUnsupported {
		t.Errorf("expected %s; got %s", ErrNfsMountingUnsupported, err)
	}
}

func TestMount_BadRemotePath(t *testing.T) {
	d := mockDriver{isInstalled: true}
	if err := Mount(&d, "remote", "local"); err != ErrMalformedNFSMountpoint {
		t.Errorf("expected %s; got %s", ErrMalformedNFSMountpoint, err)
	}

	if err := Mount(&d, "127.0.0.1", "local"); err != ErrMalformedNFSMountpoint {
		t.Errorf("expected %s; got %s", ErrMalformedNFSMountpoint, err)
	}

	if err := Mount(&d, "127.0.0.1:/", "local"); err != ErrMalformedNFSMountpoint {
		t.Errorf("expected %s; got %s", ErrMalformedNFSMountpoint, err)
	}
}

func TestMount_BadValidation(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true}

	// incompatible fs type
	info1 := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs3"},
	}
	d.MountInfo = &info1

	if err := Mount(&d, info1.RemotePath, info1.LocalPath); err != nil {
		if _, ok := err.(*validation.ValidationError); !ok {
			t.Errorf("expected validation error, got %s", err)
		}
	} else {
		t.Errorf("expected validation, got nil")
	}

	// incompatible fsid
	info2 := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	if err := Mount(&d, info2.RemotePath, info2.LocalPath); err != nil {
		if _, ok := err.(*validation.ValidationError); !ok {
			t.Errorf("expected validation error, got %s", err)
		}
	} else {
		t.Errorf("expected validation, got nil")
	}
}

func TestMount_Remount_FailUnMount(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true, unMountError: ErrTestUnMountError, mountError: nil}

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	d.MountInfo = &info

	staleNFSCheck = func(string) bool { return true }

	if err := Mount(&d, info.RemotePath, info.LocalPath); err != ErrTestUnMountError {
		t.Errorf("expected error %s, got error %s", ErrTestUnMountError, err)
	}
}

func TestMount_Remount_FailMount(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true, unMountError: nil, mountError: ErrTestMountError}

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	d.MountInfo = &info

	staleNFSCheck = func(string) bool { return true }

	if err := Mount(&d, info.RemotePath, info.LocalPath); err != ErrTestMountError {
		t.Errorf("expected error %s, got error %s", ErrTestMountError, err)
	}
}

func TestMount_Remount_Success(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true, unMountError: nil, mountError: nil}

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	d.MountInfo = &info

	staleNFSCheck = func(string) bool { return true }

	if err := Mount(&d, info.RemotePath, info.LocalPath); err != nil {
		t.Errorf("got error %s", err)
	}
}

func TestMount_Success(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: false}

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	d.MountInfo = &info

	if err := Mount(&d, info.RemotePath, info.LocalPath); err != nil {
		t.Errorf("got error %s", err)
	}
}

func TestUnmount_NotInstalled(t *testing.T) {
	d := mockDriver{isInstalled: false}
	if err := Unmount(&d, "local"); err != ErrNfsMountingUnsupported {
		if err == nil {
			t.Errorf("got no error")
		} else {
			t.Errorf("expected '%s'; got '%s'", ErrNfsMountingUnsupported, err)
		}
	}
}

func TestUnmount_NotMounted(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: false}

	localPath := "/tmp/path/umounttest"

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: localPath, FSType: "nfs4"},
	}
	d.MountInfo = &info

	if err := Unmount(&d, localPath); err != proc.ErrMountPointNotFound {
		t.Errorf("expected '%s'; got '%s'", proc.ErrMountPointNotFound, err)
	}
}

func TestUnmount_BadFS(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true}

	localPath := "/tmp/path/umounttest"

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: localPath, FSType: "btrfs"},
	}
	d.MountInfo = &info

	if err := Unmount(&d, localPath); err != ErrTestMountNotNFS {
		t.Errorf("expected '%s'; got '%s'", ErrTestMountNotNFS, err)
	}
}

func TestUnmount_UnmountSubdir(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true}

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: "/tmp/path", FSType: "nfs4"},
	}
	d.MountInfo = &info

	if err := Unmount(&d, "/tmp/path/subdir"); err != proc.ErrMountPointNotFound {
		t.Errorf("expected '%s'; got '%s'", proc.ErrMountPointNotFound, err)
	}
}

func TestUnmount_Success(t *testing.T) {
	d := mockDriver{isInstalled: true, isMounted: true}

	localPath := "/tmp/path/umounttest"

	info := proc.NFSMountInfo{
		MountInfo: proc.MountInfo{RemotePath: "127.0.0.1:/tmp/path", LocalPath: localPath, FSType: "nfs4"},
	}
	d.MountInfo = &info

	if err := Unmount(&d, localPath); err != nil {
		t.Errorf("got error: %s", err)
	}
}
