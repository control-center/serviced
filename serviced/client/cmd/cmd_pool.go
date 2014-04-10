package cmd

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/zenoss/cli"
)

// initPool is the initializer for serviced pool
func (c *ServicedCli) initPool() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "pool",
		Usage: "Administers pool data",
	})
	cmd.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "Lists all pools",
			Action: c.cmdPoolList,

			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
		},
		{
			Name:   "add",
			Usage:  "Adds a new resource pool",
			Action: c.cmdPoolAdd,
		},
		{
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing resource pool",
			Action:       c.cmdPoolRemove,
			BashComplete: c.printPools,
		},
		{
			Name:         "list-ips",
			Usage:        "Lists the IP addresses for a resource pool",
			Action:       c.cmdPoolListIPs,
			BashComplete: c.printPools,
		},
	}
}

// printPools is the generic completion action for each pool action
// usage: serviced pool COMMAND --generate-bash-completion
func (c *ServicedCli) printPools(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	pools, err := c.driver.ListPools()
	if err != nil || pools == nil || len(pools) == 0 {
		return
	}

	for _, p := range pools {
		fmt.Println(p.ID)
	}
}

// cmdPoolList is the command-line interaction for serviced pool list
// usage: serviced pool list
func (c *ServicedCli) cmdPoolList(ctx *cli.Context) {
	pools, err := c.driver.ListPools()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to retrieve resource pools: %s\n", err)
		return
	} else if pools == nil || len(pools) == 0 {
		fmt.Fprintln(os.Stderr, "no resource pools found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonPool, err := json.MarshalIndent(pools, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resource pool list: %s", err)
		}
	} else {
		tablePool := newTable(0, 8, 2)
		tablePool.PrintRow("ID", "PARENT", "CORE", "MEM", "PRI")
		for _, p := range pools {
			tablePool.PrintRow(p.ID, p.ParentID, p.CoreLimit, p.MemoryLimit, p.Priority)
		}
		tablePool.Flush()
	}
}

// cmdPoolAdd is the command-line interaction for serviced pool add
// usage: serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
func (c *ServicedCli) cmdPoolAdd(ctx *cli.Context) {
	fmt.Println("serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY")
}

// cmdPoolRemove is the command-line interaction for serviced pool remove
// usage: serviced pool remove POOLID
func (c *ServicedCli) cmdPoolRemove(ctx *cli.Context) {
	fmt.Println("serviced pool remove POOLID")
}

// cmdPoolListIPs is the command-line interaction for serviced pool list-ips
// usage: serviced pool list-ips POOLID
func (c *ServicedCli) cmdPoolListIPs(ctx *cli.Context) {
	fmt.Println("serviced pool list-ips POOLID")
}