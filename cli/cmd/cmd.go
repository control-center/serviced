package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/cli/api"
)

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver api.API
	app    *cli.App
}

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
	c.app.Version = "1.0.0"
	c.app.EnableBashCompletion = true
	c.app.Before = c.cmdInit
	c.app.Flags = []cli.Flag{
		cli.StringFlag{"port", agentIP, "port for remote serviced (example.com:8080)"},
		cli.StringFlag{"uiport", ":443", "port for ui"},
		cli.StringFlag{"listen", ":4979", "port for local serviced (example.com:8080)"},
		cli.StringSliceFlag{"docker-dns", &dockerDNS, "docker dns configuration used for running containers"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control plane service"},
		cli.BoolFlag{"agent", "run in agent mode, i.e., a host in a resource pool"},
		cli.IntFlag{"mux", 22250, "multiplexing port"},
		cli.BoolFlag{"tls", "enable TLS"},
		cli.StringFlag{"var", varPath, "path to store serviced data"},
		cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private key)"},
		cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringSliceFlag{"zk", &cli.StringSlice{}, "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181)"},
		cli.StringSliceFlag{"mount", &cli.StringSlice{}, "bind mount: DOCKER_IMAGE,HOST_PATH[,CONTAINER_PATH]"},
		cli.StringFlag{"vfs", "rsync", "filesystem for container volumes"},
		cli.StringSliceFlag{"alias", &cli.StringSlice{}, "list of aliases for this host, e.g., localhost"},
		cli.IntFlag{"es-startup-timeout", esStartupTimeout, "time to wait on elasticsearch startup before bailing"},

		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", "127.0.0.1:8443", "container statistics for host:port"},
		cli.IntFlag{"stats-period", 60, "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", "scott", "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", "tiger", "Password for the Zenoss metric consumer"},

		// Reimplementing GLOG flags :(
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
		cli.BoolFlag{"alsologtostderr", "log to standard error as well as files"},
		cli.StringFlag{"logstashtype", "", "enable logstash logging and define the type"},
		cli.StringFlag{"logstashurl", "172.17.42.1:5042", "logstash url and port"},
		cli.IntFlag{"v", 0, "log level for V logs"},
		cli.StringFlag{"stderrthreshold", "", "logs at or above this threshold go to stderr"},
		cli.StringFlag{"vmodule", "", "comma-separated list of pattern=N settings for file-filtered logging"},
		cli.StringFlag{"log_backtrace_at", "", "when logging hits line file:N, emit a stack trace"},
	}

	c.initPool()
	c.initHost()
	c.initTemplate()
	c.initService()
	c.initSnapshot()
	c.initBackup()

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	c.app.Run(args)
}

// cmdInit starts the server if no subcommands are called
func (c *ServicedCli) cmdInit(ctx *cli.Context) error {
	options := api.Options{
		Port:             ctx.GlobalString("port"),
		UIPort:           ctx.GlobalString("uiport"),
		Listen:           ctx.GlobalString("listen"),
		DockerDNS:        ctx.GlobalStringSlice("docker-dns"),
		Master:           ctx.GlobalBool("master"),
		Agent:            ctx.GlobalBool("agent"),
		MuxPort:          ctx.GlobalInt("mux"),
		TLS:              ctx.GlobalBool("tls"),
		VarPath:          ctx.GlobalString("var"),
		KeyPEMFile:       ctx.GlobalString("keyfile"),
		CertPEMFile:      ctx.GlobalString("certfile"),
		Zookeepers:       ctx.GlobalStringSlice("zk"),
		Mount:            ctx.GlobalStringSlice("mount"),
		VFS:              ctx.GlobalString("vfs"),
		HostAliases:      ctx.GlobalStringSlice("alias"),
		ESStartupTimeout: ctx.GlobalInt("es-startup-timeout"),
		ReportStats:      ctx.GlobalBool("report-stats"),
		HostStats:        ctx.GlobalString("host-stats"),
		StatsPeriod:      ctx.GlobalInt("stats-period"),
		MCUsername:       ctx.GlobalString("mc-username"),
		MCPasswd:         ctx.GlobalString("mc-password"),
		Verbosity:        ctx.GlobalInt("v"),
	}

	api.LoadOptions(options)

	// Set logging options
	if err := setLogging(ctx); err != nil {
		return err
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

	if ctx.IsSet("alsotostderr") {
		glog.SetAlsoToStderr(ctx.GlobalBool("alsotostderr"))
	}

	if ctx.IsSet("logstashType") {
		glog.SetLogstashType(ctx.GlobalString("logstashType"))
	}

	if ctx.IsSet("logstashURL") {
		glog.SetLogstashURL(ctx.GlobalString("logstashURL"))
	}

	if ctx.IsSet("v") {
		glog.SetVerbosity(ctx.GlobalInt("v"))
	}

	if ctx.IsSet("stderrThreshold") {
		if err := glog.SetStderrThreshold(ctx.GlobalString("stderrThreshold")); err != nil {
			return err
		}
	}

	if ctx.IsSet("vmodule") {
		if err := glog.SetVModule(ctx.GlobalString("vmodule")); err != nil {
			return err
		}
	}

	if ctx.IsSet("traceLocation") {
		if err := glog.SetTraceLocation(ctx.GlobalString("traceLocation")); err != nil {
			return err
		}
	}

	return nil
}

func init() {
	// Change the representation of the version flag
	cli.VersionFlag = cli.BoolFlag{"version", "print the version"}
}
