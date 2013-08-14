/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package main

// This is here the command line arguments are parsed and executed.

import (
	"flag"
	"fmt"
	serviced "github.com/zenoss/serviced"
	agent "github.com/zenoss/serviced/agent"
	client "github.com/zenoss/serviced/client"
	"log"
	"reflect"
	"strconv"
	"strings"
)

// A type to represent the CLI. All the command will have the same signature.
// This makes it easy to call them arbitrarily.
type ServicedCli struct{}

// A helper function that creates a subcommand
func Subcmd(name, signature, description string) *flag.FlagSet {
	flags := flag.NewFlagSet(name, flag.ContinueOnError)
	flags.Usage = func() {
		fmt.Printf("\nUsage: serviced %s %s\n\n%s\n\n", name, signature, description)
		flags.PrintDefaults()
	}
	return flags
}

// Use reflection to aquire the give method by name. For example to get method
// CmdFoo, pass 'foo'. A method is returned. The second return argument
// indicates if the argument was found.
func (cli *ServicedCli) getMethod(name string) (reflect.Method, bool) {

	// Contruct the method name to be CmdFoo, where foo was passed
	methodName := "Cmd"
	for _, part := range strings.Split(name, "-") {
		methodName = methodName + strings.ToUpper(part[:1]) + strings.ToLower(part[1:])
	}
	return reflect.TypeOf(cli).MethodByName(methodName)
}

// Construct a new command line parsing object.
func NewServicedCli() (s *ServicedCli) {
	return &ServicedCli{}
}

// Show usage of serviced command line options.
func (cli *ServicedCli) CmdHelp(args ...string) error {
	if len(args) > 0 {
		method, exists := cli.getMethod(args[0])
		if !exists {
			fmt.Println("Error: Command not found:", args[0])
		} else {
			method.Func.CallSlice([]reflect.Value{
				reflect.ValueOf(cli),
				reflect.ValueOf([]string{"--help"}),
			})[0].Interface()
			return nil
		}
	}
	help := fmt.Sprintf("Usage: serviced [OPTIONS] COMMAND [arg...]\n\nA container based service management system.\n\nCommands:\n")
	for _, command := range [][2]string{
		{"hosts", "Display hosts"},
		{"update-host", "Update a host"},
		{"add-host", "Add a host"},
		{"remove-host", "Remove a host"},
		{"add-host-to-pool", "Add host to pool"},
		{"remove-host-from-pool", "Remove host to pool"},
		{"pools", "Show pools"},
		{"add-pool", "Add pool"},
		{"update-pool", "Update pool"},
		{"services", "Show services"},
		{"add-service", "Add a service"},
		{"remove-service", "Remote a service"},
		{"start-service", "Start a service"},
	} {
		help += fmt.Sprintf("    %-30.30s%s\n", command[0], command[1])
	}
	fmt.Println(help)
	return nil
}

// Attemp to find the command give on the CLI by looking up the method on the
// CLI interface. If found, execute it. Otherwise show usage.
func ParseCommands(args ...string) error {
	cli := NewServicedCli()

	if len(args) > 0 {
		method, exists := cli.getMethod(args[0])
		if !exists {
			fmt.Println("Error: Command not found:", args[0])
			return cli.CmdHelp(args[1:]...)
		}
		ret := method.Func.CallSlice([]reflect.Value{
			reflect.ValueOf(cli),
			reflect.ValueOf(args[1:]),
		})[0].Interface()
		if ret == nil {
			return nil
		}
		return ret.(error)
	}
	return cli.CmdHelp(args...)
}

// Create a client to the control plane.
func getClient() (c *client.ControlClient) {
	// setup the client
	c, err := client.NewControlClient(options.port)
	if err != nil {
		log.Fatalf("Could not create acontrol plane client %v", err)
	}
	return c
}

// List the hosts associated with the control plane.
func (cli *ServicedCli) CmdHosts(args ...string) error {

	cmd := Subcmd("hosts", "[OPTIONS]", "List hosts")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	client := getClient()

	var hosts map[string]*serviced.Host
	request := serviced.EntityRequest{}
	err := client.GetHosts(request, &hosts)
	if err != nil {
		log.Fatalf("Could not get hosts %v", err)
	}
	if len(hosts) > 0 {
		fmt.Printf("%10s  %20s  %15s  %3s  %12s\n", "Id", "Hostname", "IP Addr", "CPU", "Memory (GB)")
		for _, host := range hosts {
			fmt.Printf("%10s  %20s  %15s  %3d  %12.3f\n", host.Id, host.Name, host.IpAddr, host.Cores, float64(host.Memory)/1024.0/1024.0/1024.0)
		}
	} else {
		fmt.Printf("No hosts.\n")
	}
	return nil
}

