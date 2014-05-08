package api

import (
	"fmt"

	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/rpc/agent"
	"github.com/zenoss/serviced/rpc/master"
)

var options Options

// Options are the server options
type Options struct {
	Port             string
	UIPort           string
	Listen           string
	Master           bool
	DockerDNS        []string
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
	Verbosity        int
	StaticIPs        []string
}

// LoadOptions overwrites the existing server options
func LoadOptions(ops Options) {
	options = ops

	// Set verbosity
	glog.SetVerbosity(options.Verbosity)

	// Check option boundaries
	if options.ESStartupTimeout < minTimeout {
		glog.V(0).Infof("overriding elastic search startup timeout with minimum %d", minTimeout)
		options.ESStartupTimeout = minTimeout
	}
}

type api struct {
	master *master.Client
	agent  *agent.Client
	docker *docker.Client
	dao    dao.ControlPlane // Deprecated
}

// New creates a new API type
func New() API {
	return &api{}
}

// Starts the agent or master services on this host
func (a *api) StartServer() error {
	glog.Infof("StartServer: %v (%d)", options.StaticIPs, len(options.StaticIPs))
	d, err := newDaemon(options.StaticIPs)
	if err != nil {
		return err
	}
	return d.run()
}

// Opens a connection to the master if not already connected
func (a *api) connectMaster() (*master.Client, error) {
	if a.master == nil {
		var err error
		a.master, err = master.NewClient(options.Port)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the master: %s", err)
		}
	}
	return a.master, nil
}

// Opens a connection to the agent if not already connected
func (a *api) connectAgent(address string) (*agent.Client, error) {
	if a.agent == nil {
		var err error
		a.agent, err = agent.NewClient(address)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the agent: %s", err)
		}
	}
	return a.agent, nil
}

// Opens a connection to docker if not already connected
func (a *api) connectDocker() (*docker.Client, error) {
	if a.docker == nil {
		const DOCKER_ENDPOINT string = "unix:///var/run/docker.sock"
		var err error
		if a.docker, err = docker.NewClient(DOCKER_ENDPOINT); err != nil {
			return nil, fmt.Errorf("could not create a client to docker: %s", err)
		}
	}
	return a.docker, nil
}

// DEPRECATED: Opens a connection to the DAO if not already connected
func (a *api) connectDAO() (dao.ControlPlane, error) {
	if a.dao == nil {
		var err error
		a.dao, err = serviced.NewControlClient(options.Port)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the agent: %s", err)
		}
	}
	return a.dao, nil
}
