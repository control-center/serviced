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

package main

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
)

func main() {
	defaultRPCPort := 4979
	defaultMuxPort := 22250
	defaultMetricsForwarderPort := ":22350"
	if cpConsumerUrl, err := url.Parse(os.Getenv("CONTROLPLANE_CONSUMER_URL")); err == nil {
		hostParts := strings.Split(cpConsumerUrl.Host, ":")
		if len(hostParts) == 2 {
			defaultMetricsForwarderPort = ":" + hostParts[1]
		}
	}
	app := cli.NewApp()
	app.Name = "serviced-controller"
	app.Usage = "serviced container controller"
	app.Version = fmt.Sprintf("%s - %s ", servicedversion.Version, servicedversion.Gitcommit)

	filebeatPath := filepath.Join(utils.LOGSTASH_CONTAINER_DIRECTORY, "filebeat")
	app.Flags = []cli.Flag{
		cli.StringFlag{"forwarder-binary", filebeatPath, "path to the filebeat binary"},
		cli.StringFlag{"forwarder-config", "/etc/filebeat.conf", "path to the filebeat config file"},
		cli.IntFlag{"muxport", defaultMuxPort, "multiplexing port to use"},
		cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private keys"},
		cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
		cli.IntFlag{"rpcport", defaultRPCPort, "port to use for RPC requests"},
		cli.BoolFlag{"rpc-disable-tls", "disable TLS for RPC requests"},
		cli.BoolTFlag{"autorestart", "restart process automatically when it finishes"},
		cli.BoolFlag{"mux-disable-tls", "disable contacting the mux via TLS"},
		cli.BoolFlag{"disable-metric-forwarding", "disable forwarding of metrics for this container"},
		cli.StringFlag{"metric-forwarder-port", defaultMetricsForwarderPort, "the port the container processes send performance data to"},
		cli.BoolTFlag{"logstash", "forward service logs via filebeat"},
		cli.StringFlag{"logstash-settle-time", "60s", "time duration to wait for logstash to flush log messages before closing"},
		cli.StringFlag{"virtual-address-subnet", "10.3.0.0/16", "/16 subnet for virtual addresses"},
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
	}

	app.Action = CmdServiceProxy
	app.Run(os.Args)
}
