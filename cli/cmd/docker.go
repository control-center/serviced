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
	"github.com/zenoss/glog"

	"fmt"
)

// initDocker is the initializer for serviced docker
func (c *ServicedCli) initDocker() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "docker",
		Usage:       "Docker administration commands",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "squash",
				Usage:       "serviced docker squash IMAGE_NAME [DOWN_TO_LAYER] [NEW_NAME]",
				Description: "squash exports a docker image and flattens it down a base layer to reduce the number of total layers",
				Action:      c.cmdSquash,
				Flags: []cli.Flag{
					cli.StringFlag{"endpoint", "unix:///var/run/docker.sock", "docker endpoint"},
					cli.StringFlag{"tempdir", "", "temp directory"},
				},
			},
			{
				Name:        "sync",
				Usage:       "serviced docker sync",
				Description: "sync pushes all images to local registry - allows single host to easily be made master for multi-host",
				Action:      c.cmdRegistrySync,
				Flags: []cli.Flag{
					cli.StringFlag{"endpoint", "unix:///var/run/docker.sock", "docker endpoint"},
				},
			},
		},
	})
}

func (c *ServicedCli) cmdSquash(ctx *cli.Context) {

	imageName := ""
	baseLayer := ""
	newName := ""
	args := ctx.Args()
	switch len(ctx.Args()) {
	case 3:
		newName = args[2]
		fallthrough
	case 2:
		baseLayer = args[1]
		fallthrough
	case 1:
		imageName = args[0]
		break
	default:
		cli.ShowCommandHelp(ctx, "squash")
		return
	}

	imageID, err := c.driver.Squash(imageName, baseLayer, newName, ctx.String("tempdir"))
	if err != nil {
		glog.Fatalf("error squashing: %s", err)
	}
	fmt.Println(imageID)
}

// serviced docker sync
func (c *ServicedCli) cmdRegistrySync(ctx *cli.Context) {

	err := c.driver.RegistrySync()
	if err != nil {
		glog.Fatalf("error syncing docker images to local registry: %s", err)
	}
}
