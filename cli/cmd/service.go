package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/cli/api"
)

// Initializer for serviced service subcommands
func (c *ServicedCli) initService() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "service",
		Usage:       "Administers services",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all services",
				Description:  "serviced service list [SERVICEID]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceList,
				Flags: []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			}, {
				Name:         "add",
				Usage:        "Adds a new service",
				Description:  "serviced service list NAME POOLID IMAGEID COMMAND",
				BashComplete: c.printServiceAdd,
				Action:       c.cmdServiceAdd,
				Flags: []cli.Flag{
					cli.GenericFlag{"p", &api.PortMap{}, "Expose a port for this service (e.g. -p tcp:3306:mysql)"},
					cli.GenericFlag{"q", &api.PortMap{}, "Map a remote service port (e.g. -q tcp:3306:mysql)"},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing service",
				Description:  "serviced service remove SERVICEID ...",
				BashComplete: c.printServicesAll,
				Action:       c.cmdServiceRemove,
			}, {
				Name:         "edit",
				Usage:        "Edits an existing service in a text editor",
				Description:  "serviced service edit SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceEdit,
				Flags: []cli.Flag{
					cli.StringFlag{"editor, e", os.Getenv("EDITOR"), "Editor used to update the service definition"},
				},
			}, {
				Name:         "assign-ip",
				Usage:        "Assigns an IP address to a service's endpoints requiring an explicit IP address",
				Description:  "serviced service assign-ip SERVICEID [IPADDRESS]",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceAssignIP,
			}, {
				Name:         "start",
				Usage:        "Starts a service",
				Description:  "serviced service start SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStart,
			}, {
				Name:         "stop",
				Usage:        "Stops a service",
				Description:  "serviced service stop SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceStop,
			}, {
				Name:         "proxy",
				Usage:        "Starts a server proxy for a container",
				Description:  "serviced service proxy SERVICEID COMMAND",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceProxy,
			}, {
				Name:         "shell",
				Usage:        "Starts a service instance",
				Description:  "serviced service shell SERVICEID COMMAND",
				BashComplete: c.printServicesFirst,
				Before:       c.cmdServiceShell,
				Flags: []cli.Flag{
					cli.StringFlag{"saveas, s", "", "saves the service instance with the given name"},
					cli.BoolFlag{"interactive, i", "runs the service instance as a tty"},
				},
			}, {
				Name:         "run",
				Usage:        "Runs a service command in a service instance",
				Description:  "serviced service run SERVICEID [COMMAND]",
				BashComplete: c.printServiceRun,
				Before:       c.cmdServiceRun,
				Flags: []cli.Flag{
					cli.StringFlag{"saveas, s", "", "saves the service instance with the given name"},
					cli.BoolFlag{"interactive, i", "runs the service instance as a tty"},
				},
			}, {
				Name:         "list-snapshots",
				Usage:        "Lists the snapshots for a service",
				Description:  "serviced service list-snapshots SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceListSnapshots,
			}, {
				Name:         "snapshot",
				Usage:        "Takes a snapshot of the service",
				Description:  "serviced service snapshot SERVICEID",
				BashComplete: c.printServicesFirst,
				Action:       c.cmdServiceSnapshot,
			},
		},
	})
}

// Returns a list of all the available service IDs
func (c *ServicedCli) services() (data []string) {
	svcs, err := c.driver.GetServices()
	if err != nil || svcs == nil || len(svcs) == 0 {
		return
	}

	data = make([]string, len(svcs))
	for i, s := range svcs {
		data[i] = s.Id
	}

	return
}

