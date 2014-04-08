package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zenoss/cli"
)

// initHost is the initializer for serviced host
func (c *ServicedCli) initHost() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "host",
		Usage: "Administers host data",
	})
	cmd.Commands = []cli.Command{
		{
			Name:  "list",
			Usage: "Lists all hosts.",
			Flags: []cli.Flag{
				cli.BoolFlag{"v", "Show JSON format"},
			},
			Action: c.cmdHostList,
		}, {
			Name:   "add",
			Usage:  "Adds a new host.",
			Action: c.cmdHostAdd,
		}, {
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing host.",
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

	if ctx.Bool("verbose") {
		if jsonHost, err := json.MarshalIndent(hosts, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host list: %s", err)
		} else {
			fmt.Println(jsonHost)
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
	fmt.Println("serviced host add HOST[:PORT] RESOURCE_POOL [[--ip IP] ...]")
}

// cmdHostRemove is the command-line interaction for serviced host remove
// usage: serviced host remove HOSTID
func (c *ServicedCli) cmdHostRemove(ctx *cli.Context) {
	fmt.Println("serviced host remove HOSTID")
}