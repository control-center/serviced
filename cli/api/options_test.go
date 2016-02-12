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
	"strings"
	"testing"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
)

func TestValidateCommonOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)

	err := ValidateCommonOptions(testOptions)

	if err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateCommonOptionsFailsWithInvalidRPCCertVerify(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.RPCCertVerify = "foobar"

	err := ValidateCommonOptions(testOptions)

	assertErrorContent(t, err, "error parsing rpc-cert-verify value")
}

func TestValidateCommonOptionsFailsWithInvalidRPCDisableTLS(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.RPCDisableTLS = "not a boolean string"

	err := ValidateCommonOptions(testOptions)

	assertErrorContent(t, err, "error parsing rpc-disable-tls value")
}

func TestValidateCommonOptionsFailsWithInvalidUIAddress(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.UIPort = "not a valid port string"

	err := ValidateCommonOptions(testOptions)

	assertErrorContent(t, err, "error validating UI port")
}

func TestValidateCommonOptionsFailsWithInvalidVirtualAddressSubnet(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.VirtualAddressSubnet = "not a valid subnet"

	err := ValidateCommonOptions(testOptions)

	assertErrorContent(t, err, "error validating virtual-address-subnet")
}

func TestValidateServerOptions(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeBtrFS
	LoadOptions(testOptions)

	err := ValidateServerOptions()
	if err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	}
}

func TestValidateServerOptionsFailsIfNotMasterOrAgent(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = false
	testOptions.Agent = false
	LoadOptions(testOptions)

	err := ValidateServerOptions()

	assertErrorContent(t, err, "no mode (master or agent) was specified")
}

func TestValidateServerOptionsFailsIfStorageInvalid(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeDeviceMapper
	testOptions.StorageArgs = []string{}
	LoadOptions(testOptions)

	err := ValidateServerOptions()

	assertErrorContent(t, err, "Use of devicemapper loop back device is not allowed")
}

func TestValidateServerOptionsFailsIfAgentMissingEndpoint(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Agent = true
	testOptions.Endpoint = ""
	LoadOptions(testOptions)

	err := ValidateServerOptions()

	assertErrorContent(t, err, "No endpoint to master has been configured")
}

func TestValidateServerOptionsSetsEndpointIfMasterMissingEndpoint(t *testing.T) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeBtrFS
	testOptions.Endpoint = ""
	LoadOptions(testOptions)

	err := ValidateServerOptions()

	if err != nil {
		t.Errorf("expected pass, but got error: %s", err)
	} else if len(options.Endpoint) == 0 {
		t.Errorf("options.Endpoint pass was not set")
	}
}

func assertErrorContent(t *testing.T, err error, expectedContent string) {
	if err == nil {
		t.Errorf("expected error, but got ni")
	} else if !strings.Contains(err.Error(), expectedContent) {
		t.Errorf("expected error message to contain %q, but got %q", expectedContent, err)
	}

}
