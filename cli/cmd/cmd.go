// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
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
	"os/signal"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"
)

const defaultRPCPort = 4979

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver       api.API
	app          *cli.App
	config       utils.ConfigReader
	exitDisabled bool
}

// New instantiates a new command-line client
func New(driver api.API, config utils.ConfigReader) *ServicedCli {
	if config == nil {
		glog.Fatal("Missing configuration data!")
	}
	defaultOps := getDefaultOptions(config)
	masterIP := config.StringVal("MASTER_IP", "127.0.0.1")

	c := &ServicedCli{
		driver: driver,
		app:    cli.NewApp(),
		config: config,
	}

	c.app.Name = "serviced"
	c.app.Usage = "A container-based management system"
	c.app.Version = fmt.Sprintf("%s - %s ", servicedversion.Version, servicedversion.Gitcommit)
	c.app.EnableBashCompletion = true
	c.app.Before = c.cmdInit
	c.app.Flags = []cli.Flag{
		cli.StringFlag{"docker-registry", defaultOps.DockerRegistry, "local docker registry to use"},
		cli.StringSliceFlag{"static-ip", convertToStringSlice(defaultOps.StaticIPs), "static ips for this agent to advertise"},
		cli.StringFlag{"endpoint", defaultOps.Endpoint, fmt.Sprintf("endpoint for remote serviced (example.com:%d)", defaultRPCPort)},
		cli.StringFlag{"outbound", defaultOps.OutboundIP, "outbound ip address"},
		cli.StringFlag{"uiport", defaultOps.UIPort, "port for ui"},
		cli.StringFlag{"nfs-client", defaultOps.NFSClient, "establish agent as an nfs client sharing data, 0 to disable"},
		cli.IntFlag{"listen", config.IntVal("RPC_PORT", defaultRPCPort), fmt.Sprintf("rpc port for serviced (%d)", defaultRPCPort)},
		cli.StringSliceFlag{"docker-dns", convertToStringSlice(defaultOps.DockerDNS), "docker dns configuration used for running containers"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control center service"},
		cli.BoolFlag{"agent", "run in agent mode, i.e., a host in a resource pool"},
		cli.IntFlag{"mux", defaultOps.MuxPort, "multiplexing port"},
		cli.StringFlag{"volumes-path", defaultOps.VolumesPath, "path where application data is stored"},
		cli.StringFlag{"isvcs-path", defaultOps.IsvcsPath, "path where internal application data is stored"},
		cli.StringFlag{"backups-path", defaultOps.BackupsPath, "default path where backups are stored"},
		cli.StringFlag{"keyfile", defaultOps.KeyPEMFile, "path to private key file (defaults to compiled in private key)"},
		cli.StringFlag{"certfile", defaultOps.CertPEMFile, "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringSliceFlag{"zk", convertToStringSlice(defaultOps.Zookeepers), "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181)"},
		cli.StringSliceFlag{"mount", convertToStringSlice(defaultOps.Mount), "bind mount: DOCKER_IMAGE,HOST_PATH[,CONTAINER_PATH]"},
		cli.StringFlag{"fstype", string(defaultOps.FSType), "driver for underlying file system"},
		cli.StringSliceFlag{"alias", convertToStringSlice(defaultOps.HostAliases), "list of aliases for this host, e.g., localhost"},
		cli.IntFlag{"es-startup-timeout", defaultOps.ESStartupTimeout, "time (in seconds) to wait on elasticsearch startup before bailing"},
		cli.IntFlag{"max-container-age", defaultOps.MaxContainerAge, "maximum age (seconds) of a stopped container before removing"},
		cli.IntFlag{"max-dfs-timeout", defaultOps.MaxDFSTimeout, "max timeout to perform a dfs snapshot"},
		cli.StringFlag{"virtual-address-subnet", defaultOps.VirtualAddressSubnet, "/16 subnet for virtual addresses"},
		cli.StringFlag{"master-pool-id", defaultOps.MasterPoolID, "master's pool ID"},
		cli.StringFlag{"admin-group", defaultOps.AdminGroup, "system group that can log in to control center"},
		cli.StringSliceFlag{"storage-opts", convertToStringSlice(defaultOps.StorageArgs), "storage args to initialize filesystem"},
		cli.StringSliceFlag{"isvcs-start", convertToStringSlice(defaultOps.StartISVCS), "isvcs to start on agent"},

		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", defaultOps.HostStats, "container statistics for host:port"},
		cli.IntFlag{"stats-period", defaultOps.StatsPeriod, "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", defaultOps.MCUsername, "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", defaultOps.MCPasswd, "Password for the Zenoss metric consumer"},
		cli.StringFlag{"cpuprofile", defaultOps.CPUProfile, "write cpu profile to file"},
		cli.StringSliceFlag{"isvcs-env", convertToStringSlice(config.StringSlice("ISVCS_ENV", []string{})), "internal-service environment variable: ISVC:KEY=VAL"},
		cli.IntFlag{"debug-port", defaultOps.DebugPort, "Port on which to listen for profiler connections"},
		cli.IntFlag{"max-rpc-clients", defaultOps.MaxRPCClients, "max number of rpc clients to an endpoint"},
		cli.IntFlag{"rpc-dial-timeout", defaultOps.RPCDialTimeout, "timeout for creating rpc connections"},
		cli.IntFlag{"snapshot-ttl", defaultOps.SnapshotTTL, "snapshot TTL in hours, 0 to disable"},
		cli.StringFlag{"controller-binary", defaultOps.ControllerBinary, "path to the container controller binary"},

		// Reimplementing GLOG flags :(
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
		cli.BoolFlag{"alsologtostderr", "log to standard error as well as files"},
		cli.StringFlag{"logstashurl", config.StringVal("LOG_ADDRESS", fmt.Sprintf("%s:5042", masterIP)), "logstash url and port"},
		cli.StringFlag{"logstash-es", defaultOps.LogstashES, "host and port for logstash elastic search"},
		cli.IntFlag{"logstash-max-days", defaultOps.LogstashMaxDays, "days to keep Logstash data"},
		cli.IntFlag{"logstash-max-size", defaultOps.LogstashMaxSize, "max size of Logstash data to keep in gigabytes"},
		cli.IntFlag{"v", defaultOps.Verbosity, "log level for V logs"},
		cli.StringFlag{"stderrthreshold", "", "logs at or above this threshold go to stderr"},
		cli.StringFlag{"vmodule", "", "comma-separated list of pattern=N settings for file-filtered logging"},
		cli.StringFlag{"log_backtrace_at", "", "when logging hits line file:N, emit a stack trace"},
		cli.StringFlag{"config-file", "/etc/default/serviced", "path to config"},
	}

	c.initVersion()
	c.initPool()
	c.initHealthCheck()
	c.initHost()
	c.initTemplate()
	c.initService()
	c.initSnapshot()
	c.initLog()
	c.initBackup()
	c.initMetric()
	c.initDocker()
	c.initScript()
	c.initServer()
	c.initVolume()

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	if err := c.app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

// cmdInit starts the server if no subcommands are called
func (c *ServicedCli) cmdInit(ctx *cli.Context) error {
	options := api.Options{
		DockerRegistry:       ctx.GlobalString("docker-registry"),
		NFSClient:            ctx.GlobalString("nfs-client"),
		Endpoint:             ctx.GlobalString("endpoint"),
		StaticIPs:            ctx.GlobalStringSlice("static-ip"),
		UIPort:               ctx.GlobalString("uiport"),
		RPCPort:              fmt.Sprintf("%d", ctx.GlobalInt("listen")),
		Listen:               fmt.Sprintf(":%d", ctx.GlobalInt("listen")),
		DockerDNS:            ctx.GlobalStringSlice("docker-dns"),
		Master:               ctx.GlobalBool("master"),
		Agent:                ctx.GlobalBool("agent"),
		MuxPort:              ctx.GlobalInt("mux"),
		TLS:                  true,
		VolumesPath:          ctx.GlobalString("volumes-path"),
		IsvcsPath:            ctx.GlobalString("isvcs-path"),
		BackupsPath:          ctx.GlobalString("backups-path"),
		KeyPEMFile:           ctx.GlobalString("keyfile"),
		CertPEMFile:          ctx.GlobalString("certfile"),
		Zookeepers:           ctx.GlobalStringSlice("zk"),
		Mount:                ctx.GlobalStringSlice("mount"),
		HostAliases:          ctx.GlobalStringSlice("alias"),
		ESStartupTimeout:     ctx.GlobalInt("es-startup-timeout"),
		ReportStats:          ctx.GlobalBool("report-stats"),
		HostStats:            ctx.GlobalString("host-stats"),
		StatsPeriod:          ctx.GlobalInt("stats-period"),
		MCUsername:           ctx.GlobalString("mc-username"),
		MCPasswd:             ctx.GlobalString("mc-password"),
		Verbosity:            ctx.GlobalInt("v"),
		CPUProfile:           ctx.GlobalString("cpuprofile"),
		MaxContainerAge:      ctx.GlobalInt("max-container-age"),
		MaxDFSTimeout:        ctx.GlobalInt("max-dfs-timeout"),
		VirtualAddressSubnet: ctx.GlobalString("virtual-address-subnet"),
		MasterPoolID:         ctx.GlobalString("master-pool-id"),
		OutboundIP:           ctx.GlobalString("outbound"),
		LogstashES:           ctx.GlobalString("logstash-es"),
		LogstashMaxDays:      ctx.GlobalInt("logstash-max-days"),
		LogstashMaxSize:      ctx.GlobalInt("logstash-max-size"),
		DebugPort:            ctx.GlobalInt("debug-port"),
		AdminGroup:           ctx.GlobalString("admin-group"),
		MaxRPCClients:        ctx.GlobalInt("max-rpc-clients"),
		RPCDialTimeout:       ctx.GlobalInt("rpc-dial-timeout"),
		SnapshotTTL:          ctx.GlobalInt("snapshot-ttl"),
		StorageArgs:          ctx.GlobalStringSlice("storage-opts"),
		ControllerBinary:     ctx.GlobalString("controller-binary"),
		StartISVCS:           ctx.GlobalStringSlice("isvcs-start"),
	}
	if os.Getenv("SERVICED_MASTER") == "1" {
		options.Master = true
	}
	if os.Getenv("SERVICED_AGENT") == "1" {
		options.Agent = true
	}

	if options.Master {
		fstype := ctx.GlobalString("fstype")
		options.FSType = volume.DriverType(fstype)
	} else {
		options.FSType = volume.DriverTypeNFS
	}

	if len(options.StorageArgs) == 0 {
		options.StorageArgs = getDefaultStorageOptions(options.FSType)
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

	if err := setIsvcsEnv(ctx); err != nil {
		fmt.Println(err)
		return err
	}

	return nil
}

func (c *ServicedCli) exit(code int) error {
	if c.exitDisabled {
		return fmt.Errorf("exit code %v", code)
	}
	os.Exit(code)
	return nil
}

func setLogging(ctx *cli.Context) error {

	if ctx.GlobalBool("master") || ctx.GlobalBool("agent") {
		hostname, err := os.Hostname()
		if err != nil {
			hostname = "unknown"
		}
		glog.SetLogstashType("serviced-" + hostname)
	}

	if ctx.IsSet("logtostderr") {
		glog.SetToStderr(ctx.GlobalBool("logtostderr"))
	}

	if ctx.IsSet("alsologtostderr") {
		glog.SetAlsoToStderr(ctx.GlobalBool("alsologtostderr"))
	}

	glog.SetLogstashURL(ctx.GlobalString("logstashurl"))

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

	// Listen for SIGUSR1 and, when received, toggle the log level between
	// 0 and 2.	 If the log level is anything but 0, we set it to 0, and on
	// subsequent signals, set it to 2.
	go func() {
		signalChan := make(chan os.Signal, 1)
		signal.Notify(signalChan, syscall.SIGUSR1)
		for {
			<-signalChan
			glog.Infof("Received signal SIGUSR1")
			if glog.GetVerbosity() == 0 {
				glog.SetVerbosity(2)
			} else {
				glog.SetVerbosity(0)
			}
			glog.Infof("Log level changed to %v", glog.GetVerbosity())
		}
	}()

	return nil
}

func setIsvcsEnv(ctx *cli.Context) error {
	for _, val := range ctx.GlobalStringSlice("isvcs-env") {
		if err := isvcs.AddEnv(val); err != nil {
			return err
		}
	}
	return nil
}

func init() {
	// Change the representation of the version flag
	cli.VersionFlag = cli.BoolFlag{"version", "print the version"}
}
