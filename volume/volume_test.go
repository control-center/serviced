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

// +build integration

package volume_test

import (
	"path/filepath"
	"testing"

	. "github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func TestDriver(t *testing.T) { TestingT(t) }

type DriverSuite struct {
	drv *mocks.Driver
	vol *mocks.Volume
	dir string
}

var (
	_ = Suite(&DriverSuite{})

	drvArgs                 = make([]string, 0)
	drvName      DriverType = "mock"
	unregistered DriverType = "unregistered"
)

func (s *DriverSuite) MockInit(rootDir string, _ []string) (Driver, error) {
	s.drv.On("Root").Return(rootDir)
	return s.drv, nil
}

func (s *DriverSuite) SetUpTest(c *C) {
	s.dir = c.MkDir()
	s.drv = &mocks.Driver{}
	s.vol = &mocks.Volume{}
	Register(drvName, s.MockInit)
}

func (s *DriverSuite) TearDownTest(c *C) {
	s.drv.On("DriverType").Return(drvName)
	Unregister(drvName)
}

func (s *DriverSuite) TestNilRegistration(c *C) {
	err := Register("nilregistration", nil)
	c.Assert(err, ErrorMatches, ErrInvalidDriverInit.Error())
}

func (s *DriverSuite) TestRedundantRegistration(c *C) {
	err := Register(drvName, s.MockInit)
	c.Assert(err, ErrorMatches, ErrDriverExists.Error())
}

func (s *DriverSuite) TestRegistration(c *C) {
	err := InitDriver(drvName, s.dir, drvArgs)
	c.Assert(err, IsNil)
	driver, err := GetDriver(s.dir)
	c.Assert(err, IsNil)
	c.Assert(driver, FitsTypeOf, s.drv)
}

func (s *DriverSuite) TestUnsupported(c *C) {
	err := InitDriver(unregistered, s.dir, drvArgs)
	c.Assert(err, Equals, ErrDriverNotSupported)
	driver, err := GetDriver(s.dir)
	c.Assert(err, Equals, ErrDriverNotInit)
	c.Assert(driver, IsNil)
}

func (s *DriverSuite) TestFindMountNoDriver(c *C) {
	volPath := "/this/is/a/test/volume"
	v, err := FindMount(volPath)
	c.Assert(err, Equals, ErrDriverNotInit)
	c.Assert(v, IsNil)
}

func (s *DriverSuite) TestFindMountDriver(c *C) {
	volPath := filepath.Join(s.dir, "test", "volume")
	s.drv.On("Exists", "test/volume").Return(false)
	s.drv.On("Create", "test/volume").Return(s.vol, nil)
	err := InitDriver(drvName, s.dir, drvArgs)
	c.Assert(err, IsNil)
	v, err := FindMount(volPath)
	s.drv.AssertExpectations(c)
	c.Assert(v, Equals, s.vol)
	c.Assert(err, IsNil)
}

func (s *DriverSuite) TestMountWhenDoesNotExist(c *C) {
	volname := "testvolume"
	s.drv.On("Exists", volname).Return(false)
	s.drv.On("Create", volname).Return(s.vol, nil)
	err := InitDriver(drvName, s.dir, drvArgs)
	c.Assert(err, IsNil)
	v, err := Mount(volname, s.dir)
	s.drv.AssertExpectations(c)
	c.Assert(v, Equals, s.vol)
	c.Assert(err, IsNil)
}

func (s *DriverSuite) TestMountWhenExists(c *C) {
	volname := "testvolume"
	s.drv.On("Exists", volname).Return(true)
	s.drv.On("Get", volname).Return(s.vol, nil)
	s.drv.On("Root").Return(s.dir)
	err := InitDriver(drvName, s.dir, drvArgs)
	c.Assert(err, IsNil)
	v, err := Mount(volname, s.dir)
	s.drv.AssertExpectations(c)
	s.drv.AssertNotCalled(c, "Create", volname)
	c.Assert(v, Equals, s.vol)
	c.Assert(err, IsNil)
}

func (s *DriverSuite) TestBadMount(c *C) {
	err := InitDriver(unregistered, s.dir, drvArgs)
	c.Assert(err, Equals, ErrDriverNotSupported)
	v, err := Mount("", s.dir)
	c.Assert(err, Equals, ErrDriverNotInit)
	c.Assert(v, IsNil)
}
