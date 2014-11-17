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
	"net"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
)

// Initializer for serviced host subcommands
func (c *ServicedCli) initHost() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "host",
		Usage:       "Administers host data",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all hosts",
				Description:  "serviced host list [SERVICEID]",
				BashComplete: c.printHostsFirst,
				Action:       c.cmdHostList,
				Flags: []cli.Flag{
					cli.BoolFlag{"verbose, v", "Show JSON format"},
				},
			}, {
				Name:         "add",
				Usage:        "Adds a new host",
				Description:  "serviced host add HOST:PORT RESOURCE_POOL",
				BashComplete: c.printHostAdd,
				Action:       c.cmdHostAdd,
				Flags: []cli.Flag{
					cli.StringSliceFlag{"ip", &cli.StringSlice{}, "List of available endpoints"},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing host",
				Description:  "serviced host remove HOSTID ...",
				BashComplete: c.printHostsAll,
				Action:       c.cmdHostRemove,
			},
		},
	})
}

// Returns a list of all available host IDs
func (c *ServicedCli) hosts() (data []string) {
	hosts, err := c.driver.GetHosts()
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
	if len(ctx.Args()) > 0 {
		return
	}
	fmt.Println(strings.Join(c.hosts(), "\n"))
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
	fmt.Println(strings.Join(output, "\n"))
}

// serviced host list [--verbose, -v] [HOSTID]
func (c *ServicedCli) cmdHostList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		hostID := ctx.Args()[0]
		if host, err := c.driver.GetHost(hostID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if host == nil {
			fmt.Fprintln(os.Stderr, "host not found")
		} else if jsonHost, err := json.MarshalIndent(host, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host: %s", err)
		} else {
			fmt.Println(string(jsonHost))
		}
		return
	}

	hosts, err := c.driver.GetHosts()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
		tableHost := newtable(0, 8, 2)
		tableHost.printrow("ID", "POOL", "NAME", "ADDR", "RPCPORT", "CORES", "MEM", "NETWORK")
		for _, h := range hosts {
			tableHost.printrow(h.ID, h.PoolID, h.Name, h.IPAddr, h.RPCPort, h.Cores, h.Memory, h.PrivateNetwork)
		}
		tableHost.flush()
	}
}

// serviced host add HOST:PORT POOLID
func (c *ServicedCli) cmdHostAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	var address api.URL
	if err := address.Set(args[0]); err != nil {
		fmt.Println(err)
		return
	}
	if ip := net.ParseIP(address.Host); ip == nil {
		// Host did not parse, try resolving
		addr, err := net.ResolveTCPAddr("tcp", args[0])
		if err != nil {
			fmt.Printf("Could not resolve %s.\n\n", args[0])
			return
		}
		address.Host = addr.IP.String()
		if strings.HasPrefix(address.Host, "127.") {
			fmt.Printf("%s must not resolve to a loopback address\n\n", args[0])
			return
		}
	}

	cfg := api.HostConfig{
		Address: &address,
		PoolID:  args[1],
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
