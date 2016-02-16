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
	"github.com/control-center/serviced/utils"
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
	New(DefaultServerAPITest, utils.TestConfigReader(map[string]string{})).Run(args)
}

func ExampleSerivcedCLI_CmdServer_good() {
	InitServerAPITest("serviced", "--master", "--allow-loop-back=true", "server")
	InitServerAPITest("serviced", "--agent", "--endpoint", "10.20.30.40", "server")
	InitServerAPITest("serviced", "--agent", "--master", "--allow-loop-back=true", "server")

	// Output:
	// starting server
	// starting server
	// starting server
}
