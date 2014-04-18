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
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/shell"
)

var empty interface{}

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

		{"templates", "Show application templates"},
		{"add-template", "Add application templates"},
		{"remove-template", "Remove application templates"},
		{"deploy-template", "Deploy application template"},

		{"hosts", "Display hosts"},
		{"add-host", "Add a host"},
		{"remove-host", "Remove a host"},

		{"pools", "Show pools"},
		{"add-pool", "Add pool"},
		{"remove-pool", "Remove pool"},
		{"list-pool-ips", "Show pool IP addresses"},
		{"add-virtual-ip", "Add a virtual IP address to a pool"},
		{"remove-virtual-ip", "Remove a virtual IP address from a pool"},
		{"auto-assign-ips", "Automatically assign IP addresses to service's endpoints requiring an explicit IP address"},
		{"manual-assign-ips", "Manually assign IP addresses to service's endpoints requiring an explicit IP address"},

		{"services", "Show services"},
		{"add-service", "Add a service"},
		{"remove-service", "Remove a service"},
		{"start-service", "Start a service"},
		{"stop-service", "Stop a service"},
		{"edit-service", "Edit a service"},

		{"proxy", "Start a proxy in the foreground"},

		{"show", "Show all available commands"},
		{"run", "Starts a shell to run defined commands from a container"},
		{"shell", "Starts a shell to run arbitrary system commands from a container"},
		{"rollback", "Rollback a service to a particular snapshot"},
		{"commit", "Commit a container to an image"},
		{"snapshot", "Snapshot a service"},
		{"delete-snapshot", "Snapshot a service"},
		{"snapshots", "Show snapshots for a service"},

		{"attach", "attach to a running service container and execute arbitrary bash command"},
		{"action", "attach to service instances and perform the predefined action"},
		{"backup", "Dump templates, services, images, and shared file system to a tgz file"},
		{"restore", "Import templates, services, images, and shared files sytems from a tgz file"},
	} {
		help += fmt.Sprintf("    %-30.30s%s\n", command[0], command[1])
	}
	fmt.Println(help)
	return nil
}

// Attempt to find the command give on the CLI by looking up the method on the
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
func getClient() (c dao.ControlPlane) {
	// setup the client
	c, err := serviced.NewControlClient(options.port)
	if err != nil {
		glog.Fatalf("Could not create a control plane client %v", err)
	}
	return c
}

var proxyOptions struct {
	muxport          int
	mux              bool
	servicedId       string
	tls              bool
	keyPEMFile       string
	certPEMFile      string
	servicedEndpoint string
	autorestart      bool
	logstash         bool
}

var proxyCmd *flag.FlagSet

