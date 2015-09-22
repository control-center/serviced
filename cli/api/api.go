// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"fmt"
	"os"
	"runtime/pprof"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/rpc/agent"
	"github.com/control-center/serviced/rpc/master"
	"github.com/control-center/serviced/volume"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

var options Options

// Options are the server options
type Options struct {
	Endpoint             string // the endpoint address to make RPC requests to
	UIPort               string
	NFSClient            string
	RPCPort              string
	Listen               string
	OutboundIP           string // outbound ip to listen on
	Master               bool
	DockerDNS            []string
	Agent                bool
	MuxPort              int
	TLS                  bool
	KeyPEMFile           string
	CertPEMFile          string
	VolumesPath          string
	IsvcsPath            string
	BackupsPath          string
	ResourcePath         string
	Zookeepers           []string
	ReportStats          bool
	HostStats            string
	StatsPeriod          int
	MCUsername           string
	MCPasswd             string
	Mount                []string
	ResourcePeriod       int
	FSType               volume.DriverType
	ESStartupTimeout     int
	HostAliases          []string
	Verbosity            int
	StaticIPs            []string
	DockerRegistry       string
	CPUProfile           string // write cpu profile to file
	MaxContainerAge      int    // max container age in seconds
	MaxDFSTimeout        int    // max timeout for snapshot
	VirtualAddressSubnet string
	MasterPoolID         string
	LogstashES           string //logstatsh elasticsearch host:port
	LogstashMaxDays      int    // Days to keep logstash indices
	LogstashMaxSize      int    // Max size of logstash data
	DebugPort            int    // Port to listen for profile clients
	AdminGroup           string // user group that can log in to control center
	MaxRPCClients        int    // the max number of rpc clients to an endpoint
	RPCDialTimeout       int
	SnapshotTTL          int               // hours to keep snapshots around, zero for infinity
	StorageArgs          []string          // command-line arguments for storage options
	StorageOptions       map[string]string // environment arguments for storage options
	ControllerBinary     string            // Path to the container controller binary
	StartISVCS           []string          // ISVCS to start when running as an agent
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

// GetOptionsEndpoint returns the serviced RPC endpoint from options
func GetOptionsRPCEndpoint() string {
	return options.Endpoint
}

// GetOptionsRPCPort returns the serviced RPC port from options
func GetOptionsRPCPort() string {
	return options.RPCPort
}

// GetOptionsMaster returns the master mode setting from options
func GetOptionsMaster() bool {
	return options.Master
}

// GetOptionsAgent returns the agent mode setting from options
func GetOptionsAgent() bool {
	return options.Agent
}

// GetOptionsMasterPoolID returns the master pool ID from options
func GetOptionsMasterPoolID() string {
	return options.MasterPoolID
}

// GetOptionsMaxRPCClients returns the max RPC clients setting from options
func GetOptionsMaxRPCClients() int {
	return options.MaxRPCClients
}

type api struct {
	master *master.Client
	agent  *agent.Client
	docker *dockerclient.Client
	dao    dao.ControlPlane // Deprecated
}

func New() API {
	// let lazy init populate each interface as necessary
	return NewAPI(nil, nil, nil, nil)
}

// New creates a new API type
func NewAPI(master *master.Client, agent *agent.Client, docker *dockerclient.Client, dao dao.ControlPlane) API {
	return &api{master: master, agent: agent, docker: docker, dao: dao}
}

// Starts the agent or master services on this host
func (a *api) StartServer() error {
	glog.Infof("StartServer: %v (%d)", options.StaticIPs, len(options.StaticIPs))

	if len(options.CPUProfile) > 0 {
		f, err := os.Create(options.CPUProfile)
		if err != nil {
			glog.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	d, err := newDaemon(options.Endpoint, options.StaticIPs, options.MasterPoolID)
	if err != nil {
		return err
	}
	return d.run()
}

// Opens a connection to the master if not already connected
func (a *api) connectMaster() (*master.Client, error) {
	if a.master == nil {
		var err error
		a.master, err = master.NewClient(options.Endpoint)
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
func (a *api) connectDocker() (*dockerclient.Client, error) {
	if a.docker == nil {
		const DockerEndpoint string = "unix:///var/run/docker.sock"
		var err error
		if a.docker, err = dockerclient.NewClient(DockerEndpoint); err != nil {
			return nil, fmt.Errorf("could not create a client to docker: %s", err)
		}
	}
	return a.docker, nil
}

// DEPRECATED: Opens a connection to the DAO if not already connected
func (a *api) connectDAO() (dao.ControlPlane, error) {
	if a.dao == nil {
		var err error
		a.dao, err = node.NewControlClient(options.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("could not create a client to the agent: %s", err)
		}
	}
	return a.dao, nil
}
