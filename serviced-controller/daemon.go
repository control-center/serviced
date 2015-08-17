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
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
)

func main() {
	defaultRPCPort := 4979
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
	app.Flags = []cli.Flag{
		cli.StringFlag{"forwarder-binary", "/usr/local/serviced/resources/logstash/logstash-forwarder", "path to the logstash-forwarder binary"},
		cli.StringFlag{"forwarder-config", "/etc/logstash-forwarder.conf", "path to the logstash-forwarder config file"},
		cli.IntFlag{"muxport", 22250, "multiplexing port to use"},
		cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private keys"},
		cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringFlag{"endpoint", utils.GetGateway(defaultRPCPort), "serviced endpoint address"},
		cli.BoolTFlag{"autorestart", "restart process automatically when it finishes"},
		cli.BoolFlag{"disable-metric-forwarding", "disable forwarding of metrics for this container"},
		cli.StringFlag{"metric-forwarder-port", defaultMetricsForwarderPort, "the port the container processes send performance data to"},
		cli.BoolTFlag{"logstash", "forward service logs via logstash-forwarder"},
		cli.StringFlag{"logstash-idle-flush-time", "5s", "time duration for logstash to flush log messages"},
		cli.StringFlag{"logstash-settle-time", "0s", "time duration to wait for logstash to flush log messages before closing"},
		cli.StringFlag{"virtual-address-subnet", "10.3", "/16 subnet for virtual addresses"},
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
	}

	app.Action = CmdServiceProxy
	app.Run(os.Args)
}