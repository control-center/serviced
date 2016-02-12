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
	"testing"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"reflect"
	"strings"
)

func TestAddStorageOptionWithEmptyDefault(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	var options []string

	addStorageOption(configReader, "EMPTY_DEFAULT", "", func(value string) {
		options = append(options, value)
	})

	verifyOptions(t, options, []string{})
}

func TestAddStorageOptionWithNonEmptyDefault(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	var options []string

	addStorageOption(configReader, "USE_DEFAULT", "default", func(value string) {
		options = append(options, value)
	})

	verifyOptions(t, options, []string{"default"})
}

func TestAddStorageOptionWithEnvValue(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"IGNORE_DEFAULT": "ignore"})
	var options []string

	addStorageOption(configReader, "IGNORE_DEFAULT", "default", func(value string) {
		options = append(options, value)
	})

	verifyOptions(t, options, []string{"ignore"})
}

func TestGetDefaultNFSOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeNFS, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultRSyncOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeRsync, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultBtrfsOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeBtrFS, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultDevicemapperOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(t, options, []string{"dm.basesize=100G"})
}

func TestGetDefaultNFSOptionsWithDMOptionsSet(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeNFS, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultRSyncOptionsWithDMOptionsSet(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeRsync, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultBrtrfsOptionsWithDMOptionsSet(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeBtrFS, configReader)
	verifyOptions(t, options, []string{})
}

func TestGetDefaultDevicemapperOptionsForThinpoolDevice(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(t, options, []string{"dm.thinpooldev=foo", "dm.basesize=100G"})
}

func TestGetDefaultDevicemapperOptionsForAll(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{
		"DM_THINPOOLDEV":      "foo",
		"DM_BASESIZE":         "200G",
		"DM_LOOPDATASIZE":     "10G",
		"DM_LOOPMETADATASIZE": "1G",
		"DM_ARGS":             "arg1=a,arg2=b,arg3=c",
	})
	options := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	verifyOptions(t, options, []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
		"arg1=a,arg2=b,arg3=c",
	})
}

func TestThinPoolEnabledWithNoOptions(t *testing.T) {
	options := []string{}

	if thinPoolEnabled(options) {
		t.Errorf("expectd false but got true")
	}
}

func TestThinPoolEnabledWithoutThinPool(t *testing.T) {
	options := []string{
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	if thinPoolEnabled(options) {
		t.Errorf("expectd false but got true")
	}
}

func TestThinPoolEnabledWithThinPool(t *testing.T) {
	options := []string{
		"dm.thinpooldev=foo",
	}

	if !thinPoolEnabled(options) {
		t.Errorf("expectd true but got false")
	}
}

func TestThinPoolEnabledWithBothKindsOfOptions(t *testing.T) {
	options := []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	if !thinPoolEnabled(options) {
		t.Errorf("expectd true but got false")
	}
}

func TestLoopBackOptionsFoundWithNoOptions(t *testing.T) {
	options := []string{}

	if loopBackOptionsFound(options) {
		t.Errorf("expectd false but got true")
	}
}

func TestLoopBackOptionsFoundWithoutLoopBackOptions(t *testing.T) {
	options := []string{
		"dm.thinpooldev=foo",
	}

	if loopBackOptionsFound(options) {
		t.Errorf("expectd false but got true")
	}
}

func TestLoopBackOptionsFoundWithLoopBackOptions(t *testing.T) {
	options := []string{
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	if !loopBackOptionsFound(options) {
		t.Errorf("expectd true but got false")
	}
}

func TestLoopBackOptionsFoundWithJustBaseSize(t *testing.T) {
	options := []string{
		"dm.basesize=200G",
	}

	if !loopBackOptionsFound(options) {
		t.Errorf("expectd true but got false")
	}
}

func TestLoopBackOptionsFoundWithBothKindsOfOptions(t *testing.T) {
	options := []string{
		"dm.thinpooldev=foo",
		"dm.basesize=200G",
		"dm.loopdatasize=10G",
		"dm.loopmetadatasize=1G",
	}

	if !loopBackOptionsFound(options) {
		t.Errorf("expectd true but got false")
	}
}

func TestValidateStorageArgsPassBtrfs(t *testing.T) {
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeBtrFS,
		StorageArgs:   []string{},
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	if err := validateStorageArgs(); err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateStorageArgsPassRsync(t *testing.T) {
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeRsync,
		StorageArgs:   []string{},
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	if err := validateStorageArgs(); err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateStorageArgsPassDMWithThinpool(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{"DM_THINPOOLDEV": "foo"})
	storageArgs := getDefaultStorageOptions(volume.DriverTypeDeviceMapper, configReader)
	testOptions := Options{
		Master:        true,
		FSType:        volume.DriverTypeDeviceMapper,
		StorageArgs:   storageArgs,
		AllowLoopBack: "false",
	}
	LoadOptions(testOptions)

	if err := validateStorageArgs(); err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateStorageArgsFailDMWithLoopBack(t *testing.T) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.AllowLoopBack = "false"
	LoadOptions(testOptions)

	expectedError := "devicemapper loop back device is not allowed"
	if err := validateStorageArgs(); err == nil {
		t.Errorf("expected error, but got ni")
	} else if !strings.Contains(err.Error(), expectedError) {
		t.Errorf("expected error message to contain %q, but got %q", expectedError, err)
	}
}

func TestValidateStorageArgsPassDMWithAllowLoopBack(t *testing.T) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.AllowLoopBack = "true"
	LoadOptions(testOptions)

	if err := validateStorageArgs(); err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateStorageArgsDoesNotFailIfAgentOnly(t *testing.T) {
	testOptions := setupOptionsForDMWithLoopBack()
	testOptions.Master = false
	testOptions.Agent = true
	testOptions.AllowLoopBack = "false"
	LoadOptions(testOptions)

	if err := validateStorageArgs(); err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func verifyOptions(t *testing.T, actual []string, expected []string) {
	if len(actual) != len(expected) {
		t.Errorf("length of options incorrect: expected %d got %d; options=%v", len(expected), len(actual), actual)
	} else if len(expected) > 0 && !reflect.DeepEqual(expected, actual) {
		t.Errorf("options incorrect: expected %v got %v", expected, actual)
	}

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
