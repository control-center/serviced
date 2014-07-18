// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
