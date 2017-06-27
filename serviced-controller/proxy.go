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
	"path/filepath"
	"os"
	"strconv"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/container"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	log "github.com/Sirupsen/logrus"
	"github.com/zenoss/logri"
)

var plog = logging.PackageLogger()

func CmdServiceProxy(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		os.Exit(1)
	}
	cfg := utils.NewEnvironOnlyConfigReader("SERVICED_")
	options := ControllerOptions{
		MuxPort:                 ctx.GlobalInt("muxport"),
		MUXDisableTLS:           ctx.GlobalBool("mux-disable-tls"),
		KeyPEMFile:              ctx.GlobalString("keyfile"),
		CertPEMFile:             ctx.GlobalString("certfile"),
		RPCPort:                 ctx.GlobalInt("rpcport"),
		RPCDisableTLS:           ctx.GlobalBool("rpc-disable-tls"),
		Autorestart:             ctx.GlobalBool("autorestart"),
		MetricForwarderPort:     ctx.GlobalString("metric-forwarder-port"),
		Logstash:                ctx.GlobalBool("logstash"),
		LogstashSettleTime:      ctx.GlobalString("logstash-settle-time"),
		LogstashBinary:          ctx.GlobalString("forwarder-binary"),
		LogstashConfig:          ctx.GlobalString("forwarder-config"),
		LogstashURL:             ctx.GlobalString("logstashurl"),
		VirtualAddressSubnet:    ctx.GlobalString("virtual-address-subnet"),
		ServiceID:               args[0],
		InstanceID:              args[1],
		Command:                 args[2:],
		MetricForwardingEnabled: !ctx.GlobalBool("disable-metric-forwarding"),
	}

	options.MuxPort = cfg.IntVal("MUX_PORT", options.MuxPort)
	options.RPCPort = cfg.IntVal("RPC_PORT", options.RPCPort)
	options.KeyPEMFile = cfg.StringVal("KEY_FILE", options.KeyPEMFile)		// TODO: Is this set in container.go?
	options.CertPEMFile = cfg.StringVal("CERT_FILE", options.CertPEMFile)		// TODO: Is this set in container.go?
	options.LogstashURL = cfg.StringVal("LOG_ADDRESS", options.LogstashURL)
	options.VirtualAddressSubnet = cfg.StringVal("VIRTUAL_ADDRESS_SUBNET", options.VirtualAddressSubnet)
	options.ServicedEndpoint = utils.GetGateway(options.RPCPort)
	options.HostIPs = os.Getenv("CONTROLPLANE_HOST_IPS")

	if ctx.IsSet("logtostderr") {
		glog.SetToStderr(ctx.GlobalBool("logtostderr"))
	}

	rpcutils.RPC_CLIENT_SIZE = 2
	rpcutils.RPCDisableTLS = options.RPCDisableTLS

	if err := StartProxy(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
		// exit with an error if we can't start the proxy so that the delegate can record the container logs
		os.Exit(1)
	}
}

func StartProxy(options ControllerOptions) error {
	logri.ApplyConfigFromFile(filepath.Join("/etc/serviced", "logconfig-controller.yaml"))
	coordzk.RegisterZKLogger()

	// TODO: CC-3061: Our use of logrus does not support a similar integration with logstash
	glog.SetLogstashType("controller-" + options.ServiceID + "-" + options.InstanceID)
	glog.SetLogstashURL(options.LogstashURL)

	o, err := options.toContainerControllerOptions()
	if err != nil {
		return err
	}

	plog.WithFields(log.Fields{
		"muxport": o.Mux.Port,
		"muxdisabletls": strconv.FormatBool(o.Mux.DisableTLS),
		"serviceendpoint": o.ServicedEndpoint,
		"rpcdeisabletls": strconv.FormatBool(o.RPCDisableTLS),
	}).Debug("Starting container proxy")
	c, err := container.NewController(o)
	if err != nil {
		return err
	}
	return c.Run()

}
