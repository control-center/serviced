// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cmd

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/servicedversion"
	"github.com/zenoss/serviced/validation"
)

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver api.API
	app    *cli.App
}

const envPrefix = "SERVICED_"

func configEnv(key string, defaultVal string) string {
	s := os.Getenv(envPrefix + key)
	if len(s) == 0 {
		return defaultVal
	}
	return s
}
func configInt(key string, defaultVal int) int {
	s := configEnv(key, "")
	if len(s) == 0 {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil {
		return defaultVal
	}
	return v
}

const defaultRPCPort = 4979

// New instantiates a new command-line client
func New(driver api.API) *ServicedCli {
	var (
		agentIP          = api.GetAgentIP()
		varPath          = api.GetVarPath()
		esStartupTimeout = api.GetESStartupTimeout()
		dockerDNS        = cli.StringSlice(api.GetDockerDNS())
	)

	c := &ServicedCli{
		driver: driver,
		app:    cli.NewApp(),
	}

	c.app.Name = "serviced"
	c.app.Usage = "A container-based management system"
	c.app.Version = fmt.Sprintf("%s - %s ", servicedversion.Version, servicedversion.Gitcommit)
	c.app.EnableBashCompletion = true
	c.app.Before = c.cmdInit
	staticIps := cli.StringSlice{}
	if len(configEnv("STATIC_IPS", "")) > 0 {
		staticIps = cli.StringSlice(strings.Split(configEnv("STATIC_IPS", ""), ","))
	}

	defaultDockerRegistry := "localhost:5000"
	if hostname, err := os.Hostname(); err == nil {
		defaultDockerRegistry = fmt.Sprintf("%s:5000", hostname)
	}

	c.app.Flags = []cli.Flag{
		cli.StringFlag{"docker-registry", configEnv("DOCKER_REGISTRY", defaultDockerRegistry), "local docker registry to use"},
		cli.StringSliceFlag{"static-ip", &staticIps, "static ips for this agent to advertise"},
		cli.StringFlag{"endpoint", configEnv("ENDPOINT", agentIP), "endpoint for remote serviced (example.com:8080)"},
		cli.StringFlag{"uiport", configEnv("UI_PORT", ":443"), "port for ui"},
		cli.StringFlag{"listen", configEnv("RPC_PORT", fmt.Sprintf(":%d", defaultRPCPort)), "port for local serviced (example.com:8080)"},
		cli.StringSliceFlag{"docker-dns", &dockerDNS, "docker dns configuration used for running containers"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control plane service"},
		cli.BoolFlag{"agent", "run in agent mode, i.e., a host in a resource pool"},
		cli.IntFlag{"mux", configInt("MUX_PORT", 22250), "multiplexing port"},
		cli.BoolTFlag{"tls", "enable TLS"},
		cli.StringFlag{"var", varPath, "path to store serviced data"},
		cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private key)"},
		cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringSliceFlag{"zk", &cli.StringSlice{}, "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181)"},
		cli.StringSliceFlag{"mount", &cli.StringSlice{}, "bind mount: DOCKER_IMAGE,HOST_PATH[,CONTAINER_PATH]"},
		cli.StringFlag{"vfs", "rsync", "filesystem for container volumes"},
		cli.StringSliceFlag{"alias", &cli.StringSlice{}, "list of aliases for this host, e.g., localhost"},
		cli.IntFlag{"es-startup-timeout", esStartupTimeout, "time to wait on elasticsearch startup before bailing"},
		cli.IntFlag{"max-container-age", configInt("MAX_CONTAINER_AGE", 60), "maximum age of a stopped container before removing"},
		cli.StringFlag{"virtual-address-subnet", configEnv("VIRTUAL_ADDRESS_SUBNET", "10.3"), "/16 subnet for virtual addresses"},
		cli.StringFlag{"master-pool-id", "default", "master's pool ID"},

		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", "127.0.0.1:8443", "container statistics for host:port"},
		cli.IntFlag{"stats-period", 10, "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", "scott", "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", "tiger", "Password for the Zenoss metric consumer"},
		cli.StringFlag{"cpuprofile", "", "write cpu profile to file"},

		// Reimplementing GLOG flags :(
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
		cli.BoolFlag{"alsologtostderr", "log to standard error as well as files"},
		cli.StringFlag{"logstashtype", "", "enable logstash logging and define the type"},
		cli.StringFlag{"logstashurl", "172.17.42.1:5042", "logstash url and port"},
		cli.IntFlag{"v", configInt("LOG_LEVEL", 0), "log level for V logs"},
		cli.StringFlag{"stderrthreshold", "", "logs at or above this threshold go to stderr"},
		cli.StringFlag{"vmodule", "", "comma-separated list of pattern=N settings for file-filtered logging"},
		cli.StringFlag{"log_backtrace_at", "", "when logging hits line file:N, emit a stack trace"},
	}

	c.initVersion()
	c.initPool()
	c.initHost()
	c.initTemplate()
	c.initService()
	c.initSnapshot()
	c.initLog()
	c.initBackup()
	c.initDocker()

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	c.app.Run(args)
}

// cmdInit starts the server if no subcommands are called
func (c *ServicedCli) cmdInit(ctx *cli.Context) error {
	options := api.Options{
		DockerRegistry:       ctx.GlobalString("docker-registry"),
		Endpoint:             ctx.GlobalString("endpoint"),
		StaticIPs:            ctx.GlobalStringSlice("static-ip"),
		UIPort:               ctx.GlobalString("uiport"),
		Listen:               ctx.GlobalString("listen"),
		DockerDNS:            ctx.GlobalStringSlice("docker-dns"),
		Master:               ctx.GlobalBool("master"),
		Agent:                ctx.GlobalBool("agent"),
		MuxPort:              ctx.GlobalInt("mux"),
		TLS:                  ctx.GlobalBool("tls"),
		VarPath:              ctx.GlobalString("var"),
		KeyPEMFile:           ctx.GlobalString("keyfile"),
		CertPEMFile:          ctx.GlobalString("certfile"),
		Zookeepers:           ctx.GlobalStringSlice("zk"),
		Mount:                ctx.GlobalStringSlice("mount"),
		VFS:                  ctx.GlobalString("vfs"),
		HostAliases:          ctx.GlobalStringSlice("alias"),
		ESStartupTimeout:     ctx.GlobalInt("es-startup-timeout"),
		ReportStats:          ctx.GlobalBool("report-stats"),
		HostStats:            ctx.GlobalString("host-stats"),
		StatsPeriod:          ctx.GlobalInt("stats-period"),
		MCUsername:           ctx.GlobalString("mc-username"),
		MCPasswd:             ctx.GlobalString("mc-password"),
		Verbosity:            ctx.GlobalInt("v"),
		CPUProfile:           ctx.GlobalString("cpuprofile"),
		VirtualAddressSubnet: ctx.GlobalString("virtual-address-subnet"),
		MasterPoolID:         ctx.GlobalString("master-pool-id"),
	}

	if err := validation.IsSubnet16(options.VirtualAddressSubnet); err != nil {
		fmt.Fprintf(os.Stderr, "error validating virtual-address-subnet: %s\n", err)
		return fmt.Errorf("error validating virtual-address-subnet: %s", err)
	}

	api.LoadOptions(options)

	// Set logging options
	if err := setLogging(ctx); err != nil {
		fmt.Println(err)
	}

	if options.Master {
		fmt.Println("This master has been configured to be in pool: " + options.MasterPoolID)
	}

	// Start server mode
	if (options.Master || options.Agent) && len(ctx.Args()) == 0 {
		c.driver.StartServer()
		return fmt.Errorf("running server mode")
	}

	return nil
}

func setLogging(ctx *cli.Context) error {
	if ctx.IsSet("logtostderr") {
		glog.SetToStderr(ctx.GlobalBool("logtostderr"))
	}

	if ctx.IsSet("alsologtostderr") {
		glog.SetAlsoToStderr(ctx.GlobalBool("alsologtostderr"))
	}

	if ctx.IsSet("logstashtype") {
		glog.SetLogstashType(ctx.GlobalString("logstashtype"))
	}

	if ctx.IsSet("logstashurl") {
		glog.SetLogstashURL(ctx.GlobalString("logstashurl"))
	}

	if ctx.IsSet("v") {
		glog.SetVerbosity(ctx.GlobalInt("v"))
	}

	if ctx.IsSet("stderrthreshold") {
		if err := glog.SetStderrThreshold(ctx.GlobalString("stderrthreshold")); err != nil {
			return err
		}
	}

	if ctx.IsSet("vmodule") {
		if err := glog.SetVModule(ctx.GlobalString("vmodule")); err != nil {
			return err
		}
	}

	if ctx.IsSet("log_backtrace_at") {
		if err := glog.SetTraceLocation(ctx.GlobalString("log_backtrace_at")); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	// Change the representation of the version flag
	cli.VersionFlag = cli.BoolFlag{"version", "print the version"}
}
