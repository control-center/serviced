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

	"github.com/control-center/serviced/commons/docker"
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
				Description: "serviced script parse FILE",
				Action:      c.cmdScriptParse,
			},
			{
				Name:        "run",
				Usage:       "Run a script",
				Description: "serviced script run FILE [-n]",
				Action:      c.cmdScriptRun,
				Flags: []cli.Flag{
					cli.BoolFlag{"no-op, n", "Run through script without modifying system"},
				},
			},
			{
				Name:        "service",
				Usage:       "Run a script with a service as an argument",
				Description: "serviced script service SERVICEID FILE [-n]",
				Action:      c.cmdScriptSvcRun,
				Flags: []cli.Flag{
					cli.BoolFlag{"no-op, n", "Run through script without modifying system"},
				},
			},
		},
	})
}


// cmdScriptSvcRun serviced script <service> filename
func (c *ServicedCli) cmdScriptSvcRun(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 2 {
		fmt.Fprintln(os.Stderr, "Incorrect Usage.\n\n")
		c.exit(1)
		return
	}
	svcID := args[0]
	//verify service or translate to ID
	svc, err := c.searchForService(svcID)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}
	if svc == nil {
		fmt.Fprintf(os.Stderr, "service %s not found\n", svcID)
		c.exit(1)
		return
	}

	fileName := args[1]
	config := &script.Config{
		ServiceID: svc.ID,
	}
	runScript(c, ctx, fileName, config)
}

// cmdScriptRun serviced script run filename
func (c *ServicedCli) cmdScriptRun(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Incorrect Usage.\n\n")
		return
	}
	fileName := args[0]
	config := &script.Config{}
	runScript(c, ctx, fileName, config)
}

// cmdScriptRun serviced script run filename
func (c *ServicedCli) cmdScriptParse(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		fmt.Fprintln(os.Stderr, "Incorrect Usage.\n\n")
		return
	}
	fileName := args[0]
	config := script.Config{}
	err := c.driver.ScriptParse(fileName, &config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	return
}

func runScript(c *ServicedCli, ctx *cli.Context, fileName string, config *script.Config) {
	config.NoOp = ctx.Bool("no-op")
	config.DockerRegistry = docker.DEFAULT_REGISTRY
	err := c.driver.ScriptRun(fileName, config)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	}
}
