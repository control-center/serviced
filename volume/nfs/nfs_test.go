// Copyright 2015 The Serviced Authors.
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

// +build integration

package nfs_test

import (
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/control-center/serviced/volume"
	. "github.com/control-center/serviced/volume/nfs"
	. "gopkg.in/check.v1"
)

// Wire in gocheck
func Test(t *testing.T) { TestingT(t) }

type NFSSuite struct {
}

var _ = Suite(&NFSSuite{})

// Make sure that we don't support things we don't support
func (s *NFSSuite) TestNFSDriver(c *C) {
	var (
		root   string
		driver *NFSDriver
	)

	root = c.MkDir()
	d, err := Init(root, []string{NetworkDisabled})
	c.Assert(err, IsNil)
	driver = d.(*NFSDriver)

	// Test initializing a bad directory
	d, err = Init(root+"notadir", nil)
	c.Assert(err, NotNil)
	c.Assert(d, IsNil)

	// Test initializing a file instead of a dir
	fname := filepath.Join(root, "iamafile")
	ioutil.WriteFile(fname, []byte("hi"), 0664)
	d, err = Init(fname, nil)
	c.Assert(err, Equals, volume.ErrNotADirectory)
	c.Assert(d, IsNil)

	vol, err := driver.Create("volume")
	c.Assert(vol, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	err = driver.Remove("volume")
	c.Assert(err, Equals, ErrNotSupported)

	status, err := driver.Status()
	c.Assert(status, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	volpath := filepath.Join(root, "testvolume")
	if err := os.MkdirAll(volpath, 0775); err != nil {
		c.Error(err)
	}

	vol, err = driver.GetTenant("volume")
	c.Assert(vol, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	volname := "testvolume"
	vol, err = driver.Get(volname)

	c.Assert(err, IsNil)
	c.Assert(vol, NotNil)
	c.Assert(vol.Path(), Equals, volpath)
	c.Assert(vol.Name(), Equals, volname)
	c.Assert(vol.Driver(), Equals, driver)
	c.Assert(vol.Tenant(), Equals, volname)

	wc, err := vol.WriteMetadata("", "")
	c.Assert(wc, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	rc, err := vol.ReadMetadata("", "")
	c.Assert(rc, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	c.Assert(vol.Snapshot("", "", []string{}), Equals, ErrNotSupported)

	info, err := vol.SnapshotInfo("")
	c.Assert(info, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	snaps, err := vol.Snapshots()
	c.Assert(snaps, IsNil)
	c.Assert(err, Equals, ErrNotSupported)

	c.Assert(vol.RemoveSnapshot(""), Equals, ErrNotSupported)
	c.Assert(vol.Rollback(""), Equals, ErrNotSupported)
	c.Assert(vol.Export("", "", nil), Equals, ErrNotSupported)
	c.Assert(vol.Import("", nil), Equals, ErrNotSupported)

	c.Assert(driver.Exists(volname), Equals, true)
	c.Assert(driver.List(), DeepEquals, []string{volname})
	c.Assert(driver.Release(volname), IsNil)
	c.Assert(driver.Cleanup(), IsNil)

}
