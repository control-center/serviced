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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package api

import (
	"testing"

	"github.com/control-center/serviced/utils"
	"reflect"
	"github.com/control-center/serviced/volume"
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
		"DM_THINPOOLDEV": "foo",
		"DM_BASESIZE": "200G",
		"DM_LOOPDATASIZE": "10G",
		"DM_LOOPMETADATASIZE": "1G",
		"DM_ARGS": "arg1=a,arg2=b,arg3=c",
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

func verifyOptions(t *testing.T, actual []string, expected []string) {
	if len(actual) !=len(expected) {
		t.Errorf("length of options incorrect: expected %d got %d; options=%v", len(expected), len(actual), actual)
	} else if len(expected) > 0 && !reflect.DeepEqual(expected, actual) {
		t.Errorf("options incorrect: expected %v got %v", expected, actual)
	}

}
