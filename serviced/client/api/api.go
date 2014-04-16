package api

import (
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/signal"
	"path"
	"strings"
	"syscall"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/dao/elasticsearch"
	"github.com/zenoss/serviced/isvcs"
	"github.com/zenoss/serviced/shell"
	"github.com/zenoss/serviced/stats"
	"github.com/zenoss/serviced/volume"
	"github.com/zenoss/serviced/web"
)

var minDockerVersion = version{0, 8, 1}

var options Options

type Options struct {
	Port             string
	UIPort           string
	Listen           string
	Master           bool
	DockerDNS        string
	Agent            bool
	MuxPort          int
	TLS              bool
	KeyPEMFile       string
	CertPEMFile      string
	VarPath          string
	ResourcePath     string
	Zookeepers       []string
	ReportStats      bool
	HostStats        string
	StatsPeriod      int
	MCUsername       string
	MCPasswd         string
	Mount            []string
	ResourcePeriod   int
	VFS              string
	ESStartupTimeout int
	HostAliases      []string
}

// Load options overwrites the existing options
func LoadOptions(ops Options) {
	options = ops

	// Check option boundaries
	if options.ESStartupTimeout < MIN_TIMEOUT {
		glog.V(0).Infof("overriding elastic search startup timeout with minimum %d", MIN_TIMEOUT)
		options.ESStartupTimeout = MIN_TIMEOUT
	}
}

// Opens a connection to the control plane
func connect() (dao.ControlPlane, error) {
	// setup the client
	c, err := serviced.NewControlClient(options.Port)
	if err != nil {
		return nil, fmt.Errorf("could not create a control plane client: %s", err)
	}
	return c, nil
}

type api struct {
}

// New creates a new API type
func New() API {
	return &api{}
}

// Starts the agent or master services on this host
func (a *api) StartServer() {
	l, err := net.Listen("tcp", options.Listen)
	if err != nil {
		glog.Fatalf("could not bind to port: %s. Is another instance running?", err)
	}

	isvcs.Init()
	isvcs.Mgr.SetVolumesDir(path.Join(options.VarPath, "isvcs"))

	dockerVersion, err := serviced.GetDockerVersion()
	if err != nil {
		glog.Fatalf("could not determine docker version: %s", err)
	}

	if minDockerVersion.Compare(dockerVersion.Client) < 0 {
		glog.Fatalf("serviced needs at least docker >= %s", minDockerVersion)
	}

	if _, ok := volume.Registered(options.VFS); !ok {
		glog.Fatalf("no driver registered for %s", options.VFS)
	}

	if options.Master {
		master, err := elasticsearch.NewControlSvc("localhost", 9200, options.Zookeepers, options.VarPath, options.VFS)
		if err != nil {
			glog.Fatalf("could not start ControlPlane service: %s", err)
		}

		// Register the API
		glog.V(0).Infof("Registering the ControlPlane service")
		rpc.RegisterName("LoadBalancer", master)
		rpc.RegisterName("ControlPlane", master)

		// TODO: make bind port for web server optional?
		aliases := strings.Join(options.HostAliases, ",")

		cpserver := web.NewServiceConfig(":8787", options.Port, options.Zookeepers, options.ReportStats, aliases)
		go cpserver.ServeUI()
		go cpserver.Serve()
	}
	if options.Agent {
		mux := serviced.TCPMux{
			CertPEMFile: options.CertPEMFile,
			KeyPEMFile:  options.KeyPEMFile,
			Enabled:     true,
			Port:        options.MuxPort,
			UseTLS:      options.TLS,
		}

		dnsList := strings.Split(options.DockerDNS, ",")
		agent, err := serviced.NewHostAgent(options.Port, options.UIPort, dnsList, options.VarPath, options.Mount, options.VFS, options.Zookeepers, mux)
		if err != nil {
			glog.Fatalf("could not start ControlPlane agent")
		}

		// Register the API
		glog.V(0).Infof("Registering the ControlPlaneAgent service")
		rpc.RegisterName("ControlPlaneAgent", agent)

		go func() {
			signalChan := make(chan os.Signal, 10)
			signal.Notify(signalChan, syscall.SIGINT, syscall.SIGTERM)
			<-signalChan
			glog.V(0).Info("Shutting down due to interrupt")
			err = agent.Shutdown()
			if err != nil {
				glog.V(1).Infof("Agent shutdown with error: %v", err)
			}
			isvcs.Mgr.Stop()
			os.Exit(0)
		}()

		// TODO: integrate this server into the rps server, or something.
		// Currently its only use is for command execution.
		go func() {
			sio := shell.NewProcessExecutorServer(options.Port)
			http.ListenAndServe(":50000", sio)
		}()
	}

	rpc.HandleHTTP()

	if options.ReportStats {
		path := fmt.Sprintf("http://%s/api/metrics/store", options.HostStats)
		duration := time.Duration(options.StatsPeriod) * time.Second
		glog.V(1).Infof("Starting container statistics reporter")
		reporter := stats.NewStatsReporter(path, duration)
		defer reporter.Close()
	}

	glog.V(0).Infof("Listening on %s", l.Addr().String())
	http.Serve(l, nil) // Start the server
}
