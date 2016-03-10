// Copyright 2014 The Serviced Authors.
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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package servicedversion

import (
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"

	"fmt"
	"os/exec"
	"strings"
)

var Version string
var Date string
var Gitbranch string
var Gitcommit string
var Giturl string
var Buildtag string
var Release string

type ServicedVersion struct {
	Version   string
	Date      string
	Gitbranch string
	Gitcommit string
	Giturl    string
	Buildtag  string
	Release   string
}

func init() {
	var err error
	Release, err = GetPackageRelease("serviced")
	if err != nil {
		glog.V(1).Infof("%s", err) // this should be Warningf or Infof, but zendev infinitely loops when seeing stderr
	}
}

func GetVersion() ServicedVersion {

	return ServicedVersion{
		Version:   Version,
		Date:      Date,
		Gitbranch: Gitbranch,
		Gitcommit: Gitcommit,
		Giturl:    Giturl,
		Buildtag:  Buildtag,
		Release:   Release,
	}
}

// GetPackageRelease returns the release version of the installed package
func GetPackageRelease(pkg string) (string, error) {
	if utils.Platform == utils.Darwin {
		return "", nil
	}

	command := getCommandToGetPackageRelease(pkg)
	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		e := fmt.Errorf("unable to retrieve release of package '%s' with command:'%s' output: %s  error: %s\n", pkg, command, output, err)
		return "", e
	}

	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", command, output)
	release := strings.TrimSuffix(string(output), "\n")
	return release, nil
}

// getCommandToGetPackageRelease returns the command to get the package release
func getCommandToGetPackageRelease(pkg string) []string {
	command := []string{}
	if utils.Platform == utils.Rhel {
		command = []string{"bash", "-c", fmt.Sprintf("rpm -q --qf '%%{VERSION}-%%{Release}\n' %s", pkg)}
	} else {
		command = []string{"bash", "-o", "pipefail", "-c", fmt.Sprintf("dpkg -s %s | awk '/^Version/{print $NF;exit}'", pkg)}
	}

	return command
}
