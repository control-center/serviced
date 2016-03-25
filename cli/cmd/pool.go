// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/pool"
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
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "Show JSON format",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "ID",
						Usage: "Comma-delimited list describing which fields to display",
					},
				},
			}, {
				Name:  "add",
				Usage: "Adds a new resource pool",
				//Description:  "serviced pool add POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY REALM",
				Description:  "serviced pool add POOLID",
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
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "Show JSON format",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "InterfaceName,IPAddress,Type",
						Usage: "Comma-delimited list describing which fields to display",
					},
				},
			}, {
				Name:         "add-virtual-ip",
				Usage:        "Add a virtual IP address to a pool",
				Description:  "serviced pool add-virtual-ip POOLID IPADDRESS NETMASK BINDINTERFACE",
				BashComplete: c.printPoolsFirst,
				Action:       c.cmdAddVirtualIP,
			}, {
				Name:         "remove-virtual-ip",
				Usage:        "Remove a virtual IP address from a pool",
				Description:  "serviced pool remove-virtual-ip POOLID IPADDRESS",
				BashComplete: c.printPoolsFirst,
				Action:       c.cmdRemoveVirtualIP,
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
	fmt.Println(strings.Join(c.pools(), "\n"))
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
		t := NewTable(ctx.String("show-fields"))
		t.Padding = 6
		for _, p := range pools {
			t.AddRow(map[string]interface{}{
				"ID": p.ID,
			})
		}
		t.Print()
	}
}

// serviced pool add POOLID
func (c *ServicedCli) cmdPoolAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	cfg := api.PoolConfig{}
	cfg.PoolID = args[0]

	/* Disabled until enforced. See ZEN-11450
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
	*/

	/* TODO: 1.1
	if len(args) > 2 {
		cfg.Realm = args[2]
	}
	*/

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
		if p, err := c.driver.GetResourcePool(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else if p == nil {
			fmt.Fprintf(os.Stderr, "%s: pool not found", id)
		} else if err := c.driver.RemoveResourcePool(id); err != nil {
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

	if poolIps, err := c.driver.GetPoolIPs(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	} else if poolIps.HostIPs == nil || (len(poolIps.HostIPs) == 0 && len(poolIps.VirtualIPs) == 0) {
		fmt.Fprintln(os.Stderr, "no resource pool IPs found")
		return
	} else if ctx.Bool("verbose") {
		if jsonPoolIP, err := json.MarshalIndent(poolIps.HostIPs, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal resource pool IPs: %s", err)
		} else {
			fmt.Println(string(jsonPoolIP))
		}
	} else {
		t := NewTable(ctx.String("show-fields"))
		for _, ip := range poolIps.HostIPs {
			t.AddRow(map[string]interface{}{
				"InterfaceName": ip.InterfaceName,
				"IPAddress":     ip.IPAddress,
				"Type":          "static",
			})
		}
		for _, ip := range poolIps.VirtualIPs {
			t.AddRow(map[string]interface{}{
				"InterfaceName": ip.BindInterface,
				"IPAddress":     ip.IP,
				"Type":          "virtual",
			})
		}
		t.Padding = 6
		t.Print()
	}
}

// serviced pool add-virtual-ip POOLID IPADDRESS NETMASK BINDINTERFACE
func (c *ServicedCli) cmdAddVirtualIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 4 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add-virtual-ip")
		return
	}

	requestVirtualIP := pool.VirtualIP{PoolID: args[0], IP: args[1], Netmask: args[2], BindInterface: args[3]}
	if err := c.driver.AddVirtualIP(requestVirtualIP); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	}

	fmt.Println("Added virtual IP:", args[1])
}

// serviced pool remove-virtual-ip POOLID IPADDRESS
func (c *ServicedCli) cmdRemoveVirtualIP(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove-virtual-ip")
		return
	}

	requestVirtualIP := pool.VirtualIP{PoolID: args[0], IP: args[1], Netmask: "", BindInterface: ""}
	if err := c.driver.RemoveVirtualIP(requestVirtualIP); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return
	} else {
		fmt.Printf("Removed virtual IP: %v from pool %v\n", args[1], args[0])
	}
}
