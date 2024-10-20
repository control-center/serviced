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
	"io/ioutil"
	"net"
	"os"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/utils"
	"github.com/pivotal-golang/bytefmt"
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
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "Show JSON format",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "ID,Auth,Pool,Name,Addr,RPCPort,Cores,RAM,Cur/Max/Avg,Network,Release",
						Usage: "Comma-delimited list describing which fields to display",
					},
				},
			}, {
				Name:         "add",
				Usage:        "Adds a new host",
				Description:  "serviced host add HOST:PORT RESOURCE_POOL",
				BashComplete: c.printHostAdd,
				Action:       c.cmdHostAdd,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "memory",
						Value: "",
						Usage: "Memory to allocate on this host, e.g. 20G, 50%",
					},
					cli.StringFlag{
						Name:  "nat-address",
						Value: "",
						Usage: "The HOST:PORT of the NAT for this delegate",
					},
					cli.StringFlag{
						Name:  "key-file, k",
						Value: "",
						Usage: "Name of the output host key file",
					},
					cli.BoolFlag{
						Name:  "register, r",
						Usage: "Register delegate keys on the host via ssh",
					},
				},
			}, {
				Name:         "add-private",
				ShortName:    "addme",
				Usage:        "Adds a new host and register via common key",
				Description:  "serviced host add HOST:PORT RESOURCE_POOL",
				BashComplete: c.printHostAddPrivate,
				Action:       c.cmdHostAddPrivate,
				Flags: []cli.Flag{
					cli.StringFlag{
						Name:  "memory",
						Value: "",
						Usage: "Memory to allocate on this host, e.g. 20G, 50%",
					},
					cli.StringFlag{
						Name:  "nat-address",
						Value: "",
						Usage: "The HOST:PORT of the NAT for this delegate",
					},
				},
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Removes an existing host",
				Description:  "serviced host remove HOSTID ...",
				BashComplete: c.printHostsAll,
				Action:       c.cmdHostRemove,
			}, {
				Name:        "register",
				Usage:       "Set the authentication keys to use for this host. When KEYSFILE is -, read from stdin.",
				Description: "serviced host register KEYSFILE",
				Action:      c.cmdHostRegister,
			}, {
				Name:         "set-memory",
				Usage:        "Set the memory allocation for a specific host",
				Description:  "serviced host set-memory HOSTID ALLOCATION",
				BashComplete: c.printHostsAll,
				Action:       c.cmdHostSetMemory,
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

// Bash-completion command that completes the POOLID as the first argument
func (c *ServicedCli) printHostAddPrivate(ctx *cli.Context) {
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
		if host, err := c.driver.GetHostWithAuthInfo(hostID); err != nil {
			fmt.Fprintln(os.Stderr, err)
			c.exit(1)
		} else if host == nil {
			fmt.Fprintln(os.Stderr, "host not found")
			c.exit(1)
		} else if jsonHost, err := json.MarshalIndent(host, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host: %s", err)
			c.exit(1)
		} else {
			fmt.Println(string(jsonHost))
		}
		return
	}

	hosts, err := c.driver.GetHostsWithAuthInfo()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	} else if hosts == nil || len(hosts) == 0 {
		fmt.Fprintln(os.Stderr, "no hosts found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonHost, err := json.MarshalIndent(hosts, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal host list: %s", err)
			c.exit(1)
		} else {
			fmt.Println(string(jsonHost))
		}
	} else {
		t := NewTable(ctx.String("show-fields"))
		for _, h := range hosts {
			var usage string
			if stats, err := c.driver.GetHostMemory(h.ID); err != nil {
				usage = "--"
			} else {
				usage = fmt.Sprintf("%s / %s / %s", bytefmt.ByteSize(uint64(stats.Last)), bytefmt.ByteSize(uint64(stats.Max)), bytefmt.ByteSize(uint64(stats.Average)))
			}
			t.AddRow(map[string]interface{}{
				"ID":          h.ID,
				"Auth":        h.Authenticated,
				"Pool":        h.PoolID,
				"Name":        h.Name,
				"Addr":        h.IPAddr,
				"RPCPort":     h.RPCPort,
				"Cores":       h.Cores,
				"RAM":         bytefmt.ByteSize(h.TotalRAM()),
				"Cur/Max/Avg": usage,
				"Network":     h.PrivateNetwork,
				"Release":     h.ServiceD.Release,
			})
		}
		t.Padding = 6
		t.Print()
	}
}

