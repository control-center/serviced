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

package cmd

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

func (c *ServicedCli) initKey() {
	key_command := cli.Command{
		Name:        "key",
		Usage:       "Displays host's public key",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Shows host public key",
				Description:  "serviced key list HostID",
				BashComplete: c.printHostsFirst,
				Action:       c.cmdHostKey,
			},
		},
	}

	c.app.Commands = append(c.app.Commands, key_command)
}

func (c *ServicedCli) cmdHostKey(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list")
		return
	}
	hostID := args[0]
	if host, err := c.driver.GetHost(hostID); host == nil || err != nil {
		fmt.Fprintln(os.Stderr, "Host not found.")
		return
	}
	key, err := c.driver.GetHostPublicKey(hostID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Could not retrieve host's public key.")
		return
	}
	fmt.Printf(string(key), "\n\n")
}
