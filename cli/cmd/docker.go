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
)

// initDocker is the initializer for serviced docker
func (c *ServicedCli) initDocker() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "docker",
		Usage:       "Docker administration commands",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "sync",
				Usage:       "serviced docker sync",
				Description: "sync pushes all images to local registry - allows single host to easily be made master for multi-host",
				Action:      c.cmdRegistrySync,
			}, {
				Name:        "reset-registry",
				Usage:       "serviced docker reset-registry",
				Description: "Pulls images from the docker registry and updates the index",
				Action:      c.cmdResetRegistry,
			}, {
				Name:        "migrate-registry",
				Usage:       "service docker migrate-registry",
				Description: "Pull images from the provided docker registry and updates the index",
				Action:      c.cmdMigrateRegistry,
				Flags: []cli.Flag{
					cli.StringFlag{"registry", "", "host:port where the registry is running"},
					cli.BoolFlag{"override, f", "overrides all existing image records"},
				},
			},
		},
	})
}

// serviced docker sync
func (c *ServicedCli) cmdRegistrySync(ctx *cli.Context) {
	err := c.driver.RegistrySync()
	if err != nil {
		glog.Fatalf("error syncing docker images to local registry: %s", err)
	}
}

// serviced reset-registry
func (c *ServicedCli) cmdResetRegistry(ctx *cli.Context) {
	if err := c.driver.ResetRegistry(); err != nil {
		glog.Fatalf("error while resetting the registry: %s", err)
	}
}

// serviced migrate-registry
func (c *ServicedCli) cmdMigrateRegistry(ctx *cli.Context) {
	endpoint := ctx.String("registry")
	override := ctx.Bool("override")
	if err := c.driver.UpgradeRegistry(endpoint, override); err != nil {
		glog.Fatalf("error while upgrading the registry: %s", err)
	}
}
