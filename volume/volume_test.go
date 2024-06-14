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

//go:build integration
// +build integration

package volume_test

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/control-center/serviced/utils/iostat"
	iostatmocks "github.com/control-center/serviced/utils/iostat/mocks"
	. "github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/mocks"
	. "gopkg.in/check.v1"
	"time"
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

func (s *DriverSuite) TestInitIOStat_CallTwice(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done1 := make(chan interface{})
	done2 := make(chan interface{})

	statsChan := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChan <-chan map[string]iostat.DeviceUtilizationReport = statsChan
	getter.On("GetIOStatsCh").Return(retChan, nil).Once()

	go func() {
		InitIOStat(getter, quitCh)
		close(done1)
	}()

	// Wait 1 second for InitIOStat to get called
	timer := time.NewTimer(time.Second)
	<-timer.C

	go func() {
		InitIOStat(getter, quitCh)
		close(done2)
	}()

	// Make sure a second call exits immediately
	timer.Reset(time.Second)
	select {
	case <-done2:
		timer.Reset(time.Second)
		// Make sure the first goroutine is still running
		select {
		case <-done1:
			c.Fatalf("First call to InitIOStat exited prematurely")
		case <-timer.C:
		}
	case <-timer.C:
		c.Fatalf("Second call to InitIOStat did not return")
	}

	// Quit and make sure first goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done1:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}
}

func (s *DriverSuite) TestInitIOStat_ErrGet(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done := make(chan interface{})
	testErr := errors.New("Test error")

	getter.On("GetIOStatsCh").Return(nil, testErr)

	getter.On("GetStatInterval").Return(100 * time.Millisecond)

	go func() {
		InitIOStat(getter, quitCh)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	// Make sure the call doesn't exit
	select {
	case <-done:
		c.Fatalf("Call to InitIOStat exited prematurely")
	case <-timer.C:
	}

	// Quit and make sure goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}

	getter.AssertExpectations(c)
}

func (s *DriverSuite) TestInitIOStat_ErrGet_AndResume(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done := make(chan interface{})
	statsChan := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChan <-chan map[string]iostat.DeviceUtilizationReport = statsChan
	testErr := errors.New("Test error")

	getter.On("GetIOStatsCh").Return(nil, testErr).Twice()
	getter.On("GetIOStatsCh").Return(retChan, nil).Once()

	getter.On("GetStatInterval").Return(100 * time.Millisecond)

	go func() {
		InitIOStat(getter, quitCh)
		close(done)
	}()

	timer := time.NewTimer(time.Second)
	// Make sure the call doesn't exit
	select {
	case <-done:
		c.Fatalf("Call to InitIOStat exited prematurely")
	case <-timer.C:
	}

	// Quit and make sure goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}

	getter.AssertExpectations(c)
}

func (s *DriverSuite) TestInitIOStat_ChannelClosed(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done := make(chan interface{})
	statsChan := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChan <-chan map[string]iostat.DeviceUtilizationReport = statsChan

	getter.On("GetIOStatsCh").Return(retChan, nil)
	getter.On("GetStatInterval").Return(100 * time.Millisecond)

	go func() {
		InitIOStat(getter, quitCh)
		close(done)
	}()

	// Close the channel
	close(statsChan)

	// Make sure the call doesn't exit
	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Call to InitIOStat exited prematurely")
	case <-timer.C:
	}

	// Quit and make sure goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}

	getter.AssertExpectations(c)
}

func (s *DriverSuite) TestInitIOStat_ChannelClosed_AndResume(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done := make(chan interface{})
	statsChanClosed := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChanClosed <-chan map[string]iostat.DeviceUtilizationReport = statsChanClosed

	statsChan := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChan <-chan map[string]iostat.DeviceUtilizationReport = statsChan

	getter.On("GetIOStatsCh").Return(retChanClosed, nil).Twice()
	getter.On("GetIOStatsCh").Return(retChan, nil).Once()
	getter.On("GetStatInterval").Return(100 * time.Millisecond)

	go func() {
		InitIOStat(getter, quitCh)
		close(done)
	}()

	// Close the channel
	close(statsChanClosed)

	// Make sure the call doesn't exit
	timer := time.NewTimer(time.Second)
	select {
	case <-done:
		c.Fatalf("Call to InitIOStat exited prematurely")
	case <-timer.C:
	}

	// Quit and make sure goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}

	getter.AssertExpectations(c)
}

func (s *DriverSuite) TestInitIOStat_Success(c *C) {
	getter := &iostatmocks.Getter{}
	quitCh := make(chan interface{})
	done := make(chan interface{})
	statsChan := make(chan map[string]iostat.DeviceUtilizationReport)
	var retChan <-chan map[string]iostat.DeviceUtilizationReport = statsChan

	getter.On("GetIOStatsCh").Return(retChan, nil).Once()

	go func() {
		InitIOStat(getter, quitCh)
		close(done)
	}()

	// Send some data to the channel
	result := make(map[string]iostat.DeviceUtilizationReport)
	result["test"] = iostat.DeviceUtilizationReport{
		RPS: 1.123,
		WPS: 4.567,
	}

	timer := time.NewTimer(time.Second)
	select {
	case statsChan <- result:
	case <-timer.C:
		c.Fatalf("Could not send stats to channel")
	}

	actual, ok := GetLastIOStat()

	c.Assert(ok, Equals, true)
	c.Assert(actual, DeepEquals, result)

	// Make sure the goroutine doesn't exit
	timer.Reset(time.Second)
	select {
	case <-done:
		c.Fatalf("Call to InitIOStat exited prematurely")
	case <-timer.C:
	}

	// Quit and make sure goroutine exits
	close(quitCh)
	timer.Reset(time.Second)
	select {
	case <-done:
	case <-timer.C:
		c.Fatalf("InitIOStat did not exit")
	}

	getter.AssertExpectations(c)
}
