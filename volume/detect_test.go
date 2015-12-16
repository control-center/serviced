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

// +build integration, root

package volume_test

import (
	. "github.com/control-center/serviced/volume"
	_ "github.com/control-center/serviced/volume/btrfs"
	_ "github.com/control-center/serviced/volume/devicemapper"
	_ "github.com/control-center/serviced/volume/rsync"
	. "gopkg.in/check.v1"
)

type AutodetectSuite struct {
	dir string
}

var (
	_ = Suite(&AutodetectSuite{})
)

func (s *AutodetectSuite) SetUpTest(c *C) {
	s.dir = c.MkDir()
}

func (s *AutodetectSuite) TearDownTest(c *C) {
	ShutdownAll()
}

func (s *AutodetectSuite) TestPreexistingBtrfs(c *C) {
	root := CreateBtrfsTmpVolume(c, 32*1024*1024)
	defer CleanupTmpVolume(c, root)

	// Initialize the driver and create a btrfs volume
	err := InitDriver(DriverTypeBtrFS, root, []string{})
	c.Check(err, IsNil)
	vol, err := Mount("testvolume", root)
	c.Assert(vol.Driver().DriverType(), Equals, DriverTypeBtrFS)
	c.Assert(err, IsNil)

	// Shut down the driver
	ShutdownDriver(root)

	// Try to initialize an rsync driver
	err = InitDriver(DriverTypeRsync, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to initialize a devicemapper driver
	err = InitDriver(DriverTypeDeviceMapper, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to reinitialize a btrfs driver
	err = InitDriver(DriverTypeBtrFS, root, []string{})
	c.Assert(err, IsNil)
}

func (s *AutodetectSuite) TestPreexistingDeviceMapper(c *C) {
	root := c.MkDir()

	// Initialize the driver and create a btrfs volume
	err := InitDriver(DriverTypeDeviceMapper, root, []string{})
	c.Check(err, IsNil)
	vol, err := Mount("testvolume", root)
	c.Assert(vol.Driver().DriverType(), Equals, DriverTypeDeviceMapper)
	c.Assert(err, IsNil)

	// Shut down the driver
	ShutdownDriver(root)

	// Try to initialize a btrfs driver
	err = InitDriver(DriverTypeBtrFS, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to initialize an rsync driver
	err = InitDriver(DriverTypeRsync, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to reinitialize a devicemapper driver
	err = InitDriver(DriverTypeDeviceMapper, root, []string{})
	c.Assert(err, IsNil)
}

func (s *AutodetectSuite) TestPreexistingRsync(c *C) {
	root := c.MkDir()

	// Initialize the driver and create a btrfs volume
	err := InitDriver(DriverTypeRsync, root, []string{})
	c.Check(err, IsNil)
	vol, err := Mount("testvolume", root)
	c.Assert(vol.Driver().DriverType(), Equals, DriverTypeRsync)
	c.Assert(err, IsNil)

	// Shut down the driver
	ShutdownDriver(root)

	// Try to initialize a btrfs driver
	err = InitDriver(DriverTypeBtrFS, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to initialize a devicemapper driver
	err = InitDriver(DriverTypeDeviceMapper, root, []string{})
	c.Assert(err, ErrorMatches, ErrDriverAlreadyInit.Error())

	// Try to reinitialize an rsync driver
	err = InitDriver(DriverTypeRsync, root, []string{})
	c.Assert(err, IsNil)
}
