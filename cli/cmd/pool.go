package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"

	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/cli/api"
)

// Initializer for serviced pool subcommands
func (c *ServicedCli) initPool() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "pool",
		Usage:       "Administers pool data",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all pools",
				Description:  "serviced pool list [POOLID]",
				BashComplete: c.printPoolsFirst,
				Action:       c.cmdPoolList,
				Flags: []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			}, {
				Name:         "add",
				Usage:        "Adds a new resource pool",
				Description:  "serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY",
				BashComplete: nil,
				Action:       c.cmdPoolAdd,
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing resource pool",
				Description:  "serviced pool remove POOLID ...",
				BashComplete: c.printPoolsAll,
				Action:       c.cmdPoolRemove,
			}, {
				Name:         "list-ips",
				Usage:        "Lists the IP addresses for a resource pool",
				Description:  "serviced pool list-ips POOLID",
				BashComplete: c.printPoolsFirst,
				Action:       c.cmdPoolListIPs,
				Flags: []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			},
		},
	})
}

// Returns a list of available pools
func (c *ServicedCli) pools() (data []string) {
	pools, err := c.driver.GetResourcePools()
	if err != nil || pools == nil || len(pools) == 0 {
		return
	}

	data = make([]string, len(pools))
	for i, p := range pools {
		data[i] = p.ID
	}

	return
}

// Bash-completion command that prints the list of available pools as the
// first argument
func (c *ServicedCli) printPoolsFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, p := range c.pools() {
		fmt.Println(p)
	}

	return
}

// Bash-completion command that prints the list of available pools as all
// arguments
func (c *ServicedCli) printPoolsAll(ctx *cli.Context) {
	args := ctx.Args()
	pools := c.pools()

	for _, p := range pools {
		for _, a := range args {
			if p == a {
				goto next
			}
		}
		fmt.Println(p)
	next:
	}
}

// serviced pool list [POOLID]
func (c *ServicedCli) cmdPoolList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		poolID := ctx.Args()[0]
		if pool, err := c.driver.GetResourcePool(poolID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if pool == nil {
			fmt.Fprintln(os.Stderr, "pool not found")
		} else if jsonPool, err := json.MarshalIndent(pool, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resource pool: %s", err)
		} else {
			fmt.Println(string(jsonPool))
		}
		return
	}

	pools, err := c.driver.GetResourcePools()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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

// serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY
func (c *ServicedCli) cmdPoolAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 4 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	var err error

	cfg := api.PoolConfig{}
	cfg.PoolID = args[0]

	cfg.CoreLimit, err = strconv.Atoi(args[1])
	if err != nil {
		fmt.Println("CORE_LIMIT must be a number")
		return
	}

	cfg.MemoryLimit, err = strconv.ParseUint(args[2], 10, 64)
	if err != nil {
		fmt.Println("MEMORY_LIMIT must be a number")
		return
	}

	cfg.Priority, err = strconv.Atoi(args[3])
	if err != nil {
		fmt.Println("PRIORITY must be a number")
		return
	}

	if pool, err := c.driver.AddResourcePool(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if pool == nil {
		fmt.Fprintln(os.Stderr, "received nil resource pool")
	} else {
		fmt.Println(pool.ID)
	}
}

// serviced pool remove POOLID ...
func (c *ServicedCli) cmdPoolRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
	}

	for _, id := range args {
		if err := c.driver.RemoveResourcePool(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else {
			fmt.Println(id)
		}
	}
}

// serviced pool list-ips POOLID
func (c *ServicedCli) cmdPoolListIPs(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "list-ips")
		return
	}

	if ips, err := c.driver.GetPoolIPs(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	} else if ips.HostIPs == nil || len(ips.HostIPs) == 0 {
		fmt.Fprintln(os.Stderr, "no resource pool ips found")
		return
	} else if ctx.Bool("verbose") {
		if jsonPoolIP, err := json.MarshalIndent(ips.HostIPs, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resource pool ips: %s", err)
		} else {
			fmt.Println(string(jsonPoolIP))
		}
	} else {
		tableIPs := newTable(0, 8, 2)
		tableIPs.PrintRow("Interface Name", "IP Address")
		for _, ip := range ips.HostIPs {
			tableIPs.PrintRow(ip.InterfaceName, ip.IPAddress)
		}
		tableIPs.Flush()
	}
}
