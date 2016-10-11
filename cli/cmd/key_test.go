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

package cmd

import (
	"errors"
	"strings"
	"testing"

	"github.com/control-center/serviced/cli/api/apimocks"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type mySuite struct {
	api mocks.API
	cli *ServicedCli
}

var testHost = host.Host{
	ID:             "test-host-id-2",
	PoolID:         "default",
	Name:           "beta",
	IPAddr:         "192.168.0.1",
	Cores:          2,
	Memory:         512 * 1024 * 1024,
	PrivateNetwork: "10.0.0.1/66",
}
var testHostFilename = "IP-192-168-0-1.delegate.key"
var testKeyData = []byte("Fake Key Data")

var _ = Suite(&mySuite{})

func (s *mySuite) SetUpTest(c *C) {
	s.api = mocks.API{}
	s.cli = New(&s.api, utils.TestConfigReader(make(map[string]string)))
}

func (s *mySuite) Test_cmdKeyReset(c *C) {
	s.api.On("GetHost", testHost.ID).Return(&testHost, nil)
	s.api.On("ResetHostKey", testHost.ID).Return(testKeyData, nil)
	s.api.On("WriteDelegateKey", testHostFilename, testKeyData).Return(nil)
	s.cli.Run(strings.Split("serviced key reset "+testHost.ID, " "))
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_cmdKeyReset_hostNotFound(c *C) {
	s.api.On("GetHost", testHost.ID).Return(nil, errors.New("?"))
	s.cli.Run(strings.Split("serviced key reset "+testHost.ID, " "))
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_cmdKeyReset_resetError(c *C) {
	s.api.On("GetHost", testHost.ID).Return(&testHost, nil)
	s.api.On("ResetHostKey", testHost.ID).Return(nil, errors.New("?"))
	s.cli.Run(strings.Split("serviced key reset "+testHost.ID, " "))
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey(c *C) {
	s.api.On("WriteDelegateKey", testHostFilename, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, "", false)
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey_keyfile(c *C) {
	keyfileName := "foo.bar"
	s.api.On("WriteDelegateKey", keyfileName, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, keyfileName, false)
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey_register(c *C) {
	s.api.On("RegisterRemoteHost", &testHost, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, "", true)
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey_registerfail(c *C) {
	s.api.On("RegisterRemoteHost", &testHost, testKeyData).Return(errors.New("woot"))
	s.api.On("WriteDelegateKey", testHostFilename, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, "", true)
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey_register_keyfile(c *C) {
	keyfileName := "foo-bar"
	s.api.On("RegisterRemoteHost", &testHost, testKeyData).Return(nil)
	s.api.On("WriteDelegateKey", keyfileName, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, keyfileName, true)
	s.api.AssertExpectations(c)
}

func (s *mySuite) Test_outputDelegateKey_registerfail_keyfile(c *C) {
	keyfileName := "foo-bar"
	s.api.On("RegisterRemoteHost", &testHost, testKeyData).Return(errors.New("woot"))
	s.api.On("WriteDelegateKey", keyfileName, testKeyData).Return(nil)
	s.cli.outputDelegateKey(&testHost, testKeyData, keyfileName, true)
	s.api.AssertExpectations(c)
}
