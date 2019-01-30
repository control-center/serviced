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
	"github.com/control-center/serviced/cli/api"
)

// Initializer for serviced log
func (c *ServicedCli) initLog() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "log",
		Usage:       "Administers logs",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:        "export",
				Usage:       "Exports application log data",
				Description: "serviced log export",
				// TODO: BashComplete: c.printLogExportCompletion,
				Action: c.cmdExportLogs,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "from",
						Value: "",
						Usage: "yyyy.mm.dd",
					},
					cli.StringFlag{
						Name:  "to",
						Value: "",
						Usage: "yyyy.mm.dd",
					},
					cli.StringSliceFlag{
						Name:  "service",
						Value: &cli.StringSlice{},
						Usage: "service ID or name (includes all child services)",
					},
					cli.StringSliceFlag{
						Name:  "file",
						Value: &cli.StringSlice{},
						Usage: "the application log filename",
					},
					cli.StringFlag{
						Name:  "out",
						Value: "",
						Usage: "path to output file",
					},
					cli.BoolFlag{
						Name:  "debug, d",
						Usage: "Show additional diagnostic messages",
					},
					cli.StringFlag{
						Name:  "group-by",
						Value: "container",
						Usage: "Group results either by container, service or day",
					},
					cli.BoolFlag{
						Name:  "no-children, n",
						Usage: "Do not export child services",
					},
				},
			},
		},
	})
}

// serviced log export
func (c *ServicedCli) cmdExportLogs(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "export")
		return
	}
	from := ctx.String("from")
	to := ctx.String("to")
	outfile := ctx.String("out")

	var serviceIDs []string
	services := ctx.StringSlice("service")
	for _, service := range services {
		svc, _, err := c.searchForService(service,ctx.Bool("no-prefix-match"))
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		serviceIDs = append(serviceIDs, svc.ID)
	}

	groupBy := api.ExportGroupFromString(ctx.String("group-by"))
	if groupBy < 0 {
		fmt.Fprintf(os.Stderr,
			"ERROR: --group-by value '%s' is invalid; only 'container', 'day' or 'service' allowed\n",
			ctx.String("group-by"))
		return
	}

	cfg := api.ExportLogsConfig{
		ServiceIDs:       serviceIDs,
		FileNames:   ctx.StringSlice("file"),
		FromDate:         from,
		ToDate:           to,
		OutFileName:      outfile,
		Debug:            ctx.Bool("debug"),
		GroupBy:          groupBy,
		ExcludeChildren:  ctx.Bool("no-children"),
	}

	if err := c.driver.ExportLogs(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

// TODO: finish this, once flag completion is supported by cli.
// // Bash-completion command
// func (c *ServicedCli) printLogExportCompletion(ctx *cli.Context) {
// 	var e error
// 	flags := ctx.FlagCompletions()
// 	if len(flags) == 1 {
// 		switch flags[0] {
// 		case "from":
// 			{
// 				to := ""
// 				if ctx.IsSet("to") {
// 					if to, e = api.NormalizeYYYYMMDD(ctx.String("to")); e != nil {
// 						to = ""
// 					}
// 				}
// 				if days, e := api.LogstashDays(); e == nil {
// 					for _, yyyymmdd := range days {
// 						if to == "" || yyyymmdd <= to {
// 							fmt.Println(yyyymmdd)
// 						}
// 					}
// 				}
// 			}
// 		case "to":
// 			{
// 				from := ""
// 				if ctx.IsSet("from") {
// 					if from, e = api.NormalizeYYYYMMDD(ctx.String("from")); e != nil {
// 						from = ""
// 					}
// 				}
// 				if days, e := api.LogstashDays(); e == nil {
// 					for _, yyyymmdd := range days {
// 						if from == "" || yyyymmdd >= from {
// 							fmt.Println(yyyymmdd)
// 						}
// 					}
// 				}
// 			}
// 		case "service":
// 			{
// 				already := ctx.StringSlice("service")
// 				for _, serviceId := range c.services() {
// 					found := false
// 					for _, alreadyId := range already {
// 						if alreadyId == serviceId {
// 							found = true
// 							break
// 						}
// 						if !found {
// 							fmt.Println(serviceId)
// 						}
// 					}
// 				}
// 			}
// 		}
// 	}
// }
