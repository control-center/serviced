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
	"strconv"
	"strings"

	"github.com/Sirupsen/logrus"
	mockapi "github.com/control-center/serviced/cli/api/apimocks"
	"github.com/control-center/serviced/domain/host"
	mocklog "github.com/control-center/serviced/logging/mocks"
	"github.com/control-center/serviced/utils"

	. "gopkg.in/check.v1"
)

type CmdLoggingSuite struct {
	api mockapi.API
	log *mocklog.LogControl
}

var _ = Suite(&CmdLoggingSuite{})

func (s *CmdLoggingSuite) SetUpTest(c *C) {
	s.api = mockapi.API{}
	s.log = &mocklog.LogControl{}
}

func (s *CmdLoggingSuite) Run(env map[string]string, args ...string) {
	reader := utils.TestConfigReader(env)
	New(&s.api, reader, s.log).Run(args)
}

func (s *CmdLoggingSuite) Test_LoadsCliDefaultConfigFile(c *C) {
	env := map[string]string{}
	expected := "/opt/serviced/etc/logconfig-cli.yaml"
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)
	s.Run(env, "serviced", "host", "list")
}

func (s *CmdLoggingSuite) Test_LoadsServerDefaultConfigFile(c *C) {
	env := map[string]string{}
	expected := "/opt/serviced/etc/logconfig-server.yaml"
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("StartServer").Return(nil)
	s.Run(env, "serviced", "--fstype=rsync", "--master", "server")
}

func (s *CmdLoggingSuite) Test_LoadsDefaultConfigFileFromHome(c *C) {
	home := "/some/path/to"
	env := map[string]string{"HOME": home}
	expected := home + "/etc/logconfig-cli.yaml"
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)
	s.Run(env, "serviced", "host", "list")
}

func (s *CmdLoggingSuite) Test_LoadsConfigFile_Error(c *C) {
	// In the event of an error loading the config file, the error is emitted,
	// but the command runs and the file is watched.
	env := map[string]string{}
	expected := "/opt/serviced/etc/logconfig-cli.yaml"
	err := "foobarbazqux"
	s.log.On("ApplyConfigFromFile", expected).Return(errors.New(err))
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)
	f := func(args ...string) { s.Run(env, args...) }
	g := func(args ...string) { pipeStderr(f, args...) }
	output := string(pipe(g, "serviced", "host", "list"))
	output = strings.Split(output, "\n")[0]
	s.log.AssertExpectations(c)
	s.api.AssertExpectations(c)
	c.Assert(output, Matches, ".*"+err)
}

func (s *CmdLoggingSuite) Test_LoadsSpecifiedConfigFile(c *C) {
	expected := "/some/path/to/logconfig.yaml"
	env := map[string]string{"LOG_CONFIG": expected}
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)
	s.Run(env, "serviced", "host", "list")
}

func (s *CmdLoggingSuite) Test_VerbosityFlag(c *C) {
	env := map[string]string{}
	expected := "/opt/serviced/etc/logconfig-cli.yaml"
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)
	tests := map[int]logrus.Level{
		0:  logrus.DebugLevel,
		1:  logrus.InfoLevel,
		2:  logrus.WarnLevel,
		3:  logrus.ErrorLevel,
		4:  logrus.ErrorLevel,
		-1: logrus.ErrorLevel,
	}
	for verbosity, level := range tests {
		s.log.On("SetVerbosity", verbosity).Once()
		s.log.On("SetLevel", level).Once()
		s.Run(env, "serviced", "-v", strconv.Itoa(verbosity), "host", "list")
	}
}

func (s *CmdLoggingSuite) Test_GlogOptions(c *C) {
	env := map[string]string{}
	expected := "/opt/serviced/etc/logconfig-cli.yaml"
	s.log.On("ApplyConfigFromFile", expected).Return(nil)
	s.log.On("WatchConfigFile", expected).Return(nil)
	s.api.On("GetHosts").Return([]host.Host{}, nil)

	s.log.On("SetAlsoToStderr", true).Return().Once()
	s.Run(env, "serviced", "--alsologtostderr", "host", "list")

	s.log.On("SetStderrThreshold", "foo").Return(nil).Once()
	s.Run(env, "serviced", "--stderrthreshold", "foo", "host", "list")

	s.log.On("SetTraceLocation", "bar").Return(nil).Once()
	s.Run(env, "serviced", "--log_backtrace_at", "bar", "host", "list")

	s.log.On("SetVModule", "baz,qux").Return(nil).Once()
	s.Run(env, "serviced", "--vmodule", "baz,qux", "host", "list")

}
