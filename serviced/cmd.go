/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package main

// This is the main entry point for the application. Here we parse command line
// flags and either start a service or execute command line functions.

import (
	"flag"
	agent "github.com/zenoss/serviced/agent"
	svc "github.com/zenoss/serviced/svc"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os"
)

// Store the command line options
var options struct {
	port              string
	listen            string
	master            bool
	agent             bool
	connection_string string
}

// Setup flag options (static block)
func init() {
	flag.StringVar(&options.port, "port", "127.0.0.1:4979", "port for remote serviced (example.com:8080)")
	flag.StringVar(&options.listen, "listen", ":4979", "port for local serviced (example.com:8080)")
	flag.BoolVar(&options.master, "master", false, "run in master mode, ie the control plane service")
	flag.BoolVar(&options.agent, "agent", false, "run in agent mode, ie a host in a resource pool")
	conStr := os.Getenv("CP_PROD_DB")
	if len(conStr) == 0 {
		conStr = "mysql://root@127.0.0.1:3306/cp"
	} else {
		log.Printf("Using connection string from env var CP_PROD_DB")
	}
	flag.StringVar(&options.connection_string, "connection-string", conStr, "Database connection uri")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

// Start the agent or master services on this host.
func startServer() {
	if options.master {
		master, err := svc.NewControlSvc(options.connection_string)
		if err != nil {
			log.Fatalf("Could not start ControlPlane service: %v", err)
		}
		// register the API
		log.Printf("registering ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)
	}
	if options.agent {
		agent, err := agent.NewHostAgent(options.port)
		if err != nil {
			log.Fatalf("Could not start ControlPlane agent: %v", err)
		}
		// register the API
		log.Printf("registering ControlPlaneAgent service")
		rpc.RegisterName("ControlPlaneAgent", agent)
	}
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", options.listen)
	if err != nil {
		log.Fatalf("Could not bind to port %v", err)
	}

	log.Printf("Listening on %s", l.Addr().String())
	http.Serve(l, nil) // start the server
}

// main entry point of the product
func main() {

	// parse the command line flags
	flag.Parse()

	// are we in server mode
	if (options.master || options.agent) && len(flag.Args()) == 0 {
		startServer()
	} else {
		// we are in command line mode
		if len(flag.Args()) == 0 {
			// no arguments were give, show help
			cli := ServicedCli{}
			cli.CmdHelp(flag.Args()...)
			flag.Usage()
		} else {
			ParseCommands(flag.Args()...)
		}
	}
}
