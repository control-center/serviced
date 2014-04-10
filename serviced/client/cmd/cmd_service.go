package cmd

import (
	"encoding/json"
	"fmt"
	"os"

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
			Name:         "list",
			Usage:        "Lists all services.",
			Action:       c.cmdServiceList,
			BashComplete: c.printServices,

			Args: []string{
				"[SERVICEID]",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
		}, {
			Name:   "add",
			Usage:  "Adds a new service.",
			Action: c.cmdServiceAdd,
		}, {
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing service.",
			Action:       c.cmdServiceRemove,
			BashComplete: c.printServices,
		}, {
			Name:         "edit",
			Usage:        "Edits an existing service in a text editor.",
			Action:       c.cmdServiceEdit,
			BashComplete: c.printServices,
		}, {
			Name:         "auto-assign-ips",
			Usage:        "Automatically assign IP addresses to a service's endpoints requiring an explicit IP address.",
			Action:       c.cmdServiceAutoIPs,
			BashComplete: c.printServices,
		}, {
			Name:         "manual-assign-ips",
			Usage:        "Manually assign IP addresses to a service's endpoints requiring an explicit IP address.",
			Action:       c.cmdServiceManualIPs,
			BashComplete: c.printServices,
		}, {
			Name:         "start",
			Usage:        "Starts a service.",
			Action:       c.cmdServiceStart,
			BashComplete: c.printServices,
		}, {
			Name:         "stop",
			Usage:        "Stops a service.",
			Action:       c.cmdServiceStop,
			BashComplete: c.printServices,
		}, {
			Name:         "restart",
			Usage:        "Restarts a service.",
			Action:       c.cmdServiceRestart,
			BashComplete: c.printServices,
		}, {
			Name:         "shell",
			Usage:        "Starts a service instance.",
			Action:       c.cmdServiceShell,
			BashComplete: c.printServices,
		}, {
			Name:         "run",
			Usage:        "Runs a service command.",
			Action:       c.cmdServiceRun,
			BashComplete: c.printServiceRuns,
		}, {
			Name:         "list-snapshots",
			Usage:        "Lists the snapshots for a service.",
			Action:       c.cmdSnapshotList,
			BashComplete: c.printServices,
		}, {
			Name:         "snapshot",
			Usage:        "Takes a snapshot of the service.",
			Action:       c.cmdSnapshotAdd,
			BashComplete: c.printServices,
		},
	}
}

// printServices is the generic completion action for the service subcommand
// usage: serviced service COMMAND --generate-bash-completion
func (c *ServicedCli) printServices(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	services, err := c.driver.ListServices()
	if err != nil || services == nil || len(services) == 0 {
		return
	}

	for _, s := range services {
		fmt.Println(s.ID)
	}
}

// printServiceRuns is the generic completion action for the service run subcommand
// usage: serviced service run SERVICE COMMAND
func (c *ServicedCli) printServiceRuns(ctx *cli.Context) {
	args := ctx.Args()
	switch len(args) {
	case 0:
		services, err := c.driver.ListServices()
		if err != nil || services == nil || len(services) == 0 {
			return
		}
		for _, s := range services {
			fmt.Println(s.ID)
		}
	case 1:
		service, err := c.driver.GetService(args[0])
		if err != nil || service == nil {
			return
		}
		for run, _ := range service.Runs {
			fmt.Println(run)
		}
	}
}

// cmdServiceList is the command-line interaction for serviced service list
// usage: serviced service list
func (c *ServicedCli) cmdServiceList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		serviceID := ctx.Args()[0]
		if service, err := c.driver.GetService(serviceID); err != nil {
			fmt.Fprintf(os.Stderr, "error trying to receive service definition: %s\n", err)
		} else if service == nil {
			fmt.Fprintln(os.Stderr, "service not found")
		} else if jsonService, err := json.MarshalIndent(service, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal service definition: %s\n", err)
		} else {
			fmt.Println(string(jsonService))
		}
		return
	}

	services, err := c.driver.ListServices()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to receive service definitions: %s\n", err)
		return
	} else if services == nil || len(services) == 0 {
		fmt.Fprintf(os.Stderr, "no services found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonService, err := json.MarshalIndent(services, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal service definitions: %s\n", err)
		} else {
			fmt.Println(string(jsonService))
		}
	} else {
		// TODO: print a services map
		fmt.Println("serviced service list")
	}
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

// cmdServiceRun is the command-line interaction for serviced service run
// usage: serviced service run SERVICEID PROGRAM [ARGS ...]
func (c *ServicedCli) cmdServiceRun(ctx *cli.Context) {
	fmt.Println("serviced service run SERVICEID PROGRAM [ARGS ...]")
}