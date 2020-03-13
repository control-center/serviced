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
	"fmt"
	"os/exec"
	"strings"

	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/utils"
)

var (
	// Version of Control Center
	Version string
	// GoVersion version of Go lang
	GoVersion string
	// Date when CC was built
	Date string
	// Gitcommit git hash
	Gitcommit string
	// Gitbranch git branch
	Gitbranch string
	// Buildtag build label
	Buildtag string
	// Release label
	Release string

	plog = logging.PackageLogger()
)

// ServicedVersion is version and build metadata
type ServicedVersion struct {
	Version   string
	GoVersion string
	Date      string
	Gitcommit string
	Gitbranch string
	Buildtag  string
	Release   string
}

func init() {
	var err error
	Release, err = GetPackageRelease("serviced")
	if err != nil {
		plog.WithError(err).Debug("Unable to get version information.")
	}
}

// GetVersion returns a ServicedVersion object.
func GetVersion() ServicedVersion {
	return ServicedVersion{
		Version:   Version,
		GoVersion: GoVersion,
		Date:      Date,
		Gitcommit: Gitcommit,
		Gitbranch: Gitbranch,
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
		e := fmt.Errorf(
			"unable to retrieve release of package '%s' with command:'%s' output: %s  error: %s",
			pkg, command, output, err,
		)
		return "", e
	}

	release := strings.TrimSuffix(string(output), "\n")
	return release, nil
}

// getCommandToGetPackageRelease returns the command to get the package release
func getCommandToGetPackageRelease(pkg string) []string {
	command := []string{}
	if utils.Platform == utils.Rhel {
		command = []string{
			"bash", "-c", fmt.Sprintf("rpm -q --qf '%%{VERSION}-%%{Release}\n' %s", pkg),
		}
	} else {
		command = []string{
			"bash",
			"-o", "pipefail",
			"-c", fmt.Sprintf("dpkg -s %s | awk '/^Version/{print $NF;exit}'", pkg),
		}
	}

	return command
}
