// Copyright 2015 The Serviced Authors.
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

package cmd

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/options"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

func getDefaultOptions(config utils.ConfigReader) options.Options {
	masterIP := config.StringVal("MASTER_IP", "127.0.0.1")

	opts := options.Options{
		UIPort:               config.StringVal("UI_PORT", ":443"),
		NFSClient:            config.StringVal("NFS_CLIENT", "1"),
		RPCPort:              config.StringVal("RPC_PORT", fmt.Sprintf("%d", defaultRPCPort)),
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

	opts.Endpoint = config.StringVal("ENDPOINT", "")

	// Set the path to the controller binary
	dir, _, err := node.ExecPath()
	if err != nil {
		glog.Warningf("Unable to find path to current serviced binary; assuming /opt/serviced/bin")
		dir = "/opt/serviced/bin"
	}
	defaultControllerBinary := filepath.Join(dir, "serviced-controller")
	opts.ControllerBinary = config.StringVal("CONTROLLER_BINARY", defaultControllerBinary)

	// Set the volumePath to /tmp if running serviced as just an agent
	homepath := config.StringVal("HOME", "")
	varpath := config.StringVal("VARPATH", getDefaultVarPath(homepath))
	if opts.Master {
		opts.IsvcsPath = config.StringVal("ISVCS_PATH", filepath.Join(varpath, "isvcs"))
		opts.VolumesPath = config.StringVal("VOLUMES_PATH", filepath.Join(varpath, "volumes"))
		opts.BackupsPath = config.StringVal("BACKUPS_PATH", filepath.Join(varpath, "backups"))
	} else {
		tmpvarpath := getDefaultVarPath("")
		opts.IsvcsPath = filepath.Join(varpath, "isvcs")
		opts.VolumesPath = filepath.Join(tmpvarpath, "volumes")
		opts.BackupsPath = filepath.Join(varpath, "backups")
	}

	opts.StorageArgs = getDefaultStorageOptions(opts.FSType, config)

	return opts
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

func convertToStringSlice(list []string) *cli.StringSlice {
	slice := cli.StringSlice(list)
	return &slice
}
