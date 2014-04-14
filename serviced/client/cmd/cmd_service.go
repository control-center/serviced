package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zenoss/cli"
)

// Initializer for serviced service subcommands
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
			BashComplete: c.printServicesFirst,

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
			Action: c.printServiceAdd,

			Args: []string{
				"NAME", "POOLID", "IMAGEID", "COMMAND",
			},
			Flags: []cli.Flag{
				cli.GenericSliceFlag{"p", new(api.PortOpts), "Expose a port for this service (e.g. -p tcp:3306:mysql )"},
				cli.GenericSliceFlag{"q", new(api.PortOpts), "Map a remote service port (e.g. -q tcp:3306:mysql )"},
			},
		}, {
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing service.",
			Action:       c.cmdServiceRemove,
			BashComplete: c.printServicesAll,

			Args: []string{
				"SERVICEID ...",
			},
		}, {
			Name:         "edit",
			Usage:        "Edits an existing service in a text editor.",
			Action:       c.cmdServiceEdit,
			BashComplete: c.printServicesFirst,

			Args: []string{
				"SERVICEID",
			},
			Flags: []cli.Flags{
				cli.StringFlag{"editor, e", os.Getenv("EDITOR"), "Editor used to update the service definition"},
			},
		}, {
			Name:         "assign-ip",
			Usage:        "Assign an IP address to a service's endpoints requiring an explicit IP address.",
			Action:       c.cmdServiceAssignIP,
			BashComplete: c.printServicesFirst,

			Args: []string{
				"SERVICEID", "[IPADDRESS]",
			},
		}, {
			Name:         "start",
			Usage:        "Starts a service.",
			Action:       c.cmdServiceStart,
			BashComplete: c.printServicesFirst,

			Args: []string{
				"SERVICEID",
			},
		}, {
			Name:         "stop",
			Usage:        "Stops a service.",
			Action:       c.cmdServiceStop,
			BashComplete: c.printServicesFirst,

			Args: []string{
				"SERVICEID",
			},
		}, {
			Name:         "shell",
			Usage:        "Starts a service instance.",
			Action:       c.cmdServiceShell,
			BashComplete: c.printServicesFirst,
		}, {
			Name:         "run",
			Usage:        "Runs a service command.",
			Action:       c.cmdServiceRun,
			BashComplete: c.printServiceRun,
		}, {
			Name:         "list-snapshots",
			Usage:        "Lists the snapshots for a service.",
			Action:       c.cmdSnapshotList,
			BashComplete: c.printServicesFirst,
		}, {
			Name:         "snapshot",
			Usage:        "Takes a snapshot of the service.",
			Action:       c.cmdSnapshotAdd,
			BashComplete: c.printServicesFirst,
		},
	}
}

// Returns a list of all the available service IDs
func (c *ServicedCli) services() (data []string) {
	svcs, err := c.driver.ListServices()
	if err != nil || svcs == nil || len(svcs) == 0 {
		return
	}

	data = make([]string, len(svcs))
	for i, s := range svcs {
		data[i] = s.ID
	}

	return
}

// Returns a list of runnable commands for a particular service
func (c *ServicedCli) serviceRuns(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svcs == nil {
		return
	}

	data = make([]string, len(svc.Runs))
	i := 0
	for r := range svc.Runs {
		data[i] = r
		i++
	}

	return
}

// Bash-completion command that prints a list of available services as the
// first argument
func (c *ServicedCli) printServicesFirst(c *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, s := range c.services() {
		fmt.Println(s)
	}
}

// Bash-completion command that prints a list of available services as all
// arguments
func (c *ServicedCli) printServicesAll(c *cli.Context) {
	args := ctx.Args()
	svcs := c.services()

	// If arg is a service don't add to the list
	for _, s := range svcs {
		for _, a := range args {
			if s == a {
				goto next
			}
		}
		fmt.Println(s)
	next:
	}
}

// Bash-completion command that completes the service ID as the first argument
// and runnable commands as the second argument
func (c *ServicedCli) printServiceRun(c *cli.Context) {
	var output []string

	args := ctx.Args()
	switch len(args) {
	case 0:
		output = c.services()
	case 1:
		output = c.serviceRuns(args[0])
	}

	for _, o := range output {
		fmt.Println(o)
	}
}

// Bash-completion command that completes the pool ID as the first argument
// and the docker image ID as the second argument
func (c *ServicedCli) printServiceAdd(c *cli.Context) {
	var output []string

	args := ctx.Args()
	switch len(args) {
	case 1:
		output := c.pools()
	case 2:
		// TODO: get a list of the docker images
	}

	for _, o := range output {
		fmt.Println(o)
	}
}

