// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
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

	"github.com/control-center/serviced/cli/api"
)

var DefaultServerAPITest = ServerAPITest{}

type ServerAPITest struct {
	api.API
}

func (t ServerAPITest) StartServer() error {
	fmt.Println("starting server")
	return nil
}

func InitServerAPITest(args ...string) {
	New(DefaultServerAPITest, TestConfigReader(map[string]string{})).Run(args)
}

func ExampleSerivcedCLI_CmdServer_good() {
	InitServerAPITest("serviced", "--master", "server")
	InitServerAPITest("serviced", "--agent", "server")
	InitServerAPITest("serviced", "--agent", "--master", "server")

	// Output:
	// This master has been configured to be in pool: default
	// starting server
	// starting server
	// This master has been configured to be in pool: default
	// starting server
}

func ExampleServicedCLI_CmdServer_bad() {
	pipeStderr(InitServiceAPITest, "serviced", "server")

	// Output:
	// serviced cannot be started: no mode (master or agent) was specified
}
