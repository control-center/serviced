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

// This package exists merely to avoid circular imports. It sets up the basic
// global options for our servers and the CLI. Other packages (most notable,
// cli/api) will modify these options as they see fit.  This is merely
// a stopgap, which is why it's so poorly organized.
// TODO: Use Viper or comparable configuration registry to implement global
// configuration properly
package config

import (
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/volume"
)

const (
	minTimeout = 30
)

var (
	options Options
	log     = logging.PackageLogger()
)

// Options are the server options
type Options struct {
	Endpoint                   string // the endpoint address to make RPC requests to
	UIPort                     string
	NFSClient                  string
	RPCPort                    string
	Listen                     string
	OutboundIP                 string // outbound ip to listen on
	Master                     bool
	DockerDNS                  []string
	Agent                      bool
	MuxPort                    int
	MuxDisableTLS              string            //  Disable TLS for MUX connections, string val of bool
	KeyPEMFile                 string
	CertPEMFile                string
	VolumesPath                string
	EtcPath                    string
	IsvcsPath                  string
	BackupsPath                string
	ResourcePath               string
	Zookeepers                 []string
	ReportStats                bool
	HostStats                  string
	StatsPeriod                int
	SvcStatsCacheTimeout       int
	MCUsername                 string
	MCPasswd                   string
	Mount                      []string
	ResourcePeriod             int
	FSType                     volume.DriverType
	ESStartupTimeout           int
	HostAliases                []string
	Verbosity                  int
	StaticIPs                  []string
	DockerRegistry             string
	CPUProfile                 string // write cpu profile to file
	MaxContainerAge            int    // max container age in seconds
	MaxDFSTimeout              int    // max timeout for snapshot
	VirtualAddressSubnet       string
	MasterPoolID               string
	LogstashES                 string //logstash elasticsearch host:port
	LogstashMaxDays            int    // Days to keep logstash indices
	LogstashMaxSize            int    // Max size of logstash data
	LogstashCycleTime          int    // Logstash purging cycle time in hours
	LogstashURL                string
	DebugPort                  int      // Port to listen for profile clients
	AdminGroup                 string   // user group that can log in to control center
	MaxRPCClients              int      // the max number of rpc clients to an endpoint
	MUXTLSCiphers              []string // List of tls ciphers supported for mux
	MUXTLSMinVersion           string   // Minimum TLS version supported for mux
	RPCDialTimeout             int
	RPCCertVerify              string            //  server certificate verify for rpc connections, string val of bool
	RPCDisableTLS              string            //  Disable TLS for RPC connections, string val of bool
	RPCTLSCiphers              []string          // List of tls ciphers supported for rpc
	RPCTLSMinVersion           string            // Minimum TLS version supported for rpc
	SnapshotTTL                int               // hours to keep snapshots around, zero for infinity
	StorageArgs                []string          // command-line arguments for storage options
	StorageOptions             map[string]string // environment arguments for storage options
	ControllerBinary           string            // Path to the container controller binary
	StartISVCS                 []string          // ISVCS to start when running as an agent
	IsvcsZKID                  int               // Zookeeper server id when running as a quorum
	IsvcsZKQuorum              []string          // Members of the zookeeper quorum
	TLSCiphers                 []string          // List of tls ciphers supported for http
	TLSMinVersion              string            // Minimum TLS version supported for http
	DockerLogDriver            string            // Which log driver to use with containers
	DockerLogConfigList        []string          // List of comma-separated key=value options for docker logging
	AllowLoopBack              string            // Allow loop back devices for DM storage, string val of bool
	UIPollFrequency            int               // frequency in seconds that UI should poll for service changes
	StorageStatsUpdateInterval int               // frequency in seconds that low-level devicemapper storage stats should be refreshed
	SnapshotSpacePercent       int               // Percent of tenant volume size that is assumed to be needed to create a snapshot
	ZKSessionTimeout           int               // The session timeout of a zookeeper client connection.
	TokenExpiration            int               // The time in seconds before an authentication token expires
	LogConfigFilename          string            // Path to the logri configuration
}

// GetOptions returns a COPY of the global options struct
func GetOptions() Options {
	return options
}

// LoadOptions overwrites the existing server options
func LoadOptions(ops Options) {
	options = ops

	// Check option boundaries
	if options.ESStartupTimeout < minTimeout {
		log.WithFields(logrus.Fields{
			"mintimeout": minTimeout,
		}).Debug("Overriding Elastic startup timeout")
		options.ESStartupTimeout = minTimeout
	}
}
