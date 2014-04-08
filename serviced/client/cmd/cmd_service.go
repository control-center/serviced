package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
)

// initService is the initializer for serviced service
func (c *ServicedCli) initService() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "service",
		Usage: "Administers services",
	})
	cmd.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "Lists all services.",
			Action: c.cmdServiceList,
		}, {
			Name:   "add",
			Usage:  "Adds a new service.",
			Action: c.cmdServiceAdd,
		}, {
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing service.",
			Action:    c.cmdServiceRemove,
		}, {
			Name:   "edit",
			Usage:  "Edits an existing service in a text editor.",
			Action: c.cmdServiceEdit,
		}, {
			Name:   "auto-assign-ips",
			Usage:  "Automatically assign IP addresses to a service's endpoints requiring an explicit IP address.",
			Action: c.cmdServiceAutoIPs,
		}, {
			Name:   "manual-assign-ips",
			Usage:  "Manually assign IP addresses to a service's endpoints requiring an explicit IP address.",
			Action: c.cmdServiceManualIPs,
		}, {
			Name:   "start",
			Usage:  "Starts a service.",
			Action: c.cmdServiceStart,
		}, {
			Name:   "stop",
			Usage:  "Stops a service.",
			Action: c.cmdServiceStop,
		}, {
			Name:   "restart",
			Usage:  "Restarts a service.",
			Action: c.cmdServiceRestart,
		}, {
			Name:   "shell",
			Usage:  "Starts a service instance.",
			Action: c.cmdServiceShell,
		}, {
			Name:   "list-commands",
			Usage:  "Shows a list of predefined commands for a service that can be executed with `run`.",
			Action: c.cmdServiceListCmds,
		}, {
			Name:   "run",
			Usage:  "Runs a service command as exposed by `show`.",
			Action: c.cmdServiceRun,
		}, {
			Name:   "list-snapshots",
			Usage:  "Lists the snapshots for a service.",
			Action: c.cmdSnapshotList,
		}, {
			Name:   "snapshot",
			Usage:  "Takes a snapshot of the service.",
			Action: c.cmdSnapshotAdd,
		},
	}
}

// cmdServiceList is the command-line interaction for serviced service list
// usage: serviced service list
func (c *ServicedCli) cmdServiceList(ctx *cli.Context) {
	fmt.Println("serviced service list")
}

// cmdServiceAdd is the command-line interaction for serviced service add
// usage: serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND
func (c *ServicedCli) cmdServiceAdd(ctx *cli.Context) {
	fmt.Println("serviced service add [-p PORT] [-q REMOTE_PORT] NAME POOLID IMAGEID COMMAND")
}

// cmdServiceRemove is the command-line interaction for serviced service remove
// usage: serviced service remove SERVICEID
func (c *ServicedCli) cmdServiceRemove(ctx *cli.Context) {
	fmt.Println("serviced service remove SERVICEID")
}

// cmdServiceEdit is the command-line interaction for serviced service edit
// usage: serviced service edit SERVICEID
func (c *ServicedCli) cmdServiceEdit(ctx *cli.Context) {
	fmt.Println("serviced service edit SERVICEID")
}

// cmdServiceAutoIPs is the command-line interaction for serviced service auto-assign-ips
// usage: serviced service auto-assign-ips SERVICEID
func (c *ServicedCli) cmdServiceAutoIPs(ctx *cli.Context) {
	fmt.Println("serviced service auto-assign-ips SERVICEID")
}

// cmdServiceManualIPs is the command-line interaction for serviced service manual-assign-ips
// usage: serviced service manual-assign-ips SERVICEID IPADDRESS
func (c *ServicedCli) cmdServiceManualIPs(ctx *cli.Context) {
	fmt.Println("serviced service manual-assign-ips SERVICEID IPADDRESS")
}

// cmdServiceStart is the command-line interaction for serviced service start
// usage: serviced service start SERVICEID
func (c *ServicedCli) cmdServiceStart(ctx *cli.Context) {
	fmt.Println("serviced service start SERVICEID")
}

// cmdServiceStop is the command-line interaction for serviced service stop
// usage: serviced service stop SERVICEID
func (c *ServicedCli) cmdServiceStop(ctx *cli.Context) {
	fmt.Println("serviced service stop SERVICEID")
}

// cmdServiceRestart is the command-line interaction for serviced service restart
// usage: serviced service restart SERVICEID
func (c *ServicedCli) cmdServiceRestart(ctx *cli.Context) {
	fmt.Println("serviced service restart SERVICEID")
}

// cmdServiceShell is the command-line interaction for serviced service shell
// usage: serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]
func (c *ServicedCli) cmdServiceShell(ctx *cli.Context) {
	fmt.Println("serviced service shell SERVICEID [-rm=false] [-i] COMMAND [ARGS ...]")
}

// cmdServiceListCmds is the command-line interaction for serviced service list-commands
// usage: serviced service list-commands SERVICEID
func (c *ServicedCli) cmdServiceListCmds(ctx *cli.Context) {
	fmt.Println("serviced service list-commands SERVICEID")
}

// cmdServiceRun is the command-line interaction for serviced service run
// usage: serviced service run SERVICEID PROGRAM [ARGS ...]
func (c *ServicedCli) cmdServiceRun(ctx *cli.Context) {
	fmt.Println("serviced service run SERVICEID PROGRAM [ARGS ...]")
}