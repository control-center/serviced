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
	"fmt"
	"os"

	"github.com/codegangsta/cli"
)

// Initializer for serviced backup and serviced restore
func (c *ServicedCli) initBackup() {
	c.app.Commands = append(
		c.app.Commands,
		cli.Command{
			Name:        "backup",
			Usage:       "Dump all templates and services to a tgz file",
			Description: "serviced backup DIRPATH",
			Action:      c.cmdBackup,
		},
		cli.Command{
			Name:        "restore",
			Usage:       "Restore templates and services from a tgz file",
			Description: "serviced restore FILEPATH",
			Action:      c.cmdRestore,
		},
	)
}

// serviced backup DIRPATH
func (c *ServicedCli) cmdBackup(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "backup")
		return
	}

	if path, err := c.driver.Backup(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if path == "" {
		fmt.Fprintln(os.Stderr, "received nil path to backup file")
	} else {
		fmt.Println(path)
	}
}

// serviced restore FILEPATH
func (c *ServicedCli) cmdRestore(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "restore")
		return
	}

	err := c.driver.Restore(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}
