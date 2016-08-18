// Copyright 2016 The Serviced Authors.
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

package api

import (
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	. "gopkg.in/check.v1"
)

func (s *TestAPISuite) TestAddStorageOptionWithEmptyDefault(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	var options []string

	addStorageOption(configReader, "EMPTY_DEFAULT", "", func(value string) {
		options = append(options, value)
	})

	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestAddStorageOptionWithNonEmptyDefault(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	var options []string

	addStorageOption(configReader, "USE_DEFAULT", "default", func(value string) {
		options = append(options, value)
	})

	verifyOptions(c, options, []string{"default"})
}

func (s *TestAPISuite) TestAddStorageOptionWithEnvValue(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"IGNORE_DEFAULT": "ignore"})
	var options []string

	addStorageOption(configReader, "IGNORE_DEFAULT", "default", func(value string) {
		options = append(options, value)
	})

	verifyOptions(c, options, []string{"ignore"})
}

func (s *TestAPISuite) TestGetDefaultNFSOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeNFS, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultRSyncOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeRsync, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultBtrfsOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeBtrFS, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultDevicemapperOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(c, options, []string{"dm.basesize=100G"})
}

func (s *TestAPISuite) TestGetDefaultNFSOptionsWithDMOptionsSet(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeNFS, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultRSyncOptionsWithDMOptionsSet(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeRsync, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultBrtrfsOptionsWithDMOptionsSet(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeBtrFS, configReader)
	verifyOptions(c, options, []string{})
}

func (s *TestAPISuite) TestGetDefaultDevicemapperOptionsForThinpoolDevice(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(c, options, []string{"dm.thinpooldev=foo", "dm.basesize=100G"})
}

func (s *TestAPISuite) TestGetDefaultDevicemapperOptionsForAll(c *C) {
	configReader := utils.TestConfigReader(map[string]string{
		"DM_THINPOOLDEV":      "foo",
		"DM_BASESIZE":         "200G",
		"DM_LOOPDATASIZE":     "10G",
		"DM_LOOPMETADATASIZE": "1G",
		"DM_ARGS":             "arg1=a,arg2=b,arg3=c",
	})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(c, options, []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
		"arg1=a,arg2=b,arg3=c",
	})
}

func (s *TestAPISuite) TestThinPoolEnabledWithNoOptions(c *C) {
	options := []string{}

	c.Assert(thinPoolEnabled(options), Equals, false)
}

func (s *TestAPISuite) TestThinPoolEnabledWithoutThinPool(c *C) {
	options := []string{
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	c.Assert(thinPoolEnabled(options), Equals, false)
}

func (s *TestAPISuite) TestThinPoolEnabledWithThinPool(c *C) {
	options := []string{
		"dm.thinpooldev=foo",
	}

	c.Assert(thinPoolEnabled(options), Equals, true)
}

func (s *TestAPISuite) TestThinPoolEnabledWithBothKindsOfOptions(c *C) {
	options := []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	c.Assert(thinPoolEnabled(options), Equals, true)
}

func (s *TestAPISuite) TestLoopBackOptionsFoundWithNoOptions(c *C) {
	options := []string{}

	c.Assert(loopBackOptionsFound(options), Equals, false)
}

func (s *TestAPISuite) TestLoopBackOptionsFoundWithoutLoopBackOptions(c *C) {
	options := []string{
		"dm.thinpooldev=foo",
	}

	c.Assert(loopBackOptionsFound(options), Equals, false)
}

func (s *TestAPISuite) TestLoopBackOptionsFoundWithLoopBackOptions(c *C) {
	options := []string{
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	c.Assert(loopBackOptionsFound(options), Equals, true)
}

func (s *TestAPISuite) TestLoopBackOptionsFoundWithJustBaseSize(c *C) {
	options := []string{
		"dm.basesize=200G",
	}
	c.Assert(loopBackOptionsFound(options), Equals, false)
}

func (s *TestAPISuite) TestLoopBackOptionsFoundWithBothKindsOfOptions(c *C) {
	options := []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	c.Assert(loopBackOptionsFound(options), Equals, true)
}

func (s *TestAPISuite) TestValidateStorageArgsPassBtrfs(c *C) {
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeBtrFS,
		StorageArgs:   []string{},
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	err := validateStorageArgs()

	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateStorageArgsPassRsync(c *C) {
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeRsync,
		StorageArgs:   []string{},
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	err := validateStorageArgs()
	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateStorageArgsPassDMWithThinpool(c *C) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	storageArgs := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeDeviceMapper,
		StorageArgs:   storageArgs,
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	err := validateStorageArgs()

	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateStorageArgsFailDMWithLoopBack(c *C) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.AllowLoopBack = "false"
	LoadOptions(testOptions)

	err := validateStorageArgs()

	s.assertErrorContent(c, err, "devicemapper loop back device is not allowed")
}

func (s *TestAPISuite) TestValidateStorageArgsPassDMWithAllowLoopBack(c *C) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.AllowLoopBack = "true"
	LoadOptions(testOptions)

	err := validateStorageArgs()

	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateStorageArgsDoesNotFailIfAgentOnly(c *C) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.Master = false
	testOptions.Agent = true
	testOptions.AllowLoopBack = "false"
	LoadOptions(testOptions)

	err := validateStorageArgs()

	c.Assert(err, IsNil)
}

func verifyOptions(c *C, actual []string, expected []string) {
	c.Assert(len(actual), Equals, len(expected))
	c.Assert(actual, DeepEquals, expected)
}

func setupOptionsForDMWithLoopBack() Options {
	// Technically speaking, we w/loopback is indicated by no DM_THINPOOLDEV, still add a loop-back option
	// to emphasize the point
	configReader := utils.TestConfigReader(map[string]string{"DM_LOOPDATASIZE": "1G"})
	storageArgs := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	testOptions := Options{
		Master:      true,
		FSType:      volume.DriverTypeDeviceMapper,
		StorageArgs: storageArgs,
	}

	return testOptions
}
