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
	"flag"
	"testing"
	apimocks "github.com/control-center/serviced/cli/api/mocks"
	cgcli "github.com/codegangsta/cli"
	. "gopkg.in/check.v1"
	//"github.com/docker/docker/cli"
	//"github.com/stretchr/testify/assert"
	"github.com/control-center/serviced/utils"
	"fmt"
	"os"
)

var tGlobal *testing.T

// Hook up gocheck into the "go test" runner.
func TestLogs(t *testing.T) {
	TestingT(t)

	// HACK ALERT: Some functions in testify.Mock require testing.T, but
	// 	gocheck doesn't offer access to it, so we have to save a copy in
	// 	a global variable :=(
	tGlobal = t
}

type TestLogsSuite struct {
	mockAPI *apimocks.API	// A mock implementation of the API interface
}

var _ = Suite(&TestLogsSuite{})


func (s *TestLogsSuite) SetUpTest(c *C) {
	fmt.Fprintf(os.Stderr, " STARTED %s\n", c.TestName())
	s.mockAPI = &apimocks.API{}
}

func (s *TestLogsSuite) TearDownTest(c *C) {
	fmt.Fprintf(os.Stderr, " FINISHED %s\n", c.TestName())
	// don't allow per-test-case values to be reused across test cases
	s.mockAPI = nil
}

func (s *TestLogsSuite) TestLogsSomething(c *C) {
	//call cmdExportLogs, verify that searchForService() is called
	cli := New(s.mockAPI, utils.TestConfigReader(make(map[string]string)))

	app := cgcli.NewApp()
	set := flag.NewFlagSet("test", 0)
	test := []string{"blah", "blah", "-break"}
	set.Parse(test)

	ctx := cgcli.NewContext(app, set, set)
	cli.cmdExportLogs(ctx)
}


//func TestConfigureContainer_DockerLog(t *testing.T) {
//assert := assert.New(t)
//
//// Create a fake pull registry that doesn't pull images
//fakeRegistry := &regmocks.Registry{}
//fakeRegistry.On("SetConnection", mock.Anything).Return(nil)
//fakeRegistry.On("ImagePath", mock.Anything).Return("someimage", nil)
//fakeRegistry.On("PullImage", mock.Anything).Return(nil)
//
//// Create a fake HostAgent
//fakeHostAgent := &HostAgent{
//uiport:               ":443",
//dockerLogDriver:      "fakejson-log",
//dockerLogConfig:      map[string]string{"alpha": "one", "bravo": "two", "charlie": "three"},
//virtualAddressSubnet: "0.0.0.0",
//pullreg:              fakeRegistry,
//}
//
//// Create a fake client that won't make any RPC calls
//fakeClient := &mocks.ControlPlane{}
//
//// Create a fake service.Service
//fakeService := &service.Service{
//ImageID: "busybox:latest",
//}
//
//// Create a fake servicestate.ServiceState
//fakeServiceState := &servicestate.ServiceState{}
//
//fakeClient.On("GetTenantId", mock.Anything, mock.Anything).Return(nil)
//fakeClient.On("GetSystemUser", mock.Anything, mock.Anything).Return(nil)
//
//// Call configureContainer
//config, hostconfig, err := configureContainer(
//fakeHostAgent,
//fakeClient,
//fakeService,
//fakeServiceState,
//fakeHostAgent.virtualAddressSubnet)
//
//assert.NotNil(config)
//assert.NotNil(hostconfig)
//assert.Nil(err)
//
//// Test that hostconfig values are as intended
//assert.Equal(hostconfig.LogConfig.Type, "fakejson-log")
//assert.Equal(hostconfig.LogConfig.Config["alpha"], "one")
//assert.Equal(hostconfig.LogConfig.Config["bravo"], "two")
//assert.Equal(hostconfig.LogConfig.Config["charlie"], "three")
//}
