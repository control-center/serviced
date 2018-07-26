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
	"path/filepath"
	"strconv"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/control-center/serviced/volume"
)

const (
	DefaultHomeDir       = "/opt/serviced"
	DefaultRPCPort       = 4979
	outboundIPRetryDelay = 1
	outboundIPMaxWait    = 90
)

// Validate options which are common to all CLI commands
func ValidateCommonOptions(opts config.Options) error {
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
		return fmt.Errorf("error validating UI port: %s", err)
	}

	// TODO: move this to ValidateServerOptions if this is really only used by master/agent, and not cli
	if err := validation.IsSubnetCIDR(opts.VirtualAddressSubnet); err != nil {
		return fmt.Errorf("error validating virtual-address-subnet: %s", err)
	}

	return nil
}

// Validate options which are specific to running as a server
func ValidateServerOptions(options *config.Options) error {
	if err := validateStorageArgs(options); err != nil {
		return err
	}

	// Make sure we have an endpoint to work with
	if len(options.Endpoint) == 0 {
		if options.Master {
			outboundIP, err := getOutboundIP()
			if err != nil {
				log.WithError(err).Fatal("Unable to determine outbound IP")
			}
			options.Endpoint = fmt.Sprintf("%s:%s", outboundIP, options.RPCPort)
		} else {
			return fmt.Errorf("No endpoint to master has been configured")
		}
	}

	if options.Master {
		log.WithFields(logrus.Fields{
			"poolid": options.MasterPoolID,
		}).Debug("Using configured default pool ID")
	}
	return nil
}

// GetOptionsRPCEndpoint returns the serviced RPC endpoint from options
func GetOptionsRPCEndpoint() string {
	return config.GetOptions().Endpoint
}

// GetOptionsRPCPort returns the serviced RPC port from options
func GetOptionsRPCPort() string {
	return config.GetOptions().RPCPort
}

// GetOptionsMaster returns the master mode setting from options
func GetOptionsMaster() bool {
	return config.GetOptions().Master
}

// GetOptionsAgent returns the agent mode setting from options
func GetOptionsAgent() bool {
	return config.GetOptions().Agent
}

// GetOptionsMasterPoolID returns the master pool ID from options
func GetOptionsMasterPoolID() string {
	return config.GetOptions().MasterPoolID
}

// GetOptionsMaxRPCClients returns the max RPC clients setting from options
func GetOptionsMaxRPCClients() int {
	return config.GetOptions().MaxRPCClients
}

