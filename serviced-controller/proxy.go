package main

import (
	"fmt"
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/container"
	"github.com/zenoss/glog"
)

func CmdServiceProxy(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Printf("Incorrect Usage.\n\n")
		os.Exit(1)
	}
	options := ControllerOptions{
		MuxPort:                 ctx.GlobalInt("muxport"),
		TLS:                     true,
		KeyPEMFile:              ctx.GlobalString("keyfile"),
		CertPEMFile:             ctx.GlobalString("certfile"),
		ServicedEndpoint:        ctx.GlobalString("endpoint"),
		Autorestart:             ctx.GlobalBool("autorestart"),
		MetricForwarderPort:     ctx.GlobalString("metric-forwarder-port"),
		Logstash:                ctx.GlobalBool("logstash"),
		LogstashIdleFlushTime:   ctx.GlobalString("logstash-idle-flush-time"),
		LogstashSettleTime:      ctx.GlobalString("logstash-settle-time"),
		LogstashBinary:          ctx.GlobalString("forwarder-binary"),
		LogstashConfig:          ctx.GlobalString("forwarder-config"),
		VirtualAddressSubnet:    ctx.GlobalString("virtual-address-subnet"),
		ServiceID:               args[0],
		InstanceID:              args[1],
		Command:                 args[2:],
		MetricForwardingEnabled: !ctx.GlobalBool("disable-metric-forwarding"),
	}
	if err := StartProxy(options); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}

func StartProxy(options ControllerOptions) error {
	glog.SetLogstashType("controller-" + options.ServiceID + "-" + options.InstanceID)

	o, err := options.toContainerControllerOptions()
	if err != nil {
		return err
	}

	c, err := container.NewController(o)
	if err != nil {
		return err
	}
	return c.Run()

}
