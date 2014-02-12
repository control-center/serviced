// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package main

// This is the main entry point for the application. Here we parse command line
// flags and either start a service or execute command line functions.

//svc "github.com/zenoss/serviced/svc"
import (
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/shell"
	"github.com/zenoss/serviced/volume"
	_ "github.com/zenoss/serviced/volume/btrfs"
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/web"

	"flag"
	"fmt"
	"github.com/zenoss/glog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"os/user"
	"path"
	"time"
)

// Store the command line options
var options struct {
	port           string
	listen         string
	master         bool
	agent          bool
	muxPort        int
	tls            bool
	keyPEMFile     string
	certPEMFile    string
	varPath        string // Directory to store data, eg isvcs & service volumes
	resourcePath   string
	zookeepers     ListOpts
	repstats       bool
	statshost      string
	statsperiod    int
	mcusername     string
	mcpasswd       string
	mount          ListOpts
	resourceperiod int
	vfs            string
}

var agentIP string

// Setup flag options (static block)
func init() {
	var err error
	agentIP, err = serviced.GetIpAddress()
	if err != nil {
		panic(err)
	}

	flag.StringVar(&options.port, "port", agentIP+":4979", "port for remote serviced (example.com:8080)")
	flag.StringVar(&options.listen, "listen", ":4979", "port for local serviced (example.com:8080)")
	flag.BoolVar(&options.master, "master", false, "run in master mode, ie the control plane service")
	flag.BoolVar(&options.agent, "agent", false, "run in agent mode, ie a host in a resource pool")
	flag.IntVar(&options.muxPort, "muxport", 22250, "multiplexing port to use")
	flag.BoolVar(&options.tls, "tls", true, "enable TLS")

	varPathDefault := path.Join(os.TempDir(), "serviced")
	if len(os.Getenv("SERVICED_HOME")) > 0 {
		varPathDefault = path.Join(os.Getenv("SERVICED_HOME"), "var")
	} else {
		if user, err := user.Current(); err == nil {
			varPathDefault = path.Join(os.TempDir(), "serviced-"+user.Username, "var")
		}
	}
	flag.StringVar(&options.varPath, "varPath", varPathDefault, "path to store serviced data")

	flag.StringVar(&options.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	flag.StringVar(&options.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	options.zookeepers = make(ListOpts, 0)
	flag.Var(&options.zookeepers, "zk", "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181 )")
	flag.BoolVar(&options.repstats, "reportstats", false, "report container statistics")
	flag.StringVar(&options.statshost, "statshost", "127.0.0.1:8443", "host:port for container statistics")
	flag.IntVar(&options.statsperiod, "statsperiod", 5, "Period (minutes) for container statistics reporting")
	flag.StringVar(&options.mcusername, "mcusername", "scott", "Username for the Zenoss metric consumer")
	flag.StringVar(&options.mcpasswd, "mcpasswd", "tiger", "Password for the Zenoss metric consumer")
	options.mount = make(ListOpts, 0)
	flag.Var(&options.mount, "mount", "bind mount: container_image:host_path:container_path (e.g. -mount zenoss/zenoss5x:/home/zenoss/zenhome/zenoss/Products/:/opt/zenoss/Products/)")
	flag.StringVar(&options.vfs, "vfs", "rsync", "file system for container volumes")

	flag.Usage = func() {
		flag.PrintDefaults()
	}
}

func compareVersion(a, b []int) int {
	astr := ""
	for _, s := range a {
		astr += fmt.Sprintf("%12d", s)
	}
	bstr := ""
	for _, s := range b {
		bstr += fmt.Sprintf("%12d", s)
	}
	if astr > bstr {
		return -1
	}
	if astr < bstr {
		return 1
	}
	return 0
}

// Start the agent or master services on this host.
func startServer() {
	l, err := net.Listen("tcp", options.listen)
	if err != nil {
		glog.Fatalf("Could not bind to port %v. Is another instance running", err)
	}

	isvcs.Init()
	isvcs.Mgr.SetVolumesDir(options.varPath + "/isvcs")

	dockerVersion, err := serviced.GetDockerVersion()
	if err != nil {
		glog.Fatalf("Could not determine docker version: %s", err)
	}

	atLeast := []int{0, 7, 5}
	atMost := []int{0, 7, 6}
	if compareVersion(atLeast, dockerVersion.Client) < 0 || compareVersion(atMost, dockerVersion.Client) > 0 {
		glog.Fatal("serviced needs at least docker >= 0.7.5 or <= 0.7.6")
	}

	if _, ok := volume.Registered(options.vfs); !ok {
		glog.Fatalf("no driver registered for %s", options.vfs)
	}

	if options.master {
		var master dao.ControlPlane
		var err error
		master, err = elasticsearch.NewControlSvc("localhost", 9200, options.zookeepers, options.varPath, options.vfs)

		if err != nil {
			glog.Fatalf("Could not start ControlPlane service: %v", err)
		}
		// register the API
		glog.V(0).Infoln("registering ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)

		// TODO: Make bind port for web server optional?
		cpserver := web.NewServiceConfig(":8787", options.port, options.zookeepers, options.repstats)
		go cpserver.Serve()
	}
	if options.agent {
		mux := serviced.TCPMux{}

		mux.CertPEMFile = options.certPEMFile
		mux.KeyPEMFile = options.keyPEMFile
		mux.Enabled = true
		mux.Port = options.muxPort
		mux.UseTLS = options.tls

		agent, err := serviced.NewHostAgent(options.port, options.varPath, options.mount, options.vfs, options.zookeepers, mux)
		if err != nil {
			glog.Fatalf("Could not start ControlPlane agent: %v", err)
		}
		// register the API
		glog.V(0).Infoln("registering ControlPlaneAgent service")
		rpc.RegisterName("ControlPlaneAgent", agent)

		go func() {
			signalChan := make(chan os.Signal, 10)
			signal.Notify(signalChan, os.Interrupt)
			<-signalChan
			glog.V(0).Info("Shutting down due to interrupt")
			err = agent.Shutdown()
			if err != nil {
				glog.V(1).Infof("Agent shutdown with error: %v", err)
			}
			isvcs.Mgr.Stop()
			os.Exit(0)
		}()

		// TODO: Integrate this server into the rpc server, or something.
		// Currently its only use is for command execution.
		go func() {
			sio := shell.NewProcessExecutorServer(options.port)
			sio.Handle("/", http.FileServer(http.Dir("/home/zenoss/europa/src/golang/src/github.com/zenoss/serviced/serviced/www/")))
			http.ListenAndServe(":50000", sio)
		}()
	}

	rpc.HandleHTTP()

	if options.repstats {
		statsdest := fmt.Sprintf("http://%s/api/metrics/store", options.statshost)
		sr := StatsReporter{statsdest, options.mcusername, options.mcpasswd}

		glog.V(1).Infoln("Staring container statistics reporter")
		statsduration := time.Duration(options.statsperiod) * time.Minute
		go sr.Report(statsduration)
	}

	glog.V(0).Infof("Listening on %s", l.Addr().String())
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
