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
	"os/exec"
	"regexp"
	"strings"
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
	// Create a couple devices
	devices := []string{
		s.TempDevice(c).LoopDevice(),
		s.TempDevice(c).LoopDevice(),
	}

	// First create the pvs
	err := EnsurePhysicalDevices(devices)
	c.Assert(err, IsNil)

	defer exec.Command("pvremove", devices...).Run()

	// Should be idempotent
	err = EnsurePhysicalDevices(devices)
	c.Assert(err, IsNil)

	// Invalid devices should fail
	err = EnsurePhysicalDevices([]string{"/not/a/device"})
	c.Assert(err, Not(IsNil))
}

func (s *ThinpoolSuite) TestCreateVolumeGroup(c *C) {
	volumeGroup := "serviced-test-1"

	// Should fail if devices are invalid
	err := CreateVolumeGroup(volumeGroup, []string{"/dev/invalid1", "/dev/invalid2"})
	c.Assert(err, Not(IsNil))

	// Create a couple devices
	devices := []string{
		s.TempDevice(c).LoopDevice(),
		s.TempDevice(c).LoopDevice(),
	}

	// Ensure pvs
	err = EnsurePhysicalDevices(devices)
	c.Assert(err, IsNil)
	defer exec.Command("pvremove", devices...).Run()

	// Should succeed now
	err = CreateVolumeGroup(volumeGroup, devices)
	c.Assert(err, IsNil)
	defer exec.Command("vgremove", volumeGroup).Run()

}

func (s *ThinpoolSuite) TestCreateMetadataVolume(c *C) {
	volumeGroup := "serviced-test-2"

	// Should fail if the volume group is invalid
	_, err := CreateMetadataVolume(volumeGroup)
	c.Assert(err, Not(IsNil))

	// Create some devices and a vg
	devices := []string{
		s.TempDevice(c).LoopDevice(),
		s.TempDevice(c).LoopDevice(),
	}
	if err := EnsurePhysicalDevices(devices); err != nil {
		c.Fatal(err)
	}
	defer exec.Command("pvremove", devices...).Run()
	if err := CreateVolumeGroup(volumeGroup, devices); err != nil {
		c.Fatal(err)
	}
	defer exec.Command("vgremove", "-f", volumeGroup).Run()

	// Now the metadata volume should work fine
	mdvol, err := CreateMetadataVolume(volumeGroup)
	c.Assert(err, IsNil)
	defer exec.Command("lvremove", mdvol).Run()

	out, _ := exec.Command("lvs").CombinedOutput()
	c.Assert(strings.Contains(string(out), mdvol), Equals, true)
}

func (s *ThinpoolSuite) TestCreateDataVolume(c *C) {
	volumeGroup := "serviced-test-5"

	// Should fail if the volume group is invalid
	_, err := CreateDataVolume(volumeGroup)
	c.Assert(err, Not(IsNil))

	// Create some devices and a vg
	devices := []string{
		s.TempDevice(c).LoopDevice(),
		s.TempDevice(c).LoopDevice(),
	}
	if err := EnsurePhysicalDevices(devices); err != nil {
		c.Fatal(err)
	}
	if err := CreateVolumeGroup(volumeGroup, devices); err != nil {
		c.Fatal(err)
	}
	defer exec.Command("vgremove", "-f", volumeGroup).Run()

	// Now the data volume should work fine
	mdvol, err := CreateDataVolume(volumeGroup)
	c.Assert(err, IsNil)

	out, _ := exec.Command("lvs").CombinedOutput()
	c.Assert(strings.Contains(string(out), mdvol), Equals, true)
}

