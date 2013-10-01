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
	agent "github.com/zenoss/serviced/agent"
	"github.com/zenoss/serviced/proxy"
	svc "github.com/zenoss/serviced/svc"

	"flag"
	"github.com/zenoss/glog"
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
	muxPort           int
	tls               bool
	keyPEMFile        string
	certPEMFile       string
	zookeepers        ListOpts
}

// Setup flag options (static block)
func init() {
	flag.StringVar(&options.port, "port", "127.0.0.1:4979", "port for remote serviced (example.com:8080)")
	flag.StringVar(&options.listen, "listen", ":4979", "port for local serviced (example.com:8080)")
	flag.BoolVar(&options.master, "master", false, "run in master mode, ie the control plane service")
	flag.BoolVar(&options.agent, "agent", false, "run in agent mode, ie a host in a resource pool")
	flag.IntVar(&options.muxPort, "muxport", 22250, "multiplexing port to use")
	flag.BoolVar(&options.tls, "tls", true, "enable TLS")
	flag.StringVar(&options.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	flag.StringVar(&options.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	options.zookeepers = make(ListOpts, 0)
	flag.Var(&options.zookeepers, "zk", "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181 )")

	conStr := os.Getenv("CP_PROD_DB")
	if len(conStr) == 0 {
		conStr = "mysql://root@127.0.0.1:3306/cp"
	} else {
		glog.Infoln("Using connection string from env var CP_PROD_DB")
	}
	flag.StringVar(&options.connection_string, "connection-string", conStr, "Database connection uri")
	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

// Start the agent or master services on this host.
func startServer() {
	if options.master {
		master, err := svc.NewControlSvc(options.connection_string, options.zookeepers)
		if err != nil {
			glog.Fatalf("Could not start ControlPlane service: %v", err)
		}
		// register the API
		glog.Infoln("registering ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)
	}
	if options.agent {
		mux := proxy.TCPMux{}

		mux.CertPEMFile = options.certPEMFile
		mux.KeyPEMFile = options.keyPEMFile
		mux.Enabled = true
		mux.Port = options.muxPort
		mux.UseTLS = options.tls

		agent, err := agent.NewHostAgent(options.port, mux)
		if err != nil {
			glog.Fatalf("Could not start ControlPlane agent: %v", err)
		}
		// register the API
		glog.Infoln("registering ControlPlaneAgent service")
		rpc.RegisterName("ControlPlaneAgent", agent)
	}
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", options.listen)
	if err != nil {
		glog.Fatalf("Could not bind to port %v", err)
	}

	glog.Infof("Listening on %s", l.Addr().String())
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
	glog.Flush()
}
