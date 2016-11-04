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
	"testing"

	"github.com/control-center/serviced/cli/api"
	mocks "github.com/control-center/serviced/cli/api/apimocks"
	"github.com/control-center/serviced/utils"
	"github.com/stretchr/testify/mock"
)

type LogsCLITestCase struct {
	Args                             []string
	ExpectedExportLogsConfig         api.ExportLogsConfig
	Expected_ResolveServicePathCalls []string
}

func ExampleServicedCLI_CmdLogExport_usage() {
	runLogsAPITest(&mocks.API{}, "serviced", "log", "export", "service")

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
	//    serviced log export
	//
	// OPTIONS:
	//    --from 						yyyy.mm.dd
	//    --to 						yyyy.mm.dd
	//    --service '--service option --service option'	service ID or name (includes all sub-services)
	//    --out 						path to output file
	//    --debug, -d						Show additional diagnostic messages
}

func TestLogsCLI_CmdLogExport_SingleServiceName(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--service", "zencommand"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			ServiceIDs: []string{"test-service-3"},
		},
		Expected_ResolveServicePathCalls: []string{"zencommand"},
	}
	testCmdLogExport(t, testCase)
}

func TestLogsCLI_CmdLogExport_MultipleServiceNames(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--service", "zencommand", "--service", "Zope"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			ServiceIDs: []string{"test-service-3", "test-service-2"},
		},
		Expected_ResolveServicePathCalls: []string{"Zope", "zencommand"},
	}
	testCmdLogExport(t, testCase)
}

func TestLogsCLI_CmdLogExport_FromToDate(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--service", "test-service-3", "--from", "2001.05.01", "--to", "2010.06.27"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			ServiceIDs: []string{"test-service-3"},
			FromDate:   "2001.05.01",
			ToDate:     "2010.06.27",
		},
		Expected_ResolveServicePathCalls: []string{"test-service-3"},
	}
	testCmdLogExport(t, testCase)
}

func TestLogsCLI_CmdLogExport_FromDateOnly(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--from", "2015.09.21"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			FromDate: "2015.09.21",
		},
	}
	testCmdLogExport(t, testCase)
}
func TestLogsCLI_CmdLogExport_ToDateOnly(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--to", "2015.09.21"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			ToDate: "2015.09.21",
		},
	}
	testCmdLogExport(t, testCase)
}

func TestLogsCLI_CmdLogExport_Debug(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--debug"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			Debug: true,
		},
	}
	testCmdLogExport(t, testCase)
}

func TestLogsCLI_CmdLogExport_OutFileName(t *testing.T) {
	testCase := LogsCLITestCase{
		Args: []string{"serviced", "log", "export", "--out", "TestOutFileName"},
		ExpectedExportLogsConfig: api.ExportLogsConfig{
			OutFileName: "TestOutFileName",
		},
	}
	testCmdLogExport(t, testCase)
}

// compareStringSlices compares the contents of two string slices, without order.
// It was 'borrowed' from http://stackoverflow.com/a/36000696
func compareStringSlices(x, y []string) bool {
	if len(x) != len(y) {
		return false
	}
	// create a map of string -> int
	diff := make(map[string]int, len(x))
	for _, xval := range x {
		// 0 value for int is 0, so just increment a counter for the string
		diff[xval]++
	}
	for _, yval := range y {
		// If the string _y is not in diff bail out early
		if _, ok := diff[yval]; !ok {
			return false
		}
		diff[yval] -= 1
		if diff[yval] == 0 {
			delete(diff, yval)
		}
	}
	if len(diff) == 0 {
		return true
	}
	return false
}

// makeMatcher() returns a function that can be passed to the MatchedBy() function for mocking.
// It takes an api.ExportLogsConfig struct and returns a function that returns true if
func makeMatcher(c api.ExportLogsConfig) func(api.ExportLogsConfig) bool {
	return func(tc api.ExportLogsConfig) bool {
		//fmt.Println("Matcher: Expected:")
		//spew.Dump(c)
		//fmt.Println("MATCHER: Test Case:")
		//spew.Dump(tc)
		return compareStringSlices(c.ServiceIDs, tc.ServiceIDs) && c.FromDate == tc.FromDate && c.ToDate == tc.ToDate && c.Debug == tc.Debug
	}
}

func testCmdLogExport(t *testing.T, tc LogsCLITestCase) {
	mockAPI := mocks.API{}

	for _, name := range tc.Expected_ResolveServicePathCalls {
		mockAPI.On("ResolveServicePath", name).
			Return(serviceDetailsByName(name), nil)
	}
	matcher := makeMatcher(tc.ExpectedExportLogsConfig)
	mockAPI.On("ExportLogs", mock.MatchedBy(matcher)).Once().Return(nil)
	runLogsAPITest(&mockAPI, tc.Args...)
	mockAPI.AssertExpectations(t)
}

func runLogsAPITest(driver api.API, args ...string) {
	c := New(driver, utils.TestConfigReader(make(map[string]string)), MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}
