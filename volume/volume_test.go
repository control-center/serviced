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

package volume_test

import (
	"testing"

	. "github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
)

func TestDriver(t *testing.T) { TestingT(t) }

type DriverSuite struct {
	drv *mocks.Driver
	vol *mocks.Volume
}

var (
	_       = Suite(&DriverSuite{})
	drvName = "mock"
)

func (s *DriverSuite) MockInit(ignored string) (Driver, error) {
	return s.drv, nil
}

func (s *DriverSuite) SetUpTest(c *C) {
	s.drv = &mocks.Driver{}
	s.vol = &mocks.Volume{}
	Register(drvName, s.MockInit)
}

func (s *DriverSuite) TearDownTest(c *C) {
	Unregister(drvName)
}

func (s *DriverSuite) TestNilRegistration(c *C) {
	err := Register("nilregistration", nil)
	c.Assert(err, ErrorMatches, "Can't register a nil driver initializer")
}

func (s *DriverSuite) TestRedundantRegistration(c *C) {
	err := Register(drvName, s.MockInit)
	c.Assert(err, ErrorMatches, "Already registered driver mock")
}

func (s *DriverSuite) TestRegistration(c *C) {
	driver, err := GetDriver(drvName, "")
	c.Assert(err, IsNil)
	c.Assert(driver, FitsTypeOf, s.drv)
}

func (s *DriverSuite) TestUnsupported(c *C) {
	driver, err := GetDriver("unregistered", "")
	c.Assert(err, ErrorMatches, ErrDriverNotSupported.Error())
	c.Assert(driver, IsNil)
}

func (s *DriverSuite) TestMountWhenDoesNotExist(c *C) {
	volname := "testvolume"
	s.drv.On("Exists", volname).Return(false)
	s.drv.On("Create", volname).Return(s.vol, nil)
	v, err := Mount(drvName, volname, "")
	s.drv.AssertExpectations(c)
	c.Assert(v, Equals, s.vol)
	c.Assert(err, IsNil)
}

func (s *DriverSuite) TestMountWhenExists(c *C) {
	volname := "testvolume"
	s.drv.On("Exists", volname).Return(true)
	s.drv.On("Get", volname).Return(s.vol, nil)
	v, err := Mount(drvName, volname, "")
	s.drv.AssertExpectations(c)
	s.drv.AssertNotCalled(c, "Create", volname)
	c.Assert(v, Equals, s.vol)
	c.Assert(err, IsNil)
}

func (s *DriverSuite) TestBadMount(c *C) {
	v, err := Mount("unregistered", "", "")
	c.Assert(err, ErrorMatches, ErrDriverNotSupported.Error())
	c.Assert(v, IsNil)
}
