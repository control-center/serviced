package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
)

// initPool is the initializer for serviced pool
func (c *ServicedCli) initPool() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:   "pool",
		Usage:  "Administers pool data",
		Action: cmdDefault,
	})
	cmd.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "Lists all pools",
			Action: c.cmdPoolList,
		},
		{
			Name:   "add",
			Usage:  "Adds a new resource pool",
			Action: c.cmdPoolAdd,
		},
		{
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing resource pool",
			Action:    c.cmdPoolRemove,
		},
		{
			Name:   "list-ips",
			Usage:  "Lists the IP addresses for a resource pool",
			Action: c.cmdPoolListIPs,
		},
	}
}

// cmdPoolList is the command-line interaction for serviced pool list
// usage: serviced pool list
func (c *ServicedCli) cmdPoolList(ctx *cli.Context) {
	fmt.Println("serviced pool list")
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