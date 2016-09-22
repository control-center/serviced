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

	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"

	. "gopkg.in/check.v1"
)

func (s *TestAPISuite) TestValidateCommonOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)

	err := ValidateCommonOptions(testOptions)

	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateCommonOptionsFailsWithInvalidRPCCertVerify(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.RPCCertVerify = "foobar"

	err := ValidateCommonOptions(testOptions)

	s.assertErrorContent(c, err, "error parsing rpc-cert-verify value")
}

func (s *TestAPISuite) TestValidateCommonOptionsFailsWithInvalidRPCDisableTLS(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.RPCDisableTLS = "not a boolean string"

	err := ValidateCommonOptions(testOptions)

	s.assertErrorContent(c, err, "error parsing rpc-disable-tls value")
}

func (s *TestAPISuite) TestValidateCommonOptionsFailsWithInvalidUIAddress(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.UIPort = "not a valid port string"

	err := ValidateCommonOptions(testOptions)

	s.assertErrorContent(c, err, "error validating UI port")
}

func (s *TestAPISuite) TestValidateCommonOptionsFailsWithInvalidVirtualAddressSubnet(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.VirtualAddressSubnet = "not a valid subnet"

	err := ValidateCommonOptions(testOptions)

	s.assertErrorContent(c, err, "error validating virtual-address-subnet")
}

func (s *TestAPISuite) TestValidateServerOptions(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeBtrFS
	config.LoadOptions(testOptions)

	err := ValidateServerOptions()

	c.Assert(err, IsNil)
}

func (s *TestAPISuite) TestValidateServerOptionsFailsIfNotMasterOrAgent(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = false
	testOptions.Agent = false
	config.LoadOptions(testOptions)

	err := ValidateServerOptions()

	s.assertErrorContent(c, err, "no mode (master or agent) was specified")
}

func (s *TestAPISuite) TestValidateServerOptionsFailsIfStorageInvalid(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeDeviceMapper
	testOptions.StorageArgs = []string{}
	config.LoadOptions(testOptions)

	err := ValidateServerOptions()

	s.assertErrorContent(c, err, "Use of devicemapper loop back device is not allowed")
}

func (s *TestAPISuite) TestValidateServerOptionsFailsIfAgentMissingEndpoint(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Agent = true
	testOptions.Endpoint = ""
	config.LoadOptions(testOptions)

	err := ValidateServerOptions()

	s.assertErrorContent(c, err, "No endpoint to master has been configured")
}

func (s *TestAPISuite) TestValidateServerOptionsSetsEndpointIfMasterMissingEndpoint(c *C) {
	configReader := utils.TestConfigReader(map[string]string{})
	testOptions := GetDefaultOptions(configReader)
	testOptions.Master = true
	testOptions.FSType = volume.DriverTypeBtrFS
	testOptions.Endpoint = ""
	config.LoadOptions(testOptions)

	err := ValidateServerOptions()

	c.Assert(err, IsNil)
	c.Assert(len(options.Endpoint), Not(Equals), 0)
}

func (s *TestAPISuite) assertErrorContent(c *C, err error, expectedContent string) {
	c.Assert(err, Not(IsNil))
	if !strings.Contains(err.Error(), expectedContent) {
		c.Errorf("expected error message to contain %q, but got %q", expectedContent, err)
	}
}
