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

//svc "github.com/zenoss/serviced/svc"
import (
	agent "github.com/zenoss/serviced/agent"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/proxy"
	"github.com/zenoss/serviced/web"

	"flag"
	"fmt"
	"github.com/zenoss/glog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"time"
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
	repstats          bool
	statshost         string
	statsperiod       int
	mcusername        string
	mcpasswd          string
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
	flag.BoolVar(&options.repstats, "reportstats", false, "report container statistics")
	flag.StringVar(&options.statshost, "statshost", "127.0.0.1:8443", "host:port for container statistics")
	flag.IntVar(&options.statsperiod, "statsperiod", 5, "Period (minutes) for container statistics reporting")
	flag.StringVar(&options.mcusername, "mcusername", "scott", "Username for the Zenoss metric consumer")
	flag.StringVar(&options.mcpasswd, "mcpasswd", "tiger", "Password for the Zenoss metric consumer")

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
		var master dao.ControlPlane
		var err error
		master, err = elasticsearch.NewControlSvc("localhost", 9200, options.zookeepers)

		if err != nil {
			glog.Fatalf("Could not start ControlPlane service: %v", err)
		}
		// register the API
		glog.Infoln("registering ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)

		// TODO: Make bind port for web server optional?
		cpserver := web.NewServiceConfig(":8787", options.port, options.zookeepers)
		go cpserver.Serve()
	}
	if options.agent {
		mux := proxy.TCPMux{}

		mux.CertPEMFile = options.certPEMFile
		mux.KeyPEMFile = options.keyPEMFile
		mux.Enabled = true
		mux.Port = options.muxPort
		mux.UseTLS = options.tls

		agent, err := agent.NewHostAgent(options.port, mux, options.zookeepers)
		if err != nil {
			glog.Fatalf("Could not start ControlPlane agent: %v", err)
		}
		// register the API
		glog.Infoln("registering ControlPlaneAgent service")
		rpc.RegisterName("ControlPlaneAgent", agent)
	}
	rpc.HandleHTTP()

	if options.repstats {
		statsdest := fmt.Sprintf("https://%s/api/metrics/store", options.statshost)
		sr := StatsReporter{statsdest, options.mcusername, options.mcpasswd}

		glog.V(1).Infoln("Staring containter statistics reporter")
		statsduration := time.Duration(options.statsperiod) * time.Minute
		go sr.Report(statsduration)
	}

	l, err := net.Listen("tcp", options.listen)
	if err != nil {
		glog.Warningf("Could not bind to port %v", err)
		time.Sleep(time.Second * 1000)
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
