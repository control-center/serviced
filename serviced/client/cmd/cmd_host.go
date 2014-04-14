package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/serviced/client/api"
)

// Initializer for serviced host subcommands
func (c *ServicedCli) initHost() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "host",
		Usage: "Administers host data",
	})
	cmd.Commands = []cli.Command{
		{
			Name:         "list",
			Usage:        "Lists all hosts.",
			Action:       c.cmdHostList,
			BashComplete: c.printHostsFirst,

			Args: []string{
				"[HOSTID]",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
		}, {
			Name:         "add",
			ShortName:    "+",
			Usage:        "Adds a new host",
			Action:       c.cmdHostAdd,
			BashComplete: c.printHostAdd,

			Args: []string{
				"HOST[:PORT]", "RESOURCE_POOL",
			},
			Flags: []cli.Flag{
				cli.StringSliceFlag{"ip", new(cli.StringSlice), "List of available endpoints. Default: host IP address"},
			},
		}, {
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing host",
			Action:       c.cmdHostRemove,
			BashComplete: c.printHostsAll,

			Args: []string{
				"HOSTID ...",
			},
		},
	}
}

// Returns a list of all available host IDs
func (c *ServicedCli) hosts() (data []string) {
	hosts, err := c.driver.ListHosts()
	if err != nil || hosts == nil || len(hosts) == 0 {
		return
	}

	data = make([]string, len(hosts))
	for i, h := range hosts {
		data[i] = h.ID
	}

	return
}

// Bash-completion command that prints a list of available hosts as the first
// argument
func (c *ServicedCli) printHostsFirst(ctx *cli.Context) {
	if len(ctx.Args() > 0) {
		return
	}

	for _, h := range c.hosts() {
		fmt.Println(h)
	}

	return
}

// Bash-completion command that prints a list of available hosts as all
// arguments
func (c *ServicedCli) printHostsAll(ctx *cli.Context) {
	args := ctx.Args()
	hosts := c.hosts()

	// If arg is a host, don't add to the list
	for _, h := range hosts {
		for _, a := range args {
			if h == a {
				goto next
			}
		}
		fmt.Println(h)
	next:
	}
}

// Bash-completion command that completes the POOLID as the second argument
func (c *ServicedCli) printHostAdd(ctx *cli.Context) {
	var output []string

	args := ctx.Args()
	switch len(args) {
	case 1:
		output = c.pools()
	}

	for _, o := range output {
		fmt.Println(o)
	}

	return
}

// serviced host list [--verbose, -v] [HOSTID]
func (c *ServicedCli) cmdHostList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		hostID := ctx.Args()[0]
		if host, err := c.driver.GetHost(hostID); err != nil {
			fmt.Fprintf(os.Stderr, "error trying to retrieve host: %s\n", err)
		} else if host == nil {
			fmt.Fprintln(os.Stderr, "host not found")
		} else if jsonHost, err := json.MarshalIndent(host, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host: %s", err)
		} else {
			fmt.Println(string(jsonHost))
		}
		return
	}

	hosts, err := c.driver.ListHosts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to retrieve hosts: %s\n", err)
		return
	} else if hosts == nil || len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "no hosts found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonHost, err := json.MarshalIndent(hosts, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host list: %s", err)
		} else {
			fmt.Println(string(jsonHost))
		}
	} else {
		tableHost := newTable(0, 8, 2)
		tableHost.PrintRow("ID", "POOL", "NAME", "ADDR", "CORES", "MEM", "NETWORK")
		for _, h := range hosts {
			tableHost.PrintRow(h.ID, h.PoolID, h.Name, h.IPAddr, h.Cores, h.Memory, h.PrivateNetwork)
		}
		tableHost.Flush()
	}
}

// serviced host add [[--ip IP] ...] HOST[:PORT] POOLID
func (c *ServicedCli) cmdHostAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	cfg := api.HostConfig{
		IPAddr: strings.SplitN(args[0], ":", 2)[0],
		PoolID: args[1],
		IPs:    ctx.StringSlice("ip"),
	}

	if host, err := c.driver.AddHost(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if host == nil {
		fmt.Fprintln(os.Stderr, "received nil host")
	} else {
		fmt.Println(host.ID)
	}
}

// serviced host remove HOSTID ...
func (c *ServicedCli) cmdHostRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	for _, id := range args {
		if err := c.driver.RemoveHost(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else {
			fmt.Println(id)
		}
	}
}