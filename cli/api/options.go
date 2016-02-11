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
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

const DefaultRPCPort = 4979

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

func ValidateOptions(opts Options) error {
	var err error
	rpcutils.RPCCertVerify, err = strconv.ParseBool(opts.RPCCertVerify)
	if err != nil {
		return fmt.Errorf("error parsing rpc-cert-verify value %v", err)
	}
	rpcutils.RPCDisableTLS, err = strconv.ParseBool(opts.RPCDisableTLS)
	if err != nil {
		return fmt.Errorf("error parsing rpc-disable-tls value %v", err)
	}

	if err := validation.ValidUIAddress(opts.UIPort); err != nil {
		fmt.Fprintf(os.Stderr, "error validating UI port: %s\n", err)
		return fmt.Errorf("error validating UI port: %s", err)
	}

	if err := validation.IsSubnet16(opts.VirtualAddressSubnet); err != nil {
		fmt.Fprintf(os.Stderr, "error validating virtual-address-subnet: %s\n", err)
		return fmt.Errorf("error validating virtual-address-subnet: %s", err)
	}
	return nil
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

func GetDefaultOptions(config utils.ConfigReader) Options {
	masterIP := config.StringVal("MASTER_IP", "127.0.0.1")

	options := Options{
		UIPort:               config.StringVal("UI_PORT", ":443"),
		NFSClient:            config.StringVal("NFS_CLIENT", "1"),
		RPCPort:              config.StringVal("RPC_PORT", fmt.Sprintf("%d", DefaultRPCPort)),
		OutboundIP:           config.StringVal("OUTBOUND_IP", ""),
		DockerDNS:            config.StringSlice("DOCKER_DNS", []string{}),
		Master:               config.BoolVal("MASTER", false),
		Agent:                config.BoolVal("AGENT", false),
		MuxPort:              config.IntVal("MUX_PORT", 22250),
		KeyPEMFile:           config.StringVal("KEY_FILE", ""),
		CertPEMFile:          config.StringVal("CERT_FILE", ""),
		Zookeepers:           config.StringSlice("ZK", []string{}),
		HostStats:            config.StringVal("STATS_PORT", fmt.Sprintf("%s:8443", masterIP)),
		StatsPeriod:          config.IntVal("STATS_PERIOD", 10),
		MCUsername:           "scott",
		MCPasswd:             "tiger",
		FSType:               volume.DriverType(config.StringVal("FS_TYPE", "devicemapper")),
		ESStartupTimeout:     getDefaultESStartupTimeout(config.IntVal("ES_STARTUP_TIMEOUT", isvcs.DEFAULT_ES_STARTUP_TIMEOUT_SECONDS)),
		HostAliases:          config.StringSlice("VHOST_ALIASES", []string{}),
		Verbosity:            config.IntVal("LOG_LEVEL", 0),
		StaticIPs:            config.StringSlice("STATIC_IPS", []string{}),
		DockerRegistry:       config.StringVal("DOCKER_REGISTRY", getDefaultDockerRegistry()),
		MaxContainerAge:      config.IntVal("MAX_CONTAINER_AGE", 60*60*24),
		MaxDFSTimeout:        config.IntVal("MAX_DFS_TIMEOUT", 60*5),
		VirtualAddressSubnet: config.StringVal("VIRTUAL_ADDRESS_SUBNET", "10.3"),
		MasterPoolID:         config.StringVal("MASTER_POOLID", "default"),
		LogstashES:           config.StringVal("LOGSTASH_ES", fmt.Sprintf("%s:9100", masterIP)),
		LogstashURL:          config.StringVal("LOG_ADDRESS", fmt.Sprintf("%s:5042", masterIP)),
		LogstashMaxDays:      config.IntVal("LOGSTASH_MAX_DAYS", 14),
		LogstashMaxSize:      config.IntVal("LOGSTASH_MAX_SIZE", 10),
		DebugPort:            config.IntVal("DEBUG_PORT", 6006),
		AdminGroup:           config.StringVal("ADMIN_GROUP", getDefaultAdminGroup()),
		MaxRPCClients:        config.IntVal("MAX_RPC_CLIENTS", 3),
		RPCDialTimeout:       config.IntVal("RPC_DIAL_TIMEOUT", 30),
		RPCCertVerify:        strconv.FormatBool(config.BoolVal("RPC_CERT_VERIFY", false)),
		RPCDisableTLS:        strconv.FormatBool(config.BoolVal("RPC_DISABLE_TLS", false)),
		SnapshotTTL:          config.IntVal("SNAPSHOT_TTL", 12),
		StartISVCS:           config.StringSlice("ISVCS_START", []string{}),
		IsvcsZKID:            config.IntVal("ISVCS_ZOOKEEPER_ID", 0),
		IsvcsZKQuorum:        config.StringSlice("ISVCS_ZOOKEEPER_QUORUM", []string{}),
		TLSCiphers:           config.StringSlice("TLS_CIPHERS", utils.GetDefaultCiphers()),
		TLSMinVersion:        config.StringVal("TLS_MIN_VERSION", utils.DefaultTLSMinVersion),
		DockerLogDriver:      config.StringVal("DOCKER_LOG_DRIVER", "json-file"),
		DockerLogConfigList:  config.StringSlice("DOCKER_LOG_CONFIG", []string{"max-file=5", "max-size=10m"}),
	}

	options.Endpoint = config.StringVal("ENDPOINT", "")

	// Set the path to the controller binary
	dir, _, err := node.ExecPath()
	if err != nil {
		glog.Warningf("Unable to find path to current serviced binary; assuming /opt/serviced/bin")
		dir = "/opt/serviced/bin"
	}
	defaultControllerBinary := filepath.Join(dir, "serviced-controller")
	options.ControllerBinary = config.StringVal("CONTROLLER_BINARY", defaultControllerBinary)

	// Set the volumePath to /tmp if running serviced as just an agent
	homepath := config.StringVal("HOME", "")
	varpath := config.StringVal("VARPATH", getDefaultVarPath(homepath))
	if options.Master {
		options.IsvcsPath = config.StringVal("ISVCS_PATH", filepath.Join(varpath, "isvcs"))
		options.VolumesPath = config.StringVal("VOLUMES_PATH", filepath.Join(varpath, "volumes"))
		options.BackupsPath = config.StringVal("BACKUPS_PATH", filepath.Join(varpath, "backups"))
	} else {
		tmpvarpath := getDefaultVarPath("")
		options.IsvcsPath = filepath.Join(varpath, "isvcs")
		options.VolumesPath = filepath.Join(tmpvarpath, "volumes")
		options.BackupsPath = filepath.Join(varpath, "backups")
	}

	options.StorageArgs = getDefaultStorageOptions(options.FSType, config)

	return options
}

func getDefaultDockerRegistry() string {
	if hostname, err := os.Hostname(); err != nil {
		return docker.DEFAULT_REGISTRY
	} else {
		return fmt.Sprintf("%s:5000", hostname)
	}
}

func getDefaultVarPath(home string) string {
	if home == "" {
		if user, err := user.Current(); err != nil {
			home = filepath.Join(os.TempDir(), "serviced")
		} else {
			home = filepath.Join(os.TempDir(), "serviced-"+user.Username)
		}
	}

	return filepath.Join(home, "var")
}

func getDefaultESStartupTimeout(timeout int) int {
	minTimeout := isvcs.MIN_ES_STARTUP_TIMEOUT_SECONDS
	if timeout < minTimeout {
		timeout = minTimeout
	}
	return timeout
}

func getDefaultAdminGroup() string {
	if utils.Platform == utils.Rhel {
		return "wheel"
	} else {
		return "sudo"
	}
}

