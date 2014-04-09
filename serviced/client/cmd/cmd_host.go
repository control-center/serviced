package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/serviced/client/api"
)

// initHost is the initializer for serviced host
func (c *ServicedCli) initHost() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:   "host",
		Usage:  "Administers host data",
		Action: cmdDefault,
	})
	cmd.Commands = []cli.Command{
		{
			Name:  "list",
			Usage: "Lists all hosts.",
			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
			Action: c.cmdHostList,
		}, {
			Name:      "add",
			ShortName: "+",
			Usage:     "Adds a new host",
			Args:      []string{"HOST[:PORT]", "RESOURCE_POOL"},
			Flags: []cli.Flag{
				cli.StringSliceFlag{"ip", new(cli.StringSlice), "List of available IP endpoints.  Default: HOST"},
			},
			Action: c.cmdHostAdd,
		}, {
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing host",
			Args:      []string{"HOSTID"},
			Action:    c.cmdHostRemove,
		},
	}
}

// cmdHostList is the command-line interaction for serviced host list
// usage: serviced host list
func (c *ServicedCli) cmdHostList(ctx *cli.Context) {
	hosts, err := c.driver.ListHosts()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to retrieve hosts: %s\n", err)
		return
	} else if hosts == nil || len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "no hosts installed")
		return
	}

	if ctx.Bool("v") {
		if jsonHost, err := json.MarshalIndent(hosts, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host list: %s", err)
		} else {
			fmt.Println(string(jsonHost))
		}
	} else if ctx.Bool("complete") {
		for _, h := range hosts {
			fmt.Println(h.ID)
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

// cmdHostAdd is the command-line interaction for serviced host add
// usage: serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]
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

// cmdHostRemove is the command-line interaction for serviced host remove
// usage: serviced host remove HOSTID
func (c *ServicedCli) cmdHostRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	if err := c.driver.RemoveHost(args.First()); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println("Done")
	}
}