func init() {
	gw := getDefaultGateway()

	proxyCmd = Subcmd("proxy", "[OPTIONS]", " SERVICE_ID COMMAND")
	proxyCmd.IntVar(&proxyOptions.muxport, "muxport", 22250, "multiplexing port to use")
	proxyCmd.BoolVar(&proxyOptions.mux, "mux", true, "enable port multiplexing")
	proxyCmd.BoolVar(&proxyOptions.tls, "tls", true, "enable TLS")
	proxyCmd.StringVar(&proxyOptions.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	proxyCmd.StringVar(&proxyOptions.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	proxyCmd.StringVar(&proxyOptions.servicedEndpoint, "endpoint", gw+":4979", "serviced endpoint address")
	proxyCmd.BoolVar(&proxyOptions.autorestart, "autorestart", true, "restart process automatically when it exits")
	proxyCmd.BoolVar(&proxyOptions.logstash, "logstash", true, "Forward service logs via logstash-forwarder")
	proxyCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: proxy [OPTIONS] SERVICE_ID COMMAND

SERVICE_ID   is the GUID of the service to run
COMMAND      is a quoted string that is the actual command to run

`)
		proxyCmd.PrintDefaults()
	}

}

// List the hosts associated with the control plane.
func (cli *ServicedCli) CmdHosts(args ...string) error {

	cmd := Subcmd("hosts", "[OPTIONS]", "List hosts")

	var verbose bool
	cmd.BoolVar(&verbose, "verbose", false, "Show JSON representation for each pool")

	var raw bool
	cmd.BoolVar(&raw, "raw", false, "Don't show the header line")

	if err := cmd.Parse(args); err != nil {
		return err
	}

	client := getClient()

	var hosts map[string]*dao.Host

	err := client.GetHosts(&empty, &hosts)
	if err != nil {
		glog.Fatalf("Could not get hosts %v", err)
	}

	if verbose == false {
		outfmt := "%-8s %-12s %-24s %-12s %-5d %-12d %-24s\n"

		if raw == false {
			fmt.Printf("%-8s %-12s %-24s %-12s %-5s %-12s %-24s\n",
				"ID",
				"POOL",
				"NAME",
				"ADDR",
				"CORES",
				"MEM",
				"NETWORK")
		} else {
			outfmt = "%s|%s|%s|%s|%d|%d|%s\n"
		}

		for _, h := range hosts {
			fmt.Printf(outfmt,
				h.Id,
				h.PoolId,
				h.Name,
				h.IpAddr,
				h.Cores,
				h.Memory,
				h.PrivateNetwork)
		}
	} else {
		hostsJson, err := json.MarshalIndent(hosts, " ", "  ")
		if err == nil {
			fmt.Printf("%s\n", hostsJson)
		}
	}
	return err
}

// Add a host to the control plane given the host:port.
func (cli *ServicedCli) CmdAddHost(args ...string) error {

	cmd := Subcmd("add-host", "[OPTIONS] HOST:PORT RESOURCE_POOL", "Add host")

	ipOpts := NewListOpts()
	cmd.Var(&ipOpts, "ips", "Comma separated list of IP available for endpoing assignment. If not set the default IP of the host is used")

	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}

	client, err := serviced.NewAgentClient(cmd.Arg(0))
	if err != nil {
		glog.Fatalf("Could not create connection to host %s: %v", args[0], err)
	}

	var remoteHost dao.Host
	//Add the IP used to connect
	err = client.GetInfo(ipOpts, &remoteHost)
	if err != nil {
		glog.Fatalf("Could not get remote host info: %v", err)
	}
	parts := strings.Split(cmd.Arg(0), ":")
	remoteHost.IpAddr = parts[0]
	remoteHost.PoolId = cmd.Arg(1)
	glog.V(0).Infof("Got host info: %v", remoteHost)

	controlPlane := getClient()

	var hostId string
	err = controlPlane.AddHost(remoteHost, &hostId)
	if err != nil {
		glog.Fatalf("Could not add host: %v", err)
	}
	fmt.Println(hostId)
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
		glog.Fatalf("Could not remove host: %v", err)
	}
	glog.V(0).Infof("Host %s removed.", cmd.Arg(0))
	return err
}

// A convinience struct for printing to command line
type poolWithHost struct {
	dao.ResourcePool
	Hosts []string
}

// Print a list of pools. Args are ignored.
func (cli *ServicedCli) CmdPools(args ...string) error {
	cmd := Subcmd("pools", "[OPTIONS]", "Display pools")

	var verbose bool
	cmd.BoolVar(&verbose, "verbose", false, "Show JSON representation for each pool")

	var raw bool
	cmd.BoolVar(&raw, "raw", false, "Don't show the header line")

	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	var pools map[string]*dao.ResourcePool
	err := controlPlane.GetResourcePools(&empty, &pools)
	if err != nil {
		glog.Fatalf("Could not get resource pools: %v", err)
	}

	if verbose == false {
		outfmt := "%-12s %-12s %-6d %-6d %-6d\n"

		if raw == false {
			fmt.Printf("%-12s %-12s %-6s %-6s %-6s\n", "ID", "PARENT", "CORE", "MEM", "PRI")
		} else {
			outfmt = "%s|%s|%d|%d|%d\n"
		}

		for _, pool := range pools {
			fmt.Printf(outfmt,
				pool.Id,
				pool.ParentId,
				pool.CoreLimit,
				pool.MemoryLimit,
				pool.Priority)
		}
	} else {
		poolsWithHost := make(map[string]poolWithHost)
		for _, pool := range pools {
			// get pool hosts
			var poolHosts []*dao.PoolHost
			err = controlPlane.GetHostsForResourcePool(pool.Id, &poolHosts)
			if err != nil {
				glog.Fatalf("Could not get hosts for Pool %s: %v", pool.Id, err)
			}
			hosts := make([]string, len(poolHosts))
			for i, hostPool := range poolHosts {
				hosts[i] = hostPool.HostId
			}
			poolsWithHost[pool.Id] = poolWithHost{*pool, hosts}
		}
		poolsWithHostJson, err := json.MarshalIndent(poolsWithHost, " ", "  ")
		if err == nil {
			fmt.Printf("%s\n", poolsWithHostJson)
		}
	}
	return err
}

// Add a new pool given some parameters.
func (cli *ServicedCli) CmdAddPool(args ...string) error {
	cmd := Subcmd("add-pool", "[options] POOLID CORE_LIMIT MEMORY_LIMIT PRIORITY", "Add resource pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) < 4 {
		cmd.Usage()
		return nil
	}
	pool, _ := dao.NewResourcePool(cmd.Arg(0))
	coreLimit, err := strconv.Atoi(cmd.Arg(1))
	if err != nil {
		glog.Fatalf("Bad core limit %s: %v", cmd.Arg(1), err)
	}
	pool.CoreLimit = coreLimit
	memoryLimit, err := strconv.Atoi(cmd.Arg(2))
	if err != nil {
		glog.Fatalf("Bad memory limit %s: %v", cmd.Arg(2), err)
	}
	pool.MemoryLimit = uint64(memoryLimit)
	controlPlane := getClient()
	var poolId string
	err = controlPlane.AddResourcePool(*pool, &poolId)
	if err != nil {
		glog.Fatalf("Could not add resource pool: %v", err)
	}
	fmt.Printf("%s\n", poolId)
	return err
}

// Revmove the given resource pool
func (cli *ServicedCli) CmdRemovePool(args ...string) error {
	cmd := Subcmd("remove-pool", "[options] POOLID ", "Remove resource pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	var unused int
	err := controlPlane.RemoveResourcePool(cmd.Arg(0), &unused)
	if err != nil {
		glog.Fatalf("Could not remove resource pool: %v", err)
	}
	glog.V(0).Infof("Pool %s removed.\n", cmd.Arg(0))
	return err
}

// Show pool IP address information
func (cli *ServicedCli) CmdListPoolIps(args ...string) error {
	cmd := Subcmd("list-pool-ips", "[options] POOLID ", "List pool IP addresses")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	poolId := cmd.Arg(0)

	var IPsInfo []dao.IPInfo
	err := controlPlane.RetrievePoolIPs(poolId, &IPsInfo)
	if err != nil {
		fmt.Printf("RetrievePoolIPs failed: %v", err)
		return err
	}

	outfmt := "%-20s %-20s %-20s\n"
	fmt.Printf(outfmt, "Interface Name", "IP Address", "Type")
	for _, IPInfo := range IPsInfo {
		fmt.Printf(outfmt, IPInfo.Interface, IPInfo.IP, IPInfo.Type)
	}

	return nil
}

func (cli *ServicedCli) CmdAddVirtualIp(args ...string) error {
	cmd := Subcmd("add-virtual-ip", "[options] POOLID IPADDRESS", "Add a virtual IP address to a pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	poolId := cmd.Arg(0)
	requestedIP := cmd.Arg(1)
	glog.Infof("Attempting to add virtual IP address %v to pool: %v", requestedIP, poolId)

	requestedVirtualIP := dao.VirtualIP{poolId, requestedIP}
	err := controlPlane.AddVirtualIp(requestedVirtualIP, nil)
	if err != nil {
		glog.Fatalf("Could not add virtual IP address: %v due to: %v", requestedVirtualIP.IP, err)
		return err
	}

	glog.Infof("Added virtual IP address %v to pool: %v", requestedIP, poolId)
	return nil
}

func (cli *ServicedCli) CmdRemoveVirtualIp(args ...string) error {
	cmd := Subcmd("remove-virtual-ip", "[options] POOLID IPADDRESS", "Remove a virtual IP address from a pool")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	poolId := cmd.Arg(0)
	requestedIP := cmd.Arg(1)
	glog.Infof("Attempting to remove virtual IP address %v from pool: %v", requestedIP, poolId)

	requestedVirtualIP := dao.VirtualIP{poolId, requestedIP}
	err := controlPlane.RemoveVirtualIp(requestedVirtualIP, nil)
	if err != nil {
		glog.Fatalf("Could not remove virtual IP address: %v due to: %v", requestedVirtualIP.IP, err)
		return err
	}

	glog.Infof("Removed virtual IP address %v from pool: %v", requestedIP, poolId)
	return nil
}

func (cli *ServicedCli) CmdAutoAssignIps(args ...string) error {
	cmd := Subcmd("auto-assign-ips", "[options] SERVICEID", "Automatically assign IP addresses to service's endpoints requiring an explicit IP address")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	serviceId := cmd.Arg(0)
	assignmentRequest := dao.AssignmentRequest{serviceId, "", true}
	err := controlPlane.AssignIPs(assignmentRequest, nil)
	if err != nil {
		glog.Fatalf("Could not automatically assign IPs: %v", err)
		return err
	}

	return nil
}

func (cli *ServicedCli) CmdManualAssignIps(args ...string) error {
	cmd := Subcmd("manual-assign-ips", "[options] SERVICEID IPADDRESS", "Manually assign IP addresses to service's endpoints requiring an explicit IP address")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	serviceId := cmd.Arg(0)
	setIpAddress := cmd.Arg(1)
	assignmentRequest := dao.AssignmentRequest{serviceId, setIpAddress, false}
	glog.Infof("Manually setting IP address to: %s", setIpAddress)

	err := controlPlane.AssignIPs(assignmentRequest, nil)
	if err != nil {
		glog.Fatalf("Could not manually assign IPs: %v", err)
		return err
	}

	return nil
}

// PortOpts type
type PortOpts map[string]dao.ServiceEndpoint

func NewPortOpts() PortOpts {
	return make(PortOpts)
}
func (opts *PortOpts) String() string {
	return fmt.Sprint(*opts)
}

// ListOpts type
type ListOpts []string

func NewListOpts() ListOpts {
	return make(ListOpts, 0)
}

func (opts *ListOpts) String() string {
	return fmt.Sprint(*opts)
}

func (opts *ListOpts) Set(value string) error {
	*opts = append(*opts, value)
	return nil
}

func (opts *PortOpts) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 3 {
		return fmt.Errorf("Malformed port specification: %v", value)
	}
	protocol := parts[0]
	if protocol != "tcp" && protocol != "udp" {
		return fmt.Errorf("Unsuppored protocol for port specification: %s", protocol)
	}
	portNum, err := strconv.Atoi(parts[1])
	if err != nil {
		return fmt.Errorf("Invalid port number: %s", parts[1])
	}
	portName := serviced.ApplicationType(parts[2])
	if len(portName) <= 0 {
		return fmt.Errorf("Endpoint name can not be empty")
	}
	port := fmt.Sprintf("%s:%d", protocol, portNum)
	(*opts)[port] = dao.ServiceEndpoint{Protocol: protocol, PortNumber: uint16(portNum), Application: string(portName)}
	return nil
}

func getDefaultGateway() string {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	localhost := "127.0.0.1"

	if err != nil {
		glog.V(2).Info("Error checking gateway: ", err)
		glog.V(1).Info("Could not get default gateway, using ", localhost)
		return localhost
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "default" {
			return fields[2]
		}
	}
	glog.V(1).Info("No gateway found, using ", localhost)
	return localhost
}

func ParseAddService(args []string) (*dao.Service, *flag.FlagSet, error) {
	cmd := Subcmd("add-service", "[OPTIONS] NAME POOLID IMAGEID COMMAND", "Add service.")

	if len(args) > 0 && args[0] != "--help" {
		cmd.SetOutput(ioutil.Discard)
	}

	flPortOpts := NewPortOpts()
	cmd.Var(&flPortOpts, "p", "Expose a port for this service (e.g. -p tcp:3306:mysql )")

	flServicePortOpts := NewPortOpts()
	cmd.Var(&flServicePortOpts, "q", "Map a remote service port (e.g. -q tcp:3306:mysql )")

	if err := cmd.Parse(args); err != nil {
		return nil, cmd, err
	}
	if len(cmd.Args()) < 4 {
		return nil, cmd, nil
	}

	service, err := dao.NewService()
	if err != nil {
		glog.Fatalf("Could not create service:%v\n", err)
	}
	service.Name = cmd.Arg(0)
	service.PoolId = cmd.Arg(1)
	service.ImageId = cmd.Arg(2)
	startup := cmd.Arg(3)
	for i := 4; i < len(cmd.Args()); i++ {
		startup = startup + " " + cmd.Arg(i)
	}
	glog.V(1).Info("endpoints discovered: ", flPortOpts)
	endPoints := make([]dao.ServiceEndpoint, len(flPortOpts)+len(flServicePortOpts))
	i := 0
	for _, endpoint := range flPortOpts {
		endpoint.Purpose = "remote"
		endPoints[i] = endpoint
		i++
	}
	for _, endpoint := range flServicePortOpts {
		endpoint.Purpose = "local"
		endPoints[i] = endpoint
		i++
	}
	service.Endpoints = endPoints
	service.Startup = startup
	return service, cmd, nil
}

// Add a service given a set of paramters.
func (cli *ServicedCli) CmdAddService(args ...string) error {
	service, cmd, err := ParseAddService(args)
	if err != nil {
		glog.Errorf(err.Error())
		return nil
	}
	if service == nil {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	service.Instances = 1
	glog.V(0).Info("Calling AddService.\n")
	var serviceId string
	err = controlPlane.AddService(*service, &serviceId)
	if err != nil {
		glog.Fatalf("Could not add services: %v", err)
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
	err := controlPlane.DeleteSnapshots(cmd.Arg(0), &unused)
	if err != nil {
		glog.Fatalf("Could not clean up service history: %v", err)
	}

	err = controlPlane.RemoveService(cmd.Arg(0), &unused)
	if err != nil {
		glog.Fatalf("Could not remove service: %v", err)
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
		glog.Fatalf("Could not start service: %v", err)
	}
	glog.V(0).Infof("Sevice scheduled to start on host %s\n", hostId)
	return err
}

// Schedule a service to stop given a service id.
func (cli *ServicedCli) CmdStopService(args ...string) error {
	cmd := Subcmd("stop-service", "SERVICEID", "Stop a service.")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	var unused int
	err := controlPlane.StopService(cmd.Arg(0), &unused)
	if err != nil {
		glog.Fatalf("Could not stop service: %v", err)
	}
	glog.V(0).Infoln("Sevice scheduled to stop.")
	return err
}

func getService(controlPlane *dao.ControlPlane, serviceId string) (service *dao.Service, err error) {
	// TODO: Replace with RPC call to get single service
	var services []*dao.Service
	err = (*controlPlane).GetServices(&empty, &services)
	if err != nil {
		return nil, err
	}
	for _, service = range services {
		if service.Id == serviceId || service.Name == serviceId {
			return service, nil
		}
	}
	return nil, err

}

func (cli *ServicedCli) CmdShell(args ...string) error {
	cmd := Subcmd("shell", "SERVICEID", "Open an interactive shell")
	var (
		service *dao.Service
		istty   bool
		rm      bool
	)
	cmd.BoolVar(&istty, "i", false, "Whether to run interactively")
	cmd.BoolVar(&rm, "rm", true, "Removes the container when the command completes")

	if err := cmd.Parse(args); err != nil {
		return nil
	}

	cp := getClient()
	service, err := getService(&cp, cmd.Arg(0))
	if err != nil {
		glog.Fatalf("Error while acquiring service: %s", err)
	} else if service == nil {
		glog.Fatalf("No service found: %s", cmd.Arg(0))
	}

	saveAs := ""
	if !rm {
		saveAs = serviced.GetLabel(service.Id)
		glog.Infof("Saving container as: %s", saveAs)
	}

	command := ""
	if len(cmd.Args()) > 1 {
		command = strings.Join(cmd.Args()[1:], " ")
	} else {
		glog.Fatalf("missing command")
	}

	config := shell.ProcessConfig{
		ServiceId: service.Id,
		IsTTY:     istty,
		SaveAs:    saveAs,
		Command:   command,
	}

	// TODO: Change me to call shell Forwarder
	dockercmd, err := shell.StartDocker(&config, options.port)
	if err != nil {
		glog.Fatalf("failed to connect to service: %s", err)
	}
	dockercmd.Stdin = os.Stdin
	dockercmd.Stdout = os.Stdout
	dockercmd.Stderr = os.Stderr

	return dockercmd.Run()
}

func (cli *ServicedCli) CmdShow(args ...string) error {
	cmd := Subcmd("show", "SERVICEID", "Shows the list of available serviced commands for a service container")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	cp := getClient()
	service, err := getService(&cp, cmd.Arg(0))
	if err != nil {
		glog.Fatalf("error while acquiring service: %s (%s)", cmd.Arg(0), err)
	} else if service == nil {
		glog.Fatalf("no service found: %s", cmd.Arg(0))
	}

	if len(service.Runs) == 0 {
		fmt.Printf("no commands found for service: %s\n", cmd.Arg(0))
		return nil
	}

	keys := []string{}
	for key, _ := range service.Runs {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	// Print the commands in tabular form
	const colwidth = 20
	output := make([]byte, 0, 80)
	buffer := bytes.NewBuffer(output)
	index := 0
	for index < len(keys) {
		for buffer.Len() < cap(output) && index < len(keys) {
			key := keys[index]
			if len(key) >= colwidth-2 {
				// truncate command name if it is longer than the column width
				key = fmt.Sprintf("(%s...)  ", key[:colwidth-7])
			} else {
				// append spaces to command name if it is shorter than the column width
				key = fmt.Sprintf("%s%s", key, strings.Repeat(" ", colwidth-len(key)))
			}
			buffer.Write([]byte(key))
			index += 1
		}
		// dump row to the screen and reset buffer
		fmt.Println(buffer.String())
		buffer.Reset()
	}

	return nil
}

func (cli *ServicedCli) CmdRun(args ...string) error {
	const TX_COMMIT = 42
	var (
		istty bool
		argv  []string
	)

	// Check the args
	cmd := Subcmd("run", "SERVICEID PROGRAM", "Runs serviced command on a service container")
	cmd.BoolVar(&istty, "i", false, "Whether to run interactively")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	argv = cmd.Args()
	if len(argv) < 2 {
		cmd.Usage()
		return nil
	}

	// Get the associated service
	cp := getClient()
	service, err := getService(&cp, argv[0])
	if err != nil {
		glog.Fatalf("error while acquiring service: %s", err)
	} else if service == nil {
		glog.Fatalf("no service found: %s", argv[0])
	}

	// Parse the command
	var command string
	if path, ok := service.Runs[argv[1]]; ok {
		argv[1] = path
		command = fmt.Sprintf("su - zenoss -c \"%s\"", strings.Join(argv[1:], " "))
	} else {
		glog.Fatalf("cannot access command: %s", argv[1])
	}

	// Start the container
	saveAs := serviced.GetLabel(service.Id)
	config := shell.ProcessConfig{
		ServiceId: service.Id,
		IsTTY:     istty,
		SaveAs:    saveAs,
		Command:   command,
	}

	// TODO: change me to use shell Forwarder
	dockercmd, err := shell.StartDocker(&config, options.port)
	if err != nil {
		glog.Fatalf("failed to connect to service: %s", err)
	}

	dockercmd.Stdin = os.Stdin
	dockercmd.Stdout = os.Stdout
	dockercmd.Stderr = os.Stderr

	exitcode, err := func(cmd *exec.Cmd) (int, error) {
		if err := cmd.Run(); err != nil {
			if e, ok := err.(*exec.ExitError); ok {
				if status, ok := e.Sys().(syscall.WaitStatus); ok {
					return status.ExitStatus(), nil
				}
			}
			return 0, err
		}
		return 0, nil
	}(dockercmd)

	if err != nil {
		glog.Fatalf("abnormal termination from shell command: %s", err)
	}

	dockercli, err := docker.NewClient("unix:///var/run/docker.sock")
	if err != nil {
		glog.Fatalf("unable to connect to the docker service: %s", err)
	}
	container, err := dockercli.InspectContainer(saveAs)
	if err != nil {
		glog.Fatalf("cannot acquire information about container: %s (%s)", saveAs, err)
	}
	glog.V(2).Infof("Container ID: %s", container.ID)

	switch exitcode {
	case TX_COMMIT:
		// Commit the container
		label := ""
		if err := cp.Commit(container.ID, &label); err != nil {
			glog.Fatalf("failed to commit: %s (%s)", container.ID, err)
		}
	default:
		// Delete the container
		if err := dockercli.StopContainer(container.ID, 10); err != nil {
			glog.Fatalf("failed to stop container: %s (%s)", container.ID, err)
		} else if err := dockercli.RemoveContainer(docker.RemoveContainerOptions{ID: container.ID}); err != nil {
			glog.Fatalf("failed to remove container: %s (%s)", container.ID, err)
		}
	}

	return nil
}

func (cli *ServicedCli) CmdRollback(args ...string) error {
	cmd := Subcmd("rollback", "SNAPSHOT_ID", "Reverts the container's DFS and image to a specified snapshot")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var unused int
	err := controlPlane.Rollback(cmd.Arg(0), &unused)
	if err != nil {
		glog.Errorf("Received an error: %s", err)
	}
	return err
}

// Commits a container to a docker image and updates the DFS with a new snapshot
func (cli *ServicedCli) CmdCommit(args ...string) (err error) {
	cmd := Subcmd("commit", "DOCKER_ID", "Commits a container and dfs to a new snapshot")
	err = cmd.Parse(args)
	if err != nil {
		return
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return
	}
	controlPlane := getClient()

	var label string
	err = controlPlane.Commit(cmd.Arg(0), &label)
	if err != nil {
		glog.Errorf("Received an error: %s", err)
	} else {
		fmt.Printf("%s\n", label)
	}

	return
}

func (cli *ServicedCli) CmdDeleteSnapshot(args ...string) error {
	cmd := Subcmd("delete-snapshot", "SNAPSHOT_ID", "Removes the specified snapshot")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var unused int
	err := controlPlane.DeleteSnapshot(cmd.Arg(0), &unused)
	if err != nil {
		glog.Errorf("Received an error: %s", err)
	}
	return err
}

func (cli *ServicedCli) CmdSnapshot(args ...string) error {
	cmd := Subcmd("snapshot", "SERVICEID", "Snapshots the container's DFS and image")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var snapshotId string
	if err := controlPlane.Snapshot(cmd.Arg(0), &snapshotId); err != nil {
		glog.Errorf("Received an error: %s", err)
		return err
	} else {
		fmt.Printf("%s\n", snapshotId)
	}
	return nil
}

func (cli *ServicedCli) CmdSnapshots(args ...string) error {
	cmd := Subcmd("snapshot", "SERVICEID", "Lists snapshots for the given service")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var snapshotIds []string
	if err := controlPlane.Snapshots(cmd.Arg(0), &snapshotIds); err != nil {
		glog.Errorf("Received an error: %s", err)
		return err
	} else {
		for _, snapshotId := range snapshotIds {
			fmt.Printf("%s\n", snapshotId)
		}
	}
	return nil
}

func (cli *ServicedCli) CmdGet(args ...string) error {
	cmd := Subcmd("get", "[options] SERVICEID FILE", "Download a file from a container and optional image id")

	var snapshot string
	cmd.StringVar(&snapshot, "snapshot", "", "Name of the container image (default: LATEST)")

	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var service dao.Service
	service.Id = cmd.Arg(0)
	service.ImageId = snapshot
	var file string
	file = cmd.Arg(1)
	err := controlPlane.Get(service, &file)
	return err
}

func (cli *ServicedCli) CmdRecv(args ...string) error {
	cmd := Subcmd("recv", "[options] SERVICEID FILE1..FILEN", "Upload a file to a container and optional image id")

	var snapshot string
	cmd.StringVar(&snapshot, "snapshot", "", "Name of the container image (default: LATEST)")

	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) < 2 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()

	var service dao.Service
	service.Id = cmd.Arg(0)
	service.ImageId = snapshot
	var files []string
	files = cmd.Args()[1:]
	err := controlPlane.Send(service, &files)
	return err
}

// Dump all templates and services to a tgz file.
// This includes a snapshot of all shared file systems
// and exports of all docker images the services depend on.
func (cli *ServicedCli) CmdBackup(args ...string) error {
	cmd := Subcmd("backup", "[BACKUP_DIRECTORY]", "Dump all templates and services to a tgz file")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	var backupFilePath string
	if err := controlPlane.Backup(cmd.Arg(0), &backupFilePath); err != nil {
		glog.Fatalf("%v", err)
		return err
	} else {
		fmt.Println("Backup saved to", backupFilePath)
		return nil
	}
}

// Restore templates, services, snapshots, and docker images from a tgz file.
// This is the inverse of CmdBackup.
func (cli *ServicedCli) CmdRestore(args ...string) error {
	cmd := Subcmd("backup", "[BACKUP_FILE_PATH}]", "Restore services from a tgz file")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	if len(cmd.Args()) != 1 {
		cmd.Usage()
		return nil
	}
	controlPlane := getClient()
	var unused int
	path, e := filepath.Abs(cmd.Arg(0))
	if e != nil {
		glog.Fatalf("Could not convert '%s' to an absolute file path: %v", cmd.Arg(0), e)
		return e
	}
	path = filepath.Clean(path)
	return controlPlane.Restore(filepath.Clean(path), &unused)
}
