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
	"github.com/codegangsta/cli"

	"github.com/control-center/serviced/script"
)

// Initializer for serviced metric
func (c *ServicedCli) initScript() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "script",
		Usage:       "serviced script",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "parse",
				Usage:       "Parse a script",
				Description: "serviced script parse file ",
				Before:      c.cmdScriptParse,
			},
			{
				Name:        "run",
				Usage:       "Run a script",
				Description: "serviced script run file ",
				Before:      c.cmdScriptRun,
				Flags: []cli.Flag{
					cli.BoolFlag{"no-op", "n", "Run through script without modifying system"},
					cli.StringFlag{"service-id", "", "Service ID argument for script, optional"},
				},
			},
		},
	})
}

// cmdScriptRun serviced script run filename
func (c *ServicedCli) cmdScriptRun(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return fmt.Printf("Incorrect Usage.\n\n")
	}
	fileName := args[0]
	config := script.Config{}
	//TODO: get the right dockerRegistry
	config.NoOp = ctx.GlobalBool("no-op")
	config.ServiceID = ctx.GlobalString("service-id")

	message, err := c.driver.ScriptRun(fileName, config)
	if err != nil {
		return err
	}
	return nil
}

// cmdScriptRun serviced script run filename
func (c *ServicedCli) cmdScriptParse(ctx *cli.Context) error {
	args := ctx.Args()
	if len(args) != 1 {
		return fmt.Printf("Incorrect Usage.\n\n")
	}
	fileName := args[0]
	config.NoOp = ctx.GlobalBool("no-op")
	config.ServiceID = ctx.GlobalString("service-id")
	message, err := c.driver.ScriptParse(fileName, config)
	if err != nil {
		return err
	}
	return nil
}
