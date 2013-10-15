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
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	clientlib "github.com/zenoss/serviced/client"
	"github.com/zenoss/serviced/proxy"

	"encoding/json"
	"flag"
	"fmt"
	"github.com/zenoss/glog"
	"io"
	"io/ioutil"
	"net"
	"os"
	"os/exec"
	"reflect"
	"strconv"
	"strings"
	"time"
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

		{"templates", "show application templates"},
		{"add-template", "add application templates"},
		{"remove-template", "remove application templates"},
		{"deploy-template", "deploy application template"},

		{"hosts", "Display hosts"},
		{"add-host", "Add a host"},
		{"remove-host", "Remove a host"},

		{"pools", "Show pools"},
		{"add-pool", "Add pool"},
		{"remove-pool", "Remove pool"},

		{"services", "Show services"},
		{"add-service", "Add a service"},
		{"remove-service", "Remote a service"},
		{"start-service", "Start a service"},
		{"stop-service", "Stop a service"},

		{"proxy", "start a proxy in the foreground"},
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
	c, err := clientlib.NewControlClient(options.port)
	if err != nil {
		glog.Fatalf("Could not create acontrol plane client %v", err)
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
	proxyCmd.Usage = func() {
		fmt.Fprintf(os.Stderr, `
Usage: proxy [OPTIONS] SERVICE_ID COMMAND

SERVICE_ID   is the GUID of the service to run
COMMAND      is a quoted string that is the actual command to run

`)
		proxyCmd.PrintDefaults()
	}

}

// Start a service proxy.
func (cli *ServicedCli) CmdProxy(args ...string) error {

	if err := proxyCmd.Parse(args); err != nil {
		return err
	}
	if len(proxyCmd.Args()) != 2 {
		proxyCmd.Usage()
		glog.Flush()
		os.Exit(2)
	}
	config := proxy.Config{}
	config.TCPMux.Port = proxyOptions.muxport
	config.TCPMux.Enabled = proxyOptions.mux
	config.TCPMux.UseTLS = proxyOptions.tls
	config.ServiceId = proxyCmd.Arg(0)
	config.Command = proxyCmd.Arg(1)

	if config.TCPMux.Enabled {
		go config.TCPMux.ListenAndMux()
	}

	go func(cmdString string) {
		cmd := exec.Command("bash", "-c", cmdString)
		stderr, err := cmd.StderrPipe()
		if err != nil {
			glog.Fatalf("Problem opening a stderr pipe to service: %s", err)
		}
		go io.Copy(os.Stderr, stderr)
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			glog.Fatalf("Problem opening a stdout pipe to service: %s", err)
		}
		go io.Copy(os.Stdout, stdout)
		glog.Infof("About to execute: %s", cmdString)
		err = cmd.Run()
		if err != nil {
			glog.Errorf("Problem running service: %v", err)
			time.Sleep(time.Minute)
			glog.Flush()
			os.Exit(1)
		}
		glog.Flush()
		os.Exit(0)
	}(config.Command)

	func() {
		client, err := proxy.NewLBClient(proxyOptions.servicedEndpoint)
		if err != nil {
			glog.Errorf("Could not create a client to endpoint %s: %s", proxyOptions.servicedEndpoint, err)
			return
		}
		defer client.Close()

		var endpoints map[string][]*serviced.ApplicationEndpoint
		err = client.GetServiceEndpoints(config.ServiceId, &endpoints)
		if err != nil {
			glog.Errorf("Error getting application endpoints for service %s: %s", config.ServiceId, err)
			return
		}

		for key, endpointList := range endpoints {

			glog.Infof("For %s, got %s", key, endpointList)
			if len(endpointList) <= 0 {
				continue
			}
			proxy := proxy.Proxy{}
			endpoint := endpointList[0]
			proxy.Name = fmt.Sprintf("%v", endpoint)
			proxy.Port = endpoint.ContainerPort
			proxy.Address = fmt.Sprintf("%s:%d", endpoint.HostIp, endpoint.HostPort)
			proxy.TCPMux = config.TCPMux.Enabled
			proxy.TCPMuxPort = config.TCPMux.Port
			proxy.UseTLS = config.TCPMux.UseTLS
			glog.Infof("Proxying %s", proxy)
			go proxy.ListenAndProxy()
		}
	}()

	if l, err := net.Listen("tcp", ":4321"); err == nil {
		l.Accept()
	}

	glog.Flush()
	os.Exit(0)
	return nil
}

