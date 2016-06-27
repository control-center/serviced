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
	"fmt"
	//"os"
	"testing"

	//api "github.com/control-center/serviced/cli/api"
	mockapi "github.com/control-center/serviced/cli/api/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/utils"
	//. "gopkg.in/check.v1"
	"github.com/codegangsta/cli"
	"flag"
	//"io/ioutil"
	"github.com/control-center/serviced/cli/api"
	"github.com/stretchr/testify/mock"
	//"github.com/davecgh/go-spew/spew"
	//"runtime/debug"
)

//var tGlobal *testing.T

//// Hook up gocheck into the "go test" runner.
//func TestLogs(t *testing.T) {
//	TestingT(t)
//
//	// HACK ALERT: Some functions in testify.Mock require testing.T, but
//	// 	gocheck doesn't offer access to it, so we have to save a copy in
//	// 	a global variable :=(
//	tGlobal = t
//}

type LogsAPITest struct {
	api mockapi.API
	fail  bool
	//pools []pool.ResourcePool
	//hosts []host.Host
	services        []service.Service
	runningServices []dao.RunningService
	errs            map[string]error
}

var DefaultLogsAPITest = LogsAPITest {
	api: 		 mockapi.API{},
	services:        DefaultTestServices,
	runningServices: DefaultTestRunningServices,
	errs:            make(map[string]error, 10),
}

func InitLogsAPITest(driver api.API, args ...string) {
	c := New(driver, utils.TestConfigReader(make(map[string]string)))
	/*
	// parse flags
	set := flagSet(c.App.Name, c.App.Flags)
	set.SetOutput(ioutil.Discard)
	err := set.Parse(arguments[1:])
	context := NewContext(c.App, set, set)
	*/
	c.exitDisabled = true
	c.Run(args)
}

func flagSet(name string, flags []cli.Flag) *flag.FlagSet {
	set := flag.NewFlagSet(name, flag.ContinueOnError)

	for _, f := range flags {
		f.Apply(set)
	}
	return set
}

func (t LogsAPITest) GetServices() ([]service.Service, error) {
	if t.errs["GetServices"] != nil {
		return nil, t.errs["GetServices"]
	}
	return t.services, nil
}

func (t LogsAPITest) GetRunningServices() ([]dao.RunningService, error) {
	if t.errs["GetRunningServices"] != nil {
		return nil, t.errs["GetRunningServices"]
	}
	return t.runningServices, nil
}

func (t LogsAPITest) GetService(id string) (*service.Service, error) {
	if t.errs["GetService"] != nil {
		return nil, t.errs["GetService"]
	}

	for i, s := range t.services {
		if s.ID == id {
			return &t.services[i], nil
		}
	}
	return nil, nil
}


func ExampleServicedCLI_CmdLogExport_usage() {
	InitLogsAPITest(&mockapi.API{}, "serviced", "log", "export", "service")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    export - Exports all logs
	//
	// USAGE:
	//    command export [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced log export [YYYYMMDD]
	//
	// OPTIONS:
	//    --from 						yyyy.mm.dd
	//    --to 						yyyy.mm.dd
	//    --service '--service option --service option'	service ID or name (includes all sub-services)
	//    --out 						path to output file
	//    --debug, -d						Show additional diagnostic messages
}


func TestLogsCLI_CmdLogExport_SingleServiceName(t *testing.T) {
	fmt.Printf("TestLogsCLI_CmdExport_SingleServiceName\n")
	//call cmdExportLogs, verify that searchForService() is called

	mockAPI := mockapi.API{}

	mockAPI.On("GetServices").Return(DefaultTestServices, nil)
	foo := func (cfg api.ExportLogsConfig) bool {
		//debug.PrintStack()
		//fmt.Println("MOCKING ExportLogsConfig")
		//spew.Dump(cfg)
		return cfg.ServiceIDs != nil
	}
	//mockAPI.On("ExportLogs", mock.MatchedBy(func(cfg api.ExportLogsConfig) bool { debug.PrintStack(); fmt.Println("MOCKING ExportLogsConfig"); spew.Dump(cfg); return cfg.ServiceIDs != nil })).Return(nil)
	mockAPI.On("ExportLogs", mock.MatchedBy(foo)).Once().Return(nil)

	InitLogsAPITest(&mockAPI, "serviced", "log", "export", "--service", "zencommand")
	mockAPI.AssertExpectations(t)
}

func TestLogsCLI_CmdLogExport_MultipleServiceNames(t *testing.T) {
	fmt.Printf("TestLogsCLI_CmdExport_MultipleServiceNames\n")
	//call cmdExportLogs, verify that searchForService() is called

	mockAPI := mockapi.API{}

	mockAPI.On("GetServices").Return(DefaultTestServices, nil)
	foo := func (cfg api.ExportLogsConfig) bool {
		//debug.PrintStack()
		//fmt.Println("MOCKING ExportLogsConfig")
		//spew.Dump(cfg)
		return cfg.ServiceIDs != nil
	}
	//mockAPI.On("ExportLogs", mock.MatchedBy(func(cfg api.ExportLogsConfig) bool { fmt.Println("MOCKING ExportLogsConfig"); spew.Dump(cfg); return cfg.ServiceIDs != nil })).Once().Return(nil)
	mockAPI.On("ExportLogs", mock.MatchedBy(foo)).Once().Return(nil)
	InitLogsAPITest(&mockAPI, "serviced", "log", "export", "--service", "zencommand", "--service", "Zope")

	mockAPI.AssertExpectations(t)
}
