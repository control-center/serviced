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

package cmd

import (
	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/servicedversion"

	"fmt"
)

// Initializer for serviced version
func (c *ServicedCli) initVersion() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "version",
		Usage:       "shows version information",
		Description: "",
		Action:      c.cmdVersion,
	})
}

// serviced version
func (c *ServicedCli) cmdVersion(ctx *cli.Context) {
	fmt.Printf("Version:    %s\n", servicedversion.Version)
	fmt.Printf("Gitcommit:  %s\n", servicedversion.Gitcommit)
	fmt.Printf("Gitbranch:  %s\n", servicedversion.Gitbranch)
	fmt.Printf("Giturl:     %s\n", servicedversion.Giturl)
	fmt.Printf("Date:       %s\n", servicedversion.Date)
	fmt.Printf("Buildtag:   %s\n", servicedversion.Buildtag)
	fmt.Printf("Release:    %s\n", servicedversion.Release)
	images := []string{
		fmt.Sprintf("%s:%s", isvcs.IMAGE_REPO, isvcs.IMAGE_TAG),
		fmt.Sprintf("%s:%s", isvcs.ZK_IMAGE_REPO, isvcs.ZK_IMAGE_TAG),
	}
	fmt.Printf("IsvcsImages: %v\n", images)
}