// List the hosts associated with the control plane.
func (cli *ServicedCli) CmdHosts(args ...string) error {

	cmd := Subcmd("hosts", "[OPTIONS]", "List hosts")
	if err := cmd.Parse(args); err != nil {
		return err
	}

	client := getClient()

	var hosts map[string]*dao.Host
	request := dao.EntityRequest{}
	err := client.GetHosts(request, &hosts)
	if err != nil {
		glog.Fatalf("Could not get hosts %v", err)
	}
	hostsJson, err := json.MarshalIndent(hosts, " ", "  ")
	if err == nil {
		fmt.Printf("%s\n", hostsJson)
	}
	return err
}

// Add a host to the control plane given the host:port.
func (cli *ServicedCli) CmdAddHost(args ...string) error {

	cmd := Subcmd("add-host", "HOST:PORT RESOURCE_POOL", "Add host")
	if err := cmd.Parse(args); err != nil {
		return nil
	}

	if len(cmd.Args()) != 2 {
		cmd.Usage()
		return nil
	}

	client, err := clientlib.NewAgentClient(cmd.Arg(0))
	if err != nil {
		glog.Fatalf("Could not create connection to host %s: %v", args[0], err)
	}

	var remoteHost dao.Host
	err = client.GetInfo(0, &remoteHost)
	if err != nil {
		glog.Fatalf("Could not get remote host info: %v", err)
	}
	parts := strings.Split(cmd.Arg(0), ":")
	remoteHost.IpAddr = parts[0]
	remoteHost.PoolId = cmd.Arg(1)
	glog.Infof("Got host info: %v", remoteHost)

	controlPlane := getClient()
	var unused int

	err = controlPlane.AddHost(remoteHost, &unused)
	if err != nil {
		glog.Fatalf("Could not add host: %v", err)
	}
	fmt.Println(remoteHost.Id)
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
	glog.Infof("Host %s removed.", cmd.Arg(0))
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
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	request := dao.EntityRequest{}
	var pools map[string]*dao.ResourcePool
	err := controlPlane.GetResourcePools(request, &pools)
	if err != nil {
		glog.Fatalf("Could not get resource pools: %v", err)
	}
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
	var unused int
	err = controlPlane.AddResourcePool(*pool, &unused)
	if err != nil {
		glog.Fatalf("Could not add resource pool: %v", err)
	}
	fmt.Printf("%s\n", pool.Id)
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
	glog.Infof("Pool %s removed.\n", cmd.Arg(0))
	return err
}

// Print the list of available services.
func (cli *ServicedCli) CmdServices(args ...string) error {
	cmd := Subcmd("services", "[CMD]", "Show services")
	if err := cmd.Parse(args); err != nil {
		return nil
	}
	controlPlane := getClient()
	request := dao.EntityRequest{}
	var services []*dao.Service
	err := controlPlane.GetServices(request, &services)
	if err != nil {
		glog.Fatalf("Could not get services: %v", err)
	}
	servicesJson, err := json.MarshalIndent(services, " ", " ")
	if err != nil {
		glog.Fatalf("Problem marshaling services object: %s", err)
	}
	fmt.Printf("%s\n", servicesJson)
	return err
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
	if err != nil {
		glog.Infof("Could not get default gateway")
		return "127.0.0.1"
	}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "default" {
			return fields[2]
		}
	}
	return "127.0.0.1"
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
	glog.Infof("endpoints discovered: %v", flPortOpts)
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
	service.Endpoints = &endPoints
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
	glog.Infof("Calling AddService.\n")
	var unused int
	err = controlPlane.AddService(*service, &unused)
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
	err := controlPlane.RemoveService(cmd.Arg(0), &unused)
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
	glog.Infof("Sevice scheduled to start on host %s\n", hostId)
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
	glog.Infoln("Sevice scheduled to stop.")
	return err
}
