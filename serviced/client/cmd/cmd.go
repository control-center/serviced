package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

func Run(args ...string) {
	var app = cli.NewApp()
	app.Name = "serviced"
	app.Usage = "A container-based management system."
	app.Commands = []cli.Command{
		{
			Name:   "proxy",
			Usage:  "Starts a proxy in the foreground.",
			Action: CmdProxy,
		}, {
			Name:  "pool",
			Usage: "Administers pool data.",
			Action: func(c *cli.Context) {
				var app = cli.NewApp()
				app.Name = fmt.Sprintf("%s %s", c.App.Name, "pool")
				app.Usage = "Administers pool data."
				app.Commands = []cli.Command{
					{
						Name:   "list",
						Usage:  "Lists all pools.",
						Action: CmdPoolList,
					}, {
						Name:   "add",
						Usage:  "Adds a resource pool.",
						Action: CmdPoolAdd,
					}, {
						Name:      "remove",
						ShortName: "rm",
						Usage:     "Removes an existing pool.",
						Action:    CmdPoolRemove,
					}, {
						Name:   "list-ips",
						Usage:  "Show pool IP addresses.",
						Action: CmdPoolListIPs,
					},
				}
				app.Run(append([]string{app.Name}, c.Args()...))
			},
		}, {
			Name:  "host",
			Usage: "Administers host data.",
			Action: func(c *cli.Context) {
				var app = cli.NewApp()
				app.Name = fmt.Sprintf("%s %s", c.App.Name, "host")
				app.Usage = "Administers host data."
				app.Commands = []cli.Command{
					{
						Name:   "list",
						Usage:  "Lists all hosts.",
						Action: CmdHostList,
					}, {
						Name:   "add",
						Usage:  "Adds a new host.",
						Action: CmdHostAdd,
					}, {
						Name:      "remove",
						ShortName: "rm",
						Usage:     "Removes an existing host.",
						Action:    CmdHostRemove,
					},
				}
				app.Run(append([]string{app.Name}, c.Args()...))
			},
		}, {
			Name:  "template",
			Usage: "Administers templates.",
			Action: func(c *cli.Context) {
				var app = cli.NewApp()
				app.Name = fmt.Sprintf("%s %s", c.App.Name, "template")
				app.Usage = "Administers template data."
				app.Commands = []cli.Command{
					{
						Name:   "list",
						Usage:  "Lists all templates.",
						Action: CmdTemplateList,
					}, {
						Name:   "add",
						Usage:  "Adds a new template.",
						Action: CmdTemplateAdd,
					}, {
						Name:      "remove",
						ShortName: "rm",
						Usage:     "Removes an existing template.",
						Action:    CmdTemplateRemove,
					}, {
						Name:   "deploy",
						Usage:  "Deploys template into a given pool.",
						Action: CmdTemplateDeploy,
					}, {
						Name:   "compile",
						Usage:  "Reads a given directory of service definitions to compile to a json struct.",
						Action: CmdTemplateCompile,
					},
				}
				app.Run(append([]string{app.Name}, c.Args()...))
			},
		}, {
			Name:  "service",
			Usage: "Administers services.",
			Action: func(c *cli.Context) {
				var app = cli.NewApp()
				app.Name = fmt.Sprintf("%s %s", c.App.Name, "service")
				app.Usage = "Administers services."
				app.Commands = []cli.Command{
					{
						Name:   "list",
						Usage:  "Lists all services.",
						Action: CmdServiceList,
					}, {
						Name:   "add",
						Usage:  "Adds a new service.",
						Action: CmdServiceAdd,
					}, {
						Name:      "remove",
						ShortName: "rm",
						Usage:     "Removes an existing service.",
						Action:    CmdServiceRemove,
					}, {
						Name:   "edit",
						Usage:  "Edits an existing service in a text editor",
						Action: CmdServiceEdit,
					}, {
						Name:   "auto-assign-ips",
						Usage:  "Automatically assign IP addresses to a service's endpoints requiring an explicit IP address.",
						Action: CmdServiceAutoIPs,
					}, {
						Name:   "manual-assign-ips",
						Usage:  "Manually assign IP addresses to a service's endpoints requiring an explicit IP address.",
						Action: CmdServiceManualIPs,
					}, {
						Name:   "start",
						Usage:  "Starts a service.",
						Action: CmdServiceStart,
					}, {
						Name:   "stop",
						Usage:  "Stops a service.",
						Action: CmdServiceStop,
					}, {
						Name:   "restart",
						Usage:  "Restarts a service.",
						Action: CmdServiceRestart,
					}, {
						Name:   "shell",
						Usage:  "Starts a service instance.",
						Action: CmdServiceShell,
					}, {
						Name:   "list-commands",
						Usage:  "Shows a list of predefined commands for a service that can be executed with `run`.",
						Action: CmdServiceListCmds,
					}, {
						Name:   "run",
						Usage:  "Runs a service command as exposed by `show`.",
						Action: CmdServiceRun,
					}, {
						Name:   "list-snapshots",
						Usage:  "Lists the snapshots for a service.",
						Action: CmdSnapshotList,
					}, {
						Name:   "snapshot",
						Usage:  "Takes a snapshot of the service.",
						Action: CmdSnapshotAdd,
					},
				}
				app.Run(append([]string{app.Name}, c.Args()...))
			},
		}, {
			Name:  "snapshot",
			Usage: "Administers environment snapshots.",
			Action: func(c *cli.Context) {
				var app = cli.NewApp()
				app.Name = fmt.Sprintf("%s %s", c.App.Name, "snapshot")
				app.Usage = "Administers environment snapshots."
				app.Commands = []cli.Command{
					{
						Name:   "list",
						Usage:  "Lists all snapshots.",
						Action: CmdSnapshotList,
					}, {
						Name:   "add",
						Usage:  "Take a snapshot of an existing service.",
						Action: CmdSnapshotAdd,
					}, {
						Name:      "remove",
						ShortName: "rm",
						Usage:     "Removes an existing snapshot.",
						Action:    CmdSnapshotRemove,
					}, {
						Name:   "commit",
						Usage:  "Snapshots and commits a given service instance",
						Action: CmdSnapshotCommit,
					}, {
						Name:   "rollback",
						Usage:  "Restores the environment to the state of the given snapshot.",
						Action: CmdSnapshotRollback,
					},
				}
				app.Run(append([]string{app.Name}, c.Args()...))
			},
		},
	}

	app.Run(args)
}
