// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"

	"github.com/zenoss/cli"
)

// Initializer for serviced log
func (c *ServicedCli) initLog() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "log",
		Usage:       "Administers logs",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "export",
				Usage:        "Exports all logs",
				Description:  "serviced log export [YYYYMMDD]",
				BashComplete: c.printLogDaysFirst,
				Action:       c.cmdExportLogs,
				Flags: []cli.Flag{
					cli.StringFlag{"out", "", "path to output directory (defaults to current directory)"},
				},
			},
		},
	})
}

// Bash-completion command that prints a list of available log days as the
// first argument
func (c *ServicedCli) printLogDaysFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, s := range c.logDays() {
		fmt.Println(s)
	}
}

// serviced log export [YYYYMMDD]
func (c *ServicedCli) cmdExportLogs(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) > 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "export")
		return
	}
	yyyymmdd := ""
	if len(args) == 1 {
		yyyymmdd = args[0]
	}
	dirpath := ctx.String("out")

	if err := c.driver.ExportLogs(yyyymmdd, dirpath); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func (c *ServicedCli) logDays() []string {
	return make([]string, 0) //TODO: figure out how to pull indices from elastigo
}