func (s *ThinpoolSuite) TestConvertToThinpool(c *C) {
	// Should fail if the VG/LV's are invalid
	err := ConvertToThinPool("NotAVolumeGroup", "NotALogicalVolume", "NotALogicalVolume")
	c.Assert(err, Not(IsNil))

	// Set up VG and LV's for test
	dev1 := s.TempDevice(c)
	dev2 := s.TempDevice(c)
	devices := []string{dev1.LoopDevice(), dev2.LoopDevice()}
	EnsurePhysicalDevices(devices)
	volumeGroup := "serviced-test-6"
	CreateVolumeGroup(volumeGroup, devices)
	defer exec.Command("vgremove", "-f", volumeGroup).Run()
	metadataVolume, _ := CreateMetadataVolume(volumeGroup)
	dataVolume, _ := CreateDataVolume(volumeGroup)

	err = ConvertToThinPool(volumeGroup, dataVolume, metadataVolume)
	c.Assert(err, IsNil)
	out, _ := exec.Command("lvs", "-o", "lv_name,lv_attr", "--noheading", "--nameprefixes",
		volumeGroup).CombinedOutput()

	regexName := regexp.MustCompile("LVM2_LV_NAME='(.+?)'")
	regexAttr := regexp.MustCompile("LVM2_LV_ATTR='(.+?)'")
	for _, line := range strings.Split(string(out), "n") {
		match := regexName.FindStringSubmatch(line)
		if len(match) != 2 || match[1] != dataVolume {
			continue
		}

		match = regexAttr.FindStringSubmatch(line)
		c.Assert(len(match), Equals, 2)
		attrs := match[1]
		volumeType := attrs[0]
		// Per lvs man page, volume type of "t" implies thin pool
		c.Assert(volumeType, Equals, "t"[0])
		return
	}
	// Did not find thinpool
	c.Assert(nil, Not(IsNil))
}

func (s *ThinpoolSuite) TestGetInfoForLogicalVolume(c *C) {
	// Set up thin pool for test
	dev1 := s.TempDevice(c)
	dev2 := s.TempDevice(c)
	devices := []string{dev1.LoopDevice(), dev2.LoopDevice()}
	EnsurePhysicalDevices(devices)
	volumeGroup := "serviced-test-3"
	CreateVolumeGroup(volumeGroup, devices)
	defer exec.Command("vgremove", "-f", volumeGroup).Run()
	metadataVolume, _ := CreateMetadataVolume(volumeGroup)
	dataVolume, _ := CreateDataVolume(volumeGroup)
	ConvertToThinPool(volumeGroup, dataVolume, metadataVolume)

	// Should fail if invalid logical volume
	_, err := GetInfoForLogicalVolume("/not/a/logical/volume")
	c.Assert(err, Not(IsNil))

	// Should work for valid params
	lvInfo, err := GetInfoForLogicalVolume(dataVolume)
	c.Assert(err, IsNil)
	c.Assert(lvInfo.Name, Equals, dataVolume)
	c.Assert(lvInfo.KernelMajor, Not(Equals), 0)
	c.Assert(lvInfo.KernelMinor, Not(Equals), 0)
}

func (s *ThinpoolSuite) TestGetThinpoolName(c *C) {
	// Should fail with invalid logical volume
	lvInfo := LogicalVolumeInfo{}
	_, err := lvInfo.GetThinpoolName()
	c.Assert(err, Not(IsNil))

	// Set up thin pool for test
	dev1 := s.TempDevice(c)
	dev2 := s.TempDevice(c)
	devices := []string{dev1.LoopDevice(), dev2.LoopDevice()}
	EnsurePhysicalDevices(devices)
	volumeGroup := "serviced-test-4"
	CreateVolumeGroup(volumeGroup, devices)
	defer exec.Command("vgremove", "-f", volumeGroup).Run()
	metadataVolume, _ := CreateMetadataVolume(volumeGroup)
	dataVolume, _ := CreateDataVolume(volumeGroup)
	ConvertToThinPool(volumeGroup, dataVolume, metadataVolume)
	lvInfo, err = GetInfoForLogicalVolume(dataVolume)

	// Should work for valid params
	thinpoolName, err := lvInfo.GetThinpoolName()
	c.Assert(err, IsNil)
	c.Assert(thinpoolName, Not(Equals), "")

	//TODO: How can we test that the thinpoolName actually refers to the proper device?
}
