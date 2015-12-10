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

// +build integration,root

package main_test

import (
	"testing"

	. "github.com/control-center/serviced/tools/serviced-storage"
	"github.com/control-center/serviced/volume"
	. "gopkg.in/check.v1"
)

const (
	size int64 = 100 * 1024 * 1024
)

func TestThinpool(t *testing.T) { TestingT(t) }

type ThinpoolSuite struct {
	devices []*volume.TemporaryDevice
}

func (s *ThinpoolSuite) TempDevice(c *C) *volume.TemporaryDevice {
	dev, err := volume.CreateTemporaryDevice(size)
	if err != nil {
		c.Fatal(err)
	}
	s.devices = append(s.devices, dev)
	return dev
}

var _ = Suite(&ThinpoolSuite{})

func (s *ThinpoolSuite) TearDownTest(c *C) {
	for _, dev := range s.devices {
		dev.Destroy()
	}
}

func (s *ThinpoolSuite) TestEnsurePhysicalDevices(c *C) {
	dev1 := s.TempDevice(c)
	dev2 := s.TempDevice(c)

	// First create the pvs
	err := EnsurePhysicalDevices([]string{dev1.LoopDevice(), dev2.LoopDevice()})
	c.Assert(err, IsNil)

	// Should be idempotent
	err = EnsurePhysicalDevices([]string{dev1.LoopDevice(), dev2.LoopDevice()})
	c.Assert(err, IsNil)

	// Invalid devices should fail
	err = EnsurePhysicalDevices([]string{"/not/a/device"})
	c.Assert(err, Not(IsNil))
}

func (s *ThinpoolSuite) TestCreateVolumeGroup(c *C) {
	purpose := "serviced"

	// Should fail if devices are invalid
	err := CreateVolumeGroup(purpose, []string{"/dev/invalid1", "/dev/invalid2"})
	c.Assert(err, Not(IsNil))

	// Create a couple devices
	dev1 := s.TempDevice(c)
	dev2 := s.TempDevice(c)

	devices := []string{dev1.LoopDevice(), dev2.LoopDevice()}

	// Should fail if the devices aren't pvs yet
	err = CreateVolumeGroup(purpose, devices)
	c.Assert(err, Not(IsNil))

	// Ensure pvs
	err = EnsurePhysicalDevices(devices)
	c.Assert(err, IsNil)

	// Now it should succeed
	err = CreateVolumeGroup(purpose, devices)
	c.Assert(err, Not(IsNil))

}

func (s *ThinpoolSuite) TestCreateMetadataVolume(c *C) {

}