func GetDefaultOptions(cfg utils.ConfigReader) config.Options {
	masterIP := cfg.StringVal("MASTER_IP", "127.0.0.1")

	options := config.Options{
		UIPort:                     service.ScrubPortString(cfg.StringVal("UI_PORT", ":443")),
		NFSClient:                  cfg.StringVal("NFS_CLIENT", "1"),
		RPCPort:                    cfg.StringVal("RPC_PORT", fmt.Sprintf("%d", DefaultRPCPort)),
		OutboundIP:                 cfg.StringVal("OUTBOUND_IP", ""),
		GCloud:                     cfg.BoolVal("GCLOUD", false),
		StartZK:                    cfg.BoolVal("START_ZK", true),
		StartAPIKeyProxy:           cfg.BoolVal("START_API_KEY_PROXY", false),
		BigTableMetrics:            cfg.BoolVal("BIGTABLE_METRICS", false),
		DockerDNS:                  cfg.StringSlice("DOCKER_DNS", []string{}),
		Master:                     cfg.BoolVal("MASTER", false),
		MuxPort:                    cfg.IntVal("MUX_PORT", 22250),
		MuxDisableTLS:              strconv.FormatBool(cfg.BoolVal("MUX_DISABLE_TLS", false)),
		KeyPEMFile:                 cfg.StringVal("KEY_FILE", ""),
		CertPEMFile:                cfg.StringVal("CERT_FILE", ""),
		Zookeepers:                 cfg.StringSlice("ZK", []string{}),
		HostStats:                  cfg.StringVal("STATS_PORT", fmt.Sprintf("%s:8443", masterIP)),
		StatsPeriod:                cfg.IntVal("STATS_PERIOD", 10),
		SvcStatsCacheTimeout:       cfg.IntVal("SVCSTATS_CACHE_TIMEOUT", 5),
		MCUsername:                 "scott",
		MCPasswd:                   "tiger",
		FSType:                     volume.DriverType(cfg.StringVal("FS_TYPE", "devicemapper")),
		ESStartupTimeout:           getDefaultESStartupTimeout(cfg.IntVal("ES_STARTUP_TIMEOUT", isvcs.DEFAULT_ES_STARTUP_TIMEOUT_SECONDS)),
		HostAliases:                cfg.StringSlice("VHOST_ALIASES", []string{}),
		Verbosity:                  cfg.IntVal("LOG_LEVEL", 0),
		StaticIPs:                  cfg.StringSlice("STATIC_IPS", []string{}),
		DockerRegistry:             cfg.StringVal("DOCKER_REGISTRY", "localhost:5000"),
		MaxContainerAge:            cfg.IntVal("MAX_CONTAINER_AGE", 60*60*24),
		MaxDFSTimeout:              cfg.IntVal("MAX_DFS_TIMEOUT", 60*5),
		VirtualAddressSubnet:       cfg.StringVal("VIRTUAL_ADDRESS_SUBNET", "10.3.0.0/16"),
		MasterPoolID:               cfg.StringVal("MASTER_POOLID", "default"),
		LogstashES:                 cfg.StringVal("LOGSTASH_ES", fmt.Sprintf("%s:9100", masterIP)),
		LogstashURL:                cfg.StringVal("LOG_ADDRESS", fmt.Sprintf("%s:5042", masterIP)),
		LogstashMaxDays:            cfg.IntVal("LOGSTASH_MAX_DAYS", 14),
		LogstashMaxSize:            cfg.IntVal("LOGSTASH_MAX_SIZE", 10),
		LogstashCycleTime:          cfg.IntVal("LOGSTASH_CYCLE_TIME", 6),
		DebugPort:                  cfg.IntVal("DEBUG_PORT", 6006),
		AdminGroup:                 cfg.StringVal("ADMIN_GROUP", getDefaultAdminGroup()),
		MaxRPCClients:              cfg.IntVal("MAX_RPC_CLIENTS", 3),
		MUXTLSCiphers:              cfg.StringSlice("MUX_TLS_CIPHERS", utils.GetDefaultCiphers("mux")),
		MUXTLSMinVersion:           cfg.StringVal("MUX_TLS_MIN_VERSION", utils.DefaultTLSMinVersion),
		RPCDialTimeout:             cfg.IntVal("RPC_DIAL_TIMEOUT", 30),
		RPCCertVerify:              strconv.FormatBool(cfg.BoolVal("RPC_CERT_VERIFY", false)),
		RPCDisableTLS:              strconv.FormatBool(cfg.BoolVal("RPC_DISABLE_TLS", false)),
		RPCTLSCiphers:              cfg.StringSlice("RPC_TLS_CIPHERS", utils.GetDefaultCiphers("rpc")),
		RPCTLSMinVersion:           cfg.StringVal("RPC_TLS_MIN_VERSION", utils.DefaultTLSMinVersion),
		SnapshotTTL:                cfg.IntVal("SNAPSHOT_TTL", 12),
		StartISVCS:                 cfg.StringSlice("ISVCS_START", []string{}),
		IsvcsENV:                   cfg.StringNumberedList("ISVCS_ENV", []string{}),
		IsvcsZKID:                  cfg.IntVal("ISVCS_ZOOKEEPER_ID", 0),
		IsvcsZKQuorum:              cfg.StringSlice("ISVCS_ZOOKEEPER_QUORUM", []string{}),
		TLSCiphers:                 cfg.StringSlice("TLS_CIPHERS", utils.GetDefaultCiphers("http")),
		TLSMinVersion:              cfg.StringVal("TLS_MIN_VERSION", utils.DefaultTLSMinVersion),
		DockerLogDriver:            cfg.StringVal("DOCKER_LOG_DRIVER", "json-file"),
		DockerLogConfigList:        cfg.StringSlice("DOCKER_LOG_CONFIG", []string{"max-file=5", "max-size=10m"}),
		AllowLoopBack:              strconv.FormatBool(cfg.BoolVal("ALLOW_LOOP_BACK", false)),
		UIPollFrequency:            cfg.IntVal("UI_POLL_FREQUENCY", 3),
		ConntrackFlush:             strconv.FormatBool(cfg.BoolVal("CONNTRACK_FLUSH", false)),
		StorageStatsUpdateInterval: cfg.IntVal("STORAGE_STATS_UPDATE_INTERVAL", 300),
		SnapshotSpacePercent:       cfg.IntVal("SNAPSHOT_USE_PERCENT", 20),
		ZKSessionTimeout:           cfg.IntVal("ZK_SESSION_TIMEOUT", 15),
		ZKConnectTimeout:           cfg.IntVal("ZK_CONNECT_TIMEOUT", 1),
		ZKPerHostConnectDelay:      cfg.IntVal("ZK_PER_HOST_CONNECT_DELAY", 0),
		ZKReconnectStartDelay:      cfg.IntVal("ZK_RECONNECT_START_DELAY", 1),
		ZKReconnectMaxDelay:        cfg.IntVal("ZK_RECONNECT_MAX_DELAY", 1),
		TokenExpiration:            cfg.IntVal("AUTH_TOKEN_EXPIRATION", 60*60),
		ServiceRunLevelTimeout:     cfg.IntVal("RUN_LEVEL_TIMEOUT", 60*10),
		StorageReportInterval:      cfg.IntVal("STORAGE_REPORT_INTERVAL", 30),
		StorageMetricMonitorWindow: cfg.IntVal("STORAGE_METRIC_MONITOR_WINDOW", 300),
		StorageLookaheadPeriod:     cfg.IntVal("STORAGE_LOOKAHEAD_PERIOD", 360),
		StorageMinimumFreeSpace:    cfg.StringVal("STORAGE_MIN_FREE", "3G"),
		BackupEstimatedCompression: cfg.Float64Val("BACKUP_ESTIMATED_COMPRESSION", 1.0),
		BackupMinOverhead:          cfg.StringVal("BACKUP_MIN_OVERHEAD", "0G"),
		// Auth0 configuration parameters. Default to empty strings - must edit in serviced.conf to configure for auth0.
		Auth0Domain:   cfg.StringVal("AUTH0_DOMAIN", ""),
		Auth0Audience: cfg.StringVal("AUTH0_AUDIENCE", ""),
		Auth0Group:    cfg.StringVal("AUTH0_GROUP", ""),
		Auth0ClientID: cfg.StringVal("AUTH0_CLIENT_ID", ""),
		Auth0Scope:    cfg.StringVal("AUTH0_SCOPE", ""),
		// Parameters for api-key-proxy isvc configuration
		KeyProxyJsonServer:   cfg.StringVal("KEYPROXY_JSON_SERVER", ""),
		KeyProxyListenPort:   cfg.StringVal("KEYPROXY_LISTEN_PORT", ":6443"), 
	}

	options.Endpoint = cfg.StringVal("ENDPOINT", "")

	// Set the path to the controller binary
	dir, _, err := node.ExecPath()
	if err != nil {
		log.Warn("Unable to find path to serviced binary; assuming /opt/serviced/bin")
		dir = "/opt/serviced/bin"
	}
	defaultControllerBinary := filepath.Join(dir, "serviced-controller")
	options.ControllerBinary = cfg.StringVal("CONTROLLER_BINARY", defaultControllerBinary)

	options.HomePath = cfg.StringVal("HOME", DefaultHomeDir)
	varpath := filepath.Join(options.HomePath, "var")

	options.IsvcsPath = cfg.StringVal("ISVCS_PATH", filepath.Join(varpath, "isvcs"))
	options.LogPath = cfg.StringVal("LOG_PATH", "/var/log/serviced")
	options.VolumesPath = cfg.StringVal("VOLUMES_PATH", filepath.Join(varpath, "volumes"))
	options.BackupsPath = cfg.StringVal("BACKUPS_PATH", filepath.Join(varpath, "backups"))
	options.EtcPath = cfg.StringVal("ETC_PATH", filepath.Join(options.HomePath, "etc"))
	options.StorageArgs = getDefaultStorageOptions(options.FSType, cfg)

	return options
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
	}
	return "sudo"
}

// getOutboundIP queries the network configuration for an IP address suitable for reaching the outside world.
// Will retry for a while if a path to the outside world is not yet available.
func getOutboundIP() (string, error) {
	var outboundIP string
	var err error
	timeout := time.After(outboundIPMaxWait * time.Second)
	for {
		if outboundIP, err = utils.GetIPAddress(); err == nil {
			// Success
			return outboundIP, nil
		}
		select {
		case <-timeout:
			// Give up
			return "", fmt.Errorf("Gave up waiting for network (to determine our outbound IP address)")
		default:
			// Retry
			log.Debug("Waiting for network initialization")
			time.Sleep(outboundIPRetryDelay * time.Second)
		}
	}
}