// Returns a list of runnable commands for a particular service
func (c *ServicedCli) serviceRuns(id string) (data []string) {
	svc, err := c.driver.GetService(id)
	if err != nil || svc == nil {
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
func (c *ServicedCli) printServicesFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, s := range c.services() {
		fmt.Println(s)
	}
}

// Bash-completion command that prints a list of available services as all
// arguments
func (c *ServicedCli) printServicesAll(ctx *cli.Context) {
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
func (c *ServicedCli) printServiceRun(ctx *cli.Context) {
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
func (c *ServicedCli) printServiceAdd(ctx *cli.Context) {
	var output []string

	args := ctx.Args()
	switch len(args) {
	case 1:
		output = c.pools()
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

	services, err := c.driver.GetServices()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	} else if services == nil || len(services) == 0 {
		fmt.Fprintln(os.Stderr, "no services found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonService, err := json.MarshalIndent(services, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal service definitions: %s\n", err)
		} else {
			fmt.Println(string(jsonService))
		}
	} else {
		servicemap := api.NewServiceMap(services)
		tableService := newTable(0, 8, 2)
		tableService.PrintRow("NAME", "SERVICEID", "STARTUP", "INST", "IMAGEID", "POOL", "DSTATE", "LAUNCH", "DEPID")

		var printTree func(string)
		printTree = func(root string) {
			services := servicemap.Get(root)
			for i, s := range services {
				tableService.PrintTreeRow(
					!(i+1 < len(services)),
					s.Name,
					s.Id,
					s.Startup,
					s.Instances,
					s.ImageId,
					s.PoolId,
					s.DesiredState,
					s.Launch,
					s.DeploymentId,
				)
				tableService.Indent()
				printTree(s.Id)
				tableService.Dedent()
			}
		}
		printTree("")
		tableService.Flush()
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
		LocalPorts:  ctx.Generic("p").(*api.PortMap),
		RemotePorts: ctx.Generic("q").(*api.PortMap),
	}

	if service, err := c.driver.AddService(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
	} else {
		fmt.Println(service.Id)
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
		fmt.Fprintln(os.Stderr, err)
		return
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "service not found")
		return
	}

	jsonService, err := json.MarshalIndent(service, " ", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "error marshalling service: %s\n", err)
		return
	}

	name := fmt.Sprintf("serviced_service_edit_%s", service.Id)
	reader, err := openEditor(jsonService, name, ctx.String("editor"))
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	if service, err := c.driver.UpdateService(reader); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if service == nil {
		fmt.Fprintln(os.Stderr, "received nil service")
	} else {
		fmt.Println(service.Id)
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

	var serviceID, ipAddress string
	serviceID = args[0]
	if len(args) > 1 {
		ipAddress = args[1]
	}

	cfg := api.IPConfig{
		ServiceID: serviceID,
		IPAddress: ipAddress,
	}

	if addresses, err := c.driver.AssignIP(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if addresses == nil || len(addresses) == 0 {
		fmt.Fprintln(os.Stderr, "received nil host resource")
	} else {
		for _, a := range addresses {
			fmt.Println(a.IPAddr)
		}
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
		fmt.Printf("Service scheduled to start on host: %s\n", host.ID)
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

	if err := c.driver.StopService(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Printf("Service scheduled to stop.\n")
	}
}

// serviced service proxy SERVICED COMMAND
func (c *ServicedCli) cmdServiceProxy(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		return nil
	}

	cfg := api.ProxyConfig{
		ServiceID: ctx.Args().First(),
		Command:   ctx.Args().Tail(),
	}

	if err := c.driver.StartProxy(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service proxy")
}

// serviced service shell [--saveas SAVEAS]  [--interactive, -i] SERVICEID COMMAND
func (c *ServicedCli) cmdServiceShell(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		return nil
	}

	cfg := api.ShellConfig{
		ServiceID: ctx.Args().First(),
		Command:   strings.Join(ctx.Args().Tail(), " "),
		SaveAs:    ctx.GlobalString("saveas"),
		IsTTY:     ctx.GlobalBool("interactive"),
	}

	if err := c.driver.StartShell(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service shell")
}

// serviced service run SERVICEID [COMMAND [ARGS ...]]
func (c *ServicedCli) cmdServiceRun(ctx *cli.Context) error {
	if len(ctx.Args()) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		return nil
	}

	cfg := api.ShellConfig{
		ServiceID: ctx.Args().First(),
		Command:   strings.Join(ctx.Args().Tail(), " "),
		SaveAs:    ctx.GlobalString("saveas"),
		IsTTY:     ctx.GlobalBool("interactive"),
	}

	if err := c.driver.StartShell(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	return fmt.Errorf("serviced service run")
}

// serviced service list-snapshot SERVICEID
func (c *ServicedCli) cmdServiceListSnapshots(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list-snapshots")
		return
	}

	if snapshots, err := c.driver.GetSnapshotsByServiceID(ctx.Args().First()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if snapshots == nil || len(snapshots) == 0 {
		fmt.Fprintln(os.Stderr, "no snapshots found")
	} else {
		for _, s := range snapshots {
			fmt.Println(s)
		}
	}
}

// serviced service snapshot SERVICEID
func (c *ServicedCli) cmdServiceSnapshot(ctx *cli.Context) {
	if len(ctx.Args()) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "snapshot")
		return
	}

	if snapshot, err := c.driver.AddSnapshot(ctx.Args().First()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if snapshot == "" {
		fmt.Fprintln(os.Stderr, "received nil snapshot")
	} else {
		fmt.Println(snapshot)
	}
}