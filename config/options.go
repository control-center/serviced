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
	"strconv"

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

// Options are the server options.
// Default values and environment variable overrides defined by api.GetDefaultOptions()
// Command line overrides defined by cmd.New()
type Options struct {
	GCloud                     bool   // Running in gcloud
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
	MuxDisableTLS              string //  Disable TLS for MUX connections, string val of bool
	KeyPEMFile                 string
	CertPEMFile                string
	HomePath                   string // serviced's root directory; e.g. /opt/serviced
	VolumesPath                string
	EtcPath                    string
	IsvcsPath                  string
	BackupsPath                string
	ResourcePath               string
	LogPath                    string // Serviced logs directory
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
	LogstashStdout             bool     // Write Logstash logs to stdout
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
	IsvcsENV                   []string          // Isvcs env variables
	IsvcsZKID                  int               // Zookeeper server id when running as a quorum
	IsvcsZKQuorum              []string          // Members of the zookeeper quorum
	IsvcsZKUsername            string            // Zookeeper username required for quorum authentication
	IsvcsZKPasswd              string            // Zookeeper password required for quorum authentication
	ZkAclUser                  string            // Zookeeper username required for digest ACL scheme
	ZkAclPasswd                string            // Zookeeper password required for digest ACL scheme
	TLSCiphers                 []string          // List of tls ciphers supported for http
	TLSMinVersion              string            // Minimum TLS version supported for http
	DockerLogDriver            string            // Which log driver to use with containers
	DockerLogConfigList        []string          // List of comma-separated key=value options for docker logging
	AllowLoopBack              string            // Allow loop back devices for DM storage, string val of bool
	UIPollFrequency            int               // frequency in seconds that UI should poll for service changes
	StorageStatsUpdateInterval int               // frequency in seconds that low-level devicemapper storage stats should be refreshed
	SnapshotSpacePercent       int               // Percent of tenant volume size that is assumed to be needed to create a snapshot
	ZKSessionTimeout           int               // The session timeout of a zookeeper client connection.
	ZKConnectTimeout           int               // The network connect timeout, in seconds, for a zookeeper client connection.
	ZKPerHostConnectDelay      int               // The delay, in seconds, between connection attempts to other zookeeper servers.
	ZKReconnectStartDelay      int               // The initial delay, in seconds, before attempting to reconnect after none of the zookeepers are reachable
	ZKReconnectMaxDelay        int               // The maximum delay, in seconds, before attempting to reconnect after none of the zookeepers are reachable
	TokenExpiration            int               // The time in seconds before an authentication token expires
	ConntrackFlush             string            // Whether to flush the conntrack table when a service with an assigned IP is started
	LogConfigFilename          string            // Path to the logri configuration
	StorageReportInterval      int               // frequency in seconds to report storage stats to opentsdb
	ServiceRunLevelTimeout     int               // The time in seconds serviced will wait for a batch of services to stop/start before moving to services with the next run level
	StorageMetricMonitorWindow int               // The amount of time in seconds for which serviced will consider storage availability metrics in order to predict future availability
	StorageLookaheadPeriod     int               // The amount of time in the future in seconds serviced should predict storage availability for the purposes of emergency shutdown
	StorageMinimumFreeSpace    string            // The amount of space the emergency shutdown algorithm should reserve when deciding to shut down
	BackupEstimatedCompression float64           // Best guess for tgz compression ratio (uncompressed size / compressed size) used to determine whether sufficient disk space is available for taking a backup
	BackupMinOverhead          string            // Warn user if estimated backup size would leave less than this amount of space free
	StartZK                    bool              // Should ZooKeeper ISVC be started
	StartAPIKeyProxy           bool              // Should API Key Proxy ISVC be started
	BigTableMetrics            bool              // Should serviced metrics be stored in gcp bigtable
	Auth0Domain                string            // Domain configured for tenant in Auth0. Ref: https://auth0.com/docs/getting-started/the-basics#domain
	Auth0Audience              string            // Audience configured for application (?) in Auth0
	Auth0Group                 []string          // Group membership(s) required in Auth0 token for login, comma separated list
	Auth0ClientID              string            // ClientID of Auth0 Application
	Auth0Scope                 string            // Auth0 Scope for request.
	KeyProxyJsonServer         string            // Address of api-key-server endpoint for getting CC Access tokens
	KeyProxyListenPort         string            // Port where api-key-proxy will listen
	ESRequestTimeout           int               // The http request connect timeout, in seconds, for an elasticsearch client connection.
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

	if options.ZKReconnectStartDelay < 1 {
		log.WithFields(logrus.Fields{
			"reconnectstartdelay": options.ZKReconnectStartDelay,
		}).Debug("ZK_RECONNECT_START_DELAY too low; Resetting to 1 second")
		options.ZKReconnectStartDelay = 1
	}
	if options.ZKReconnectMaxDelay < 1 {
		log.WithFields(logrus.Fields{
			"reconnectmaxdelay": options.ZKReconnectMaxDelay,
		}).Debug("ZK_RECONNECT_MAX_DELAY too low; Resetting to 1 second")
		options.ZKReconnectMaxDelay = 1
	}
	if options.ZKReconnectStartDelay > options.ZKReconnectMaxDelay {
		log.WithFields(logrus.Fields{
			"reconnectstartdelay": options.ZKReconnectStartDelay,
			"reconnectmaxdelay":   options.ZKReconnectMaxDelay,
		}).Debug("ZK_RECONNECT_START_DELAY too large; Resetting to ZK_RECONNECT_MAX_DELAY")
		options.ZKReconnectStartDelay = options.ZKReconnectMaxDelay
	}
}

func MuxTLSIsEnabled() bool {
	disabled, _ := strconv.ParseBool(options.MuxDisableTLS)
	return !disabled && options.MuxPort > 0
}