// serviced host add HOST:PORT POOLID [--memory SIZE|%] [--nat-address HOST:PORT]
func (c *ServicedCli) cmdHostAdd(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add")
		return
	}

	var address utils.URL
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

	// Parse/resolve the NAT address, if provided.
	var nat utils.URL
	natString := ctx.String("nat-address")
	if len(natString) > 0 {
		if err := nat.Set(natString); err != nil {
			fmt.Println(err)
			return
		}
		if natip := net.ParseIP(nat.Host); natip == nil {
			// NAT did not parse, try resolving
			addr, err := net.ResolveTCPAddr("tcp", natString)
			if err != nil {
				fmt.Printf("Could not resolve nat address (%s): %s\n", natString, err)
				return
			}
			nat.Host = addr.IP.String()
		}
		if strings.HasPrefix(nat.Host, "127.") {
			fmt.Printf("The nat address %s must not resolve to a loopback address\n", natString)
			return
		}
	}

	cfg := api.HostConfig{
		Address: &address,
		Nat:     &nat,
		PoolID:  args[1],
		Memory:  ctx.String("memory"),
	}

	host, privateKey, err := c.driver.AddHost(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
		return
	} else if host == nil {
		fmt.Fprintln(os.Stderr, "received nil host")
		c.exit(1)
		return
	}

	keyfileName := ctx.String("key-file")
	registerHost := ctx.Bool("register")
	c.outputDelegateKey(host, nat, privateKey, keyfileName, registerHost)
}

// serviced host add-private HOST:PORT POOLID [--memory SIZE|%] [--nat-address HOST:PORT]
func (c *ServicedCli) cmdHostAddPrivate(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "add-private")
		os.Exit(1)
	}

	var address utils.URL
	if err := address.Set(args[0]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	if ip := net.ParseIP(address.Host); ip == nil {
		// Host did not parse, try resolving
		addr, err := net.ResolveTCPAddr("tcp", args[0])
		if err != nil {
			fmt.Printf("Could not resolve %s.\n\n", args[0])
			os.Exit(1)
		}
		address.Host = addr.IP.String()
		if strings.HasPrefix(address.Host, "127.") {
			fmt.Printf("%s must not resolve to a loopback address\n\n", args[0])
			os.Exit(1)
		}
	}

	// Parse/resolve the NAT address, if provided.
	var nat utils.URL
	natString := ctx.String("nat-address")
	if len(natString) > 0 {
		if err := nat.Set(natString); err != nil {
			fmt.Println(err)
			os.Exit(1)
		}
		if natip := net.ParseIP(nat.Host); natip == nil {
			// NAT did not parse, try resolving
			addr, err := net.ResolveTCPAddr("tcp", natString)
			if err != nil {
				fmt.Printf("Could not resolve nat address (%s): %s\n", natString, err)
				os.Exit(1)
			}
			nat.Host = addr.IP.String()
		}
		if strings.HasPrefix(nat.Host, "127.") {
			fmt.Printf("The nat address %s must not resolve to a loopback address\n", natString)
			os.Exit(1)
		}
	}

	cfg := api.HostConfig{
		Address: &address,
		Nat:     &nat,
		PoolID:  args[1],
		Memory:  ctx.String("memory"),
	}

	host, keyblock, err := c.driver.AddHostPrivate(cfg)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	} else if keyblock == nil {
		fmt.Fprintln(os.Stderr, "received nil key")
		os.Exit(1)
	}

	c.outputCommonKey(host, nat, keyblock)
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
			c.exit(1)
		} else {
			fmt.Println(id)
		}
	}
}

// serviced host set-memory HOSTID MEMALLOC
func (c *ServicedCli) cmdHostSetMemory(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 2 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "set-memory")
		return
	}

	if err := c.driver.SetHostMemory(api.HostUpdateConfig{args[0], args[1]}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		c.exit(1)
	}
}

// serviced host register (KEYSFILE | -)
func (c *ServicedCli) cmdHostRegister(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) != 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "register")
		return
	}

	var (
		data []byte
		err  error
	)
	fname := args[0]
	switch fname {
	case "-":
		data, err = ioutil.ReadAll(os.Stdin)
	default:
		data, err = ioutil.ReadFile(fname)
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	if err := c.driver.RegisterHost(data); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

}
