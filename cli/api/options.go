// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package api

import (
	"github.com/control-center/serviced/volume"
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
	LogstashURL          string
	DebugPort            int    // Port to listen for profile clients
	AdminGroup           string // user group that can log in to control center
	MaxRPCClients        int    // the max number of rpc clients to an endpoint
	RPCDialTimeout       int
	RPCCertVerify        string            //  server certificate verify for rpc connections, string val of bool
	RPCDisableTLS        string            //  Disable TLS for RPC connections, string val of bool
	SnapshotTTL          int               // hours to keep snapshots around, zero for infinity
	StorageArgs          []string          // command-line arguments for storage options
	StorageOptions       map[string]string // environment arguments for storage options
	ControllerBinary     string            // Path to the container controller binary
	StartISVCS           []string          // ISVCS to start when running as an agent
	IsvcsZKID            int               // Zookeeper server id when running as a quorum
	IsvcsZKQuorum        []string          // Members of the zookeeper quorum
	TLSCiphers           []string          // List of tls ciphers supported
	TLSMinVersion        string            // Minimum TLS version supported
	DockerLogDriver      string            // Which log driver to use with containers
	DockerLogConfigList  []string          // List of comma-separated key=value options for docker logging
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

// GetOptionsRPCEndpoint returns the serviced RPC endpoint from options
func GetOptionsRPCEndpoint() string {
	return options.Endpoint
}

// SetOptionsRPCEndpoint sets the serviced RPC endpoint in the options
func SetOptionsRPCEndpoint(endpoint string) {
	options.Endpoint = endpoint
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

