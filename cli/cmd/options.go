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

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/utils"
)

func getDefaultOptions(config ConfigReader) api.Options {
	masterIP := config.StringVal("MASTER_IP", "127.0.0.1")

	options := api.Options{
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
		VarPath:              config.StringVal("VARPATH", getDefaultVarPath(config.StringVal("HOME", ""))),
		Zookeepers:           config.StringSlice("ZK", []string{}),
		HostStats:            config.StringVal("STATS_PORT", fmt.Sprintf("%s:8443", masterIP)),
		StatsPeriod:          config.IntVal("STATS_PERIOD", 10),
		MCUsername:           "scott",
		MCPasswd:             "tiger",
		FSType:               config.StringVal("FS_TYPE", "rsync"),
		ESStartupTimeout:     getDefaultESStartupTimeout(config.IntVal("ES_STARTUP_TIMEOUT", 600)),
		HostAliases:          config.StringSlice("VHOST_ALIASES", []string{}),
		Verbosity:            config.IntVal("LOG_LEVEL", 0),
		StaticIPs:            config.StringSlice("STATIC_IPS", []string{}),
		DockerRegistry:       config.StringVal("DOCKER_REGISTRY", getDefaultDockerRegistry()),
		MaxContainerAge:      config.IntVal("MAX_CONTAINER_AGE", 60*60*24),
		MaxDFSTimeout:        config.IntVal("MAX_DFS_TIMEOUT", 60*5),
		VirtualAddressSubnet: config.StringVal("VIRTUAL_ADDRESS_SUBNET", "10.3"),
		MasterPoolID:         config.StringVal("MASTER_POOLID", "default"),
		LogstashES:           config.StringVal("LOGSTASH_ES", fmt.Sprintf("%s:8443", masterIP)),
		LogstashMaxDays:      config.IntVal("LOGSTASH_MAX_DAYS", 14),
		LogstashMaxSize:      config.IntVal("LOGSTASH_MAX_SIZE", 10),
		DebugPort:            config.IntVal("DEBUG_PORT", 6006),
		AdminGroup:           config.StringVal("ADMIN_GROUP", getDefaultAdminGroup()),
		MaxRPCClients:        config.IntVal("MAX_RPC_CLIENTS", 3),
		RPCDialTimeout:       config.IntVal("RPC_DIAL_TIMEOUT", 30),
		SnapshotTTL:          config.IntVal("SNAPSHOT_TTL", 12),
		JWTTTL:               config.IntVal("JWT_TTL", 180),
	}

	options.Endpoint = config.StringVal("ENDPOINT", getDefaultEndpoint(options.OutboundIP, options.RPCPort))

	return options
}

func getDefaultDockerRegistry() string {
	if hostname, err := os.Hostname(); err != nil {
		return docker.DEFAULT_REGISTRY
	} else {
		return fmt.Sprintf("%s:5000", hostname)
	}
}

func getDefaultEndpoint(ip, port string) string {
	if ip == "" {
		var err error
		if ip, err = utils.GetIPAddress(); err != nil {
			panic(err)
		}
	}
	return fmt.Sprintf("%s:%s", ip, port)
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
	const minTimeout = 30
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