// Add a host to the control plane given the host:port.
func (cli *ServicedCli) CmdAddHost(args ...string) error {

	cmd := Subcmd("add-host", "HOST:PORT", "Add host")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	client, err := agent.NewAgentClient(cmd.Arg(0))
	if err != nil {
		log.Fatalf("Could not create connection to host %s: %v", args[0], err)
	}

	var remoteHost serviced.Host
	err = client.GetInfo(0, &remoteHost)
	if err != nil {
		log.Fatalf("Could not get remote host info: %v", err)
	}
	log.Printf("Got host info: %v", remoteHost)

	controlPlane := getClient()
	var unused int

	err = controlPlane.AddHost(remoteHost, &unused)
	if err != nil {
		log.Fatalf("Could not add host: %v", err)
	}
	fmt.Println(remoteHost.Id)
	return err
}

// Update the host information. This method contacts the agent running on
// HOST:PORT to update the information assoicated with the host.
func (cli *ServicedCli) CmdUpdateHost(args ...string) error {

	cmd := Subcmd("update-host", "HOST:PORT", "Update the host information.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	client, err := agent.NewAgentClient(cmd.Arg(0))
	if err != nil {
		log.Fatalf("Could not create connection to host %s: %v", args[0], err)
	}

	var remoteHost serviced.Host
	err = client.GetInfo(0, &remoteHost)
	if err != nil {
		log.Fatalf("Could not get remote host info: %v", err)
	}
	log.Printf("Got host info: %v", remoteHost)

	controlPlane := getClient()
	var unused int

	err = controlPlane.UpdateHost(remoteHost, &unused)
	if err != nil {
		log.Fatalf("Could not update host: %v", err)
	}
	return err
}

// This method removes the given host (by HOSTID) from the system.
func (cli *ServicedCli) CmdRemoveHost(args ...string) error {
	cmd := Subcmd("remove-host", "HOSTID", "Remove the host.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}

	controlPlane := getClient()
	var unused int
	err := controlPlane.RemoveHost(cmd.Arg(0), &unused)
	if err != nil {
		log.Fatalf("Could not remove host: %v", err)
	}
	log.Printf("Host %s removed.", cmd.Arg(0))
	return err
}

// This method adds a host (by HOSTID) to a given pool (by POOLID).
// If the pool or host are not found an error is returned. If the relation
// already exists, an error is returned.
func (cli *ServicedCli) CmdAddHostToPool(args ...string) error {
	cmd := Subcmd("add-host-to-pool", "HOSTID POOLID", "Add host to pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	poolHost := serviced.PoolHost{cmd.Arg(0), cmd.Arg(1)}
	var unused int
	err := controlPlane.AddHostToResourcePool(poolHost, &unused)
	if err != nil {
		log.Fatalf("Could not add host to pool: %v", err)
	}
	return err
}

// This method removes an association of a host to a pool. Both the host and the
// pool need to exist or an error is returned. An error is also returned if the
// association already exists.
func (cli *ServicedCli) CmdRemoveHostFromPool(args ...string) error {
	cmd := Subcmd("remove-host-from-pool", "HOSTID POOLID", "Remove host from pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	poolHost := serviced.PoolHost{cmd.Arg(0), cmd.Arg(1)}
	var unused int
	err := controlPlane.RemoveHostFromResourcePool(poolHost, &unused)
	if err != nil {
		log.Fatalf("Could not remove host from pool: %v", err)
	}
	return err
}

// Print a list of pools. Args are ignored.
func (cli *ServicedCli) CmdPools(args ...string) error {
	cmd := Subcmd("pools", "[OPTIONS]", "Display pools")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	request := serviced.EntityRequest{}
	var pools map[string]*serviced.ResourcePool
	err := controlPlane.GetResourcePools(request, &pools)
	if err != nil {
		log.Fatalf("Could not get resource pools: %v", err)
	}
	if len(pools) > 0 {
		for _, pool := range pools {
			fmt.Printf("*********************** Id: %s ****************\n", pool.Id)
			fmt.Printf("Name:         %s\n", pool.Name)
			fmt.Printf("Core Limit:   %d\n", pool.CoreLimit)
			fmt.Printf("Memory Limit: %d\n", pool.MemoryLimit)
			fmt.Printf("Priority:     %d\n", pool.Priority)

			// get pool hosts
			var poolHosts []*serviced.PoolHost
			err = controlPlane.GetHostsForResourcePool(pool.Id, &poolHosts)
			if err != nil {
				log.Fatalf("Could not get hosts for Pool %s: %v", pool.Id, err)
			}
			fmt.Printf("Hosts:        %d\n", len(poolHosts))
			if len(poolHosts) > 0 {
				fmt.Printf("    ")
				for i, poolHost := range poolHosts {
					if i > 1 {
						fmt.Printf(",")
					}
					fmt.Printf("  %s", poolHost.HostId)
				}
				fmt.Printf("\n")
			}
		}
	} else {
		fmt.Printf("No resource pools found.\n")
	}
	return err
}

// Add a new pool given some parameters.
func (cli *ServicedCli) CmdAddPool(args ...string) error {
	cmd := Subcmd("add-pool", "[options] NAME CORE_LIMIT MEMORY_LIMIT PRIORITY", "Add resource pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) < 4 {
		cmd.Usage()
		return nil
	}
	pool, _ := serviced.NewResourcePool()
	pool.Name = cmd.Arg(0)
	coreLimit, err := strconv.Atoi(cmd.Arg(1))
	if err != nil {
		log.Fatal("Bad core limit %s: %v", cmd.Arg(1), err)
	}
	pool.CoreLimit = coreLimit
	memoryLimit, err := strconv.Atoi(cmd.Arg(2))
	if err != nil {
		log.Fatal("Bad memory limit %s: %v", cmd.Arg(2), err)
	}
	pool.MemoryLimit = uint64(memoryLimit)
	controlPlane := getClient()
	var unused int
	err = controlPlane.AddResourcePool(*pool, &unused)
	if err != nil {
		log.Fatalf("Could not add resource pool: %v", err)
	}
	fmt.Printf("%s\n", pool.Id)
	return err
}

// Update a pool interactively give the POOLID.
func (cli *ServicedCli) CmdUpdatePool(args ...string) error {
	cmd := Subcmd("update-pool", "POOLID", "Update resource pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	request := serviced.EntityRequest{}
	var pools map[string]*serviced.ResourcePool
	err := controlPlane.GetResourcePools(request, &pools)
	if err != nil {
		log.Fatalf("Could not get resource pools: %v", err)
	}
	pool, found := pools[cmd.Arg(0)]
	if !found {
		log.Fatal("Pool %s not found!", cmd.Arg(0))
	}

	updatedPool, _ := serviced.NewResourcePool()
	updatedPool.Id = pool.Id
	fmt.Printf("Pool Name? (%s) ", pool.Name)
	fmt.Scanf("%s\n", &updatedPool.Name)
	if updatedPool.Name == "" {
		updatedPool.Name = pool.Name
	}
	updatedPool.CoreLimit = pool.CoreLimit
	updatedPool.MemoryLimit = pool.MemoryLimit
	updatedPool.Priority = pool.Priority
	var unused int
	err = controlPlane.UpdateResourcePool(*updatedPool, &unused)
	if err != nil {
		log.Fatalf("Could not update resource pool: %v", err)
	}
	fmt.Printf("Updated resource pool %s\n", pool.Id)
	return err
}

// Print the list of available services.
func (cli *ServicedCli) CmdServices(args ...string) error {
	cmd := Subcmd("services", "[CMD]", "Show services")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	request := serviced.EntityRequest{}
	var services []*serviced.Service
	err := controlPlane.GetServices(request, &services)
	if err != nil {
		log.Fatalf("Could not get services: %v", err)
	}
	if len(services) == 0 {
		fmt.Printf("No services.\n")
	} else {

		for _, service := range services {
			fmt.Printf("*************** %s **************\n", service.Id)
			fmt.Printf("Name:      %36s\n", service.Name)
			fmt.Printf("Pool:      %36s\n", service.PoolId)
			fmt.Printf("ImageId:   %36s\n", service.ImageId)
			fmt.Printf("Instances: %36d\n", service.Instances)
			fmt.Printf("Startup:\n  %s\n\n", service.Startup)
		}
	}
	return err
}

// Add a service given a set of paramters.
func (cli *ServicedCli) CmdAddService(args ...string) error {
	cmd := Subcmd("add-service", "NAME POOLID IMAGEID COMMAND", "Add service.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) < 4 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	service, err := serviced.NewService()
	if err != nil {
		log.Fatal("Could not create service:%v\n", err)
	}
	service.Name = cmd.Arg(0)
	service.PoolId = cmd.Arg(1)
	service.ImageId = cmd.Arg(2)
	startup := cmd.Arg(3)
	for i := 4; i < len(cmd.Args()); i++ {
		startup = startup + " " + cmd.Arg(i)
	}
	service.Startup = startup

	log.Printf("Calling AddService.\n")
	var unused int
	err = controlPlane.AddService(*service, &unused)
	if err != nil {
		log.Fatalf("Could not add services: %v", err)
	}
	fmt.Println(service.Id)
	return err
}

// Remove a service given the SERVICEID.
func (cli *ServicedCli) CmdRemoveService(args ...string) error {
	cmd := Subcmd("remove-service", "SERVICEID", "Remove a service.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var unused int
	err := controlPlane.RemoveService(cmd.Arg(0), &unused)
	if err != nil {
		log.Fatalf("Could not remove service: %v", err)
	}
	return err
}

// Schedule a service to start given a service id.
func (cli *ServicedCli) CmdStartService(args ...string) error {
	cmd := Subcmd("start-service", "SERVICEID", "Start a service.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	var hostId string
	err := controlPlane.StartService(cmd.Arg(0), &hostId)
	if err != nil {
		log.Fatalf("Could not start service: %v", err)
	}
	log.Printf("Sevice scheduled to start on host %s\n", hostId)
	return err
}
