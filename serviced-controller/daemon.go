package main

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/servicedversion"
	"github.com/zenoss/glog"
)

// URL parses and handles URL typed options
type URL struct {
	Host string
	Port int
}

// Set converts a URL string to a URL object
func (u *URL) Set(value string) error {
	parts := strings.Split(value, ":")
	if len(parts) != 2 {
		return fmt.Errorf("bad format: %s; must be formatted as HOST:PORT", value)
	}

	u.Host = parts[0]
	if port, err := strconv.Atoi(parts[1]); err != nil {
		return fmt.Errorf("port does not parse as an integer")
	} else {
		u.Port = port
	}
	return nil
}

func (u *URL) String() string {
	return fmt.Sprintf("%s:%d", u.Host, u.Port)
}

// GetGateway returns the default gateway
func GetGateway(defaultRPCPort int) string {
	cmd := exec.Command("ip", "route")
	output, err := cmd.Output()
	localhost := URL{"127.0.0.1", defaultRPCPort}

	if err != nil {
		glog.V(2).Info("Error checking gateway: ", err)
		glog.V(1).Info("Could not get gateway using ", localhost.Host)
		return localhost.String()
	}

	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) > 2 && fields[0] == "default" {
			endpoint := URL{fields[2], defaultRPCPort}
			return endpoint.String()
		}
	}
	glog.V(1).Info("No gateway found, using ", localhost.Host)
	return localhost.String()
}

func main() {
	defaultMetricsForwarderPort := ":22350"
	defaultRPCPort := 4979

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
		cli.StringFlag{"endpoint", GetGateway(defaultRPCPort), "serviced endpoint address"},
		cli.BoolTFlag{"autorestart", "restart process automatically when it finishes"},
		cli.BoolFlag{"disable-metric-forwarding", "disable forwarding of metrics for this container"},
		cli.StringFlag{"metric-forwarder-port", defaultMetricsForwarderPort, "the port the container processes send performance data to"},
		cli.BoolTFlag{"logstash", "forward service logs via logstash-forwarder"},
		cli.StringFlag{"logstash-idle-flush-time", "5s", "time duration for logstash to flush log messages"},
		cli.StringFlag{"logstash-settle-time", "0s", "time duration to wait for logstash to flush log messages before closing"},
		cli.StringFlag{"virtual-address-subnet", "10.3", "/16 subnet for virtual addresses"},
	}

	app.Action = CmdServiceProxy
	app.Run(os.Args)
}