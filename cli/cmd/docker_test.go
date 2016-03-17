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

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/utils"
)

const (
	OverrideFail = "Fail"
)

var DefaultDockerAPITest = DockerAPITest{}

var (
	ErrOverrideFailed = errors.New("override failed")
)

type DockerAPITest struct {
	api.API
}

func InitDockerAPITest(args ...string) {
	New(DefaultDockerAPITest, utils.TestConfigReader{}).Run(args)
}

func (t DockerAPITest) DockerOverride(newImage, oldImage string) error {
	switch newImage {
	case OverrideFail:
		return ErrOverrideFailed
	default:
		return nil
	}
}

func ExampleServicedCLI_CmdDockerOverride_usage() {
	InitDockerAPITest("serviced", "docker", "override")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    override - Replace an image in the registry with a new image
	//
	// USAGE:
	//    command override [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced docker override OLDIMAGE NEWIMAGE
	//
	// OPTIONS:
}

func ExampleServicedCli_cmdDockerOverride_fail() {
	pipeStderr(InitDockerAPITest, "serviced", "docker", "override", "anything", OverrideFail)

	// Output:
	// override failed
}

func ExampleServicedCli_cmdDockerOverride_success() {
	InitDockerAPITest("serviced", "docker", "override", "anything1", "anything2")

	// Output:
}
