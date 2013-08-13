/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package main

import (
	"flag"
	"github.com/zenoss/serviced"
	"log"
	"net"
	"net/http"
	"net/rpc"
)

var options struct {
	port              string
	listen            string
	master            bool
	agent             bool
	connection_string string
}

func init() {
	flag.StringVar(&options.port, "port", "127.0.0.1:4979", "port for remote serviced (example.com:8080)")
	flag.StringVar(&options.listen, "listen", ":4979", "port for local serviced (example.com:8080)")
	flag.BoolVar(&options.master, "master", false, "run in master mode, ie the control plane service")
	flag.BoolVar(&options.agent, "agent", false, "run in agent mode, ie a host in a resource pool")
	flag.StringVar(&options.connection_string, "connection-string", "cp/root/", "Database connection string (eg \"dbname/user/password\")")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

func handleCmd() {

}

func startServer() {
	if options.master {
		master, err := serviced.NewControlSvc(options.connection_string)
		if err != nil {
			log.Fatalf("Could not start ControlPlane service: %v", err)
		}
		// register the API
		log.Printf("registering ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)
	}
	if options.agent {
		agent, err := serviced.NewHostAgent(options.port)
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

func main() {
	flag.Parse()

	if (options.master || options.agent) && len(flag.Args()) == 0 {
		startServer()
	}
	if len(flag.Args()) == 0 {
		cli := ServicedCli{}
		cli.CmdHelp(flag.Args()...)
		flag.Usage()
	} else {
		ParseCommands(flag.Args()...)
	}
}
