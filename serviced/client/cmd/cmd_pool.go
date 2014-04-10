package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/serviced/client/api"
)

// initPool is the initializer for serviced pool
func (c *ServicedCli) initPool() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "pool",
		Usage: "Administers pool data",
	})
	cmd.Commands = []cli.Command{
		{
			Name:         "list",
			Usage:        "Lists all pools",
			Action:       c.cmdPoolList,
			BashComplete: c.printPools,

			Args: []string{
				"[POOLID]",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
		},
		{
			Name:   "add",
			Usage:  "Adds a new resource pool",
			Action: c.cmdPoolAdd,

			Args: []string{
				"POOLID", "CORE_LIMIT", "MEMORY_LIMIT", "PRIORITY",
			},
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
	if len(ctx.Args()) > 0 {
		poolID := ctx.Args()[0]
		if pool, err := c.driver.GetPool(poolID); err != nil {
			fmt.Fprintf(os.Stderr, "error trying to retrieve resource pool: %s\n", err)
		} else if pool == nil {
			fmt.Fprintf(os.Stderr, "pool not found")
		} else if jsonPool, err := json.MarshalIndent(pool, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resource pool: %s", err)
		} else {
			fmt.Printf(string(jsonPool))
		}
		return
	}

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
		} else {
			fmt.Println(string(jsonPool))
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
	args := ctx.Args()
	if len(args) < 4 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	cfg := api.PoolConfig{}
	cfg.PoolID = args[0]
	if core, err := strconv.Atoi(args[1]); err != nil {
		fmt.Println("CORE_LIMIT must be a number")
		return
	} else {
		cfg.CoreLimit = core
	}
	if memory, err := strconv.ParseUint(args[2], 10, 64); err != nil {
		fmt.Println("MEMORY_LIMIT must be a number")
		return
	} else {
		cfg.MemoryLimit = memory
	}
	if priority, err := strconv.Atoi(args[3]); err != nil {
		fmt.Println("PRIORITY must be a number")
		return
	} else {
		cfg.Priority = priority
	}

	if pool, err := c.driver.AddPool(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if pool == nil {
		fmt.Fprintln(os.Stderr, "received nil pool")
	} else {
		fmt.Println(pool.ID)
	}
}

// cmdPoolRemove is the command-line interaction for serviced pool remove
// usage: serviced pool remove POOLID
func (c *ServicedCli) cmdPoolRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
	}

	if err := c.driver.RemovePool(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println("Done")
	}
}

// cmdPoolListIPs is the command-line interaction for serviced pool list-ips
// usage: serviced pool list-ips POOLID
func (c *ServicedCli) cmdPoolListIPs(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list-ips")
	}

	if ips, err := c.driver.ListPoolIPs(args[0]); err != nil {
		fmt.Fprintf(os.Stderr, "error trying to retrieve resource pool ips: %s\n", err)
		return
	} else if ips == nil || len(ips) == 0 {
		fmt.Fprintln(os.Stderr, "no resource pool ips found")
		return
	} else {
		tableIPs := newTable(0, 8, 2)
		tableIPs.PrintRow("Interface Name", "IP Address")
		for _, ip := range ips {
			tableIPs.PrintRow(ip.InterfaceName, ip.IPAddress)
		}
		tableIPs.Flush()
	}
}