// serviced service list [--verbose, -v] [SERVICEID]
func (c *ServicedCli) cmdServiceList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		serviceID := ctx.Args()[0]
		if service, err := c.driver.GetService(serviceID); err != nil {
			fmt.Fprintln(os.Stderr, err)
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
		fmt.Fprintln(os.Stderr, err)
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
		newServiceMap(services).PrintTree("")
	}
}

// serviced service add [[-p PORT]...] [[-q REMOTE]...] NAME POOLID IMAGEID COMMAND
func (c *ServicedCli) cmdServiceAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 4 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	cfg := api.ServiceConfig{
		Name:        args[0],
		PoolID:      args[1],
		ImageID:     args[2],
		Command:     args[3],
		LocalPorts:  ctx.GenericSlice("p"),
		RemotePorts: ctx.GenericSlice("q"),
	}

	if service, err := c.driver.AddService(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
	} else {
		fmt.Println(service.ID)
	}
}

// serviced service remove SERVICEID ...
func (c *ServicedCli) cmdServiceRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	for _, id := range args {
		if err := c.driver.RemoveService(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else {
			fmt.Println(id)
		}
	}
}

// serviced service edit SERVICEID
func (c *ServicedCli) cmdServiceEdit(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "edit")
		return
	}

	service, err := c.driver.GetService(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, err)
		return
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "service not found")
		return
	}

	jsonService, err := json.MarshallIndent(service, " ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling service: %s\n", err)
		return
	}

	name := fmt.Sprintf("serviced_service_edit_%s", service.ID)
	reader, err := openEditor(jsonService, name, cli.StringFlag("editor"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if service, err := c.driver.UpdateService(reader); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
	} else {
		fmt.Println(service.ID)
	}
}

// serviced service assign-ip SERVICEID [IPADDRESS]
func (c *ServicedCli) cmdServiceAssignIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "assign-ip")
		return
	}

	cfg := api.IPConfig{
		ServiceID: args[0],
		IPAddress: args[1],
	}

	if hostResource, err := c.driver.AssignIP(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if hostResource == nil {
		fmt.Fprintln(os.Stderr, "received nil host resource")
	} else {
		fmt.Println(hostResource.IPAddress)
	}
}

// serviced service start SERVICEID
func (c *ServicedCli) cmdServiceStart(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "start")
		return
	}

	if host, err := c.driver.StartService(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if host == nil {
		fmt.Fprintln(os.Stderr, "received nil host")
	} else {
		fmt.Printf("Service scheduled to start on host: %s", host.ID)
	}
}

// serviced service stop SERVICEID
func (c *ServicedCli) cmdServiceStop(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "stop")
		return
	}

	if host, err := c.driver.StopService(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if host == nil {
		fmt.Fprintln(os.Stderr, "received nil host")
	} else {
		fmt.Printf("Service scheduled to stop on host: %s", host.ID)
	}
}

// serviced service shell [--saveas SAVEAS] SERVICEID COMMAND
func (c *ServicedCli) cmdServiceShell(ctx *cli.Context) {
	// TODO: Make me work with channels!
	/*
		args := ctx.Args()
		if len(args) < 2 {
			fmt.Printf("Incorrect Usage.\n\n")
			cli.ShowCommandHelp(ctx, "shell")
			return
		}

		cfg := api.ShellConfig{
			ServiceID: args[0],
			IsTTY: ctx.BoolFlag("interactive"),
			Remove: ctx.BoolFlag("rm"),
		}
	*/
}

// serviced service run SERVICEID COMMAND [ARGS ...]
func (c *Serviced) cmdServiceRun(ctx *cli.Context) {
	// TODO: same issue as with shell!
}

/* OTHER STUFFS */

// TODO: restore to parity!
type serviceMap map[string][]*service.Service

func newServiceMap(services []service.Service) (m serviceMap) {
	for _, s := range services {
		m[s.ParentID] = append(serviceMap[s.ParentID], &s)
	}

	return
}

func (m serviceMap) PrintTree(parentID string) {
	t := newTable(0, 8, 2)
	t.PrintRow("NAME", "SERVICEID", "STARTUP", "INST", "IMAGEID", "POOL", "DSTATE", "LAUNCH", "DEPIP")

	nextRow := func(id string, order int) {
		for _, s := range m[id] {
			t.PrintRow(s.Name, s.ID, s.Startup, s.Instances, s.ImageID, s.PoolID, s.DesiredState, s.Launch, s.DeploymentID)
			nextRow(s.ID)
		}
	}
	nextRow(parentID, 0)
}