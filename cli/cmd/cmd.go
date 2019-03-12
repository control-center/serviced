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
	"path/filepath"
	"syscall"

	"github.com/Sirupsen/logrus"
	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	"github.com/control-center/serviced/volume/nfs"
)

var (
	log = logging.PackageLogger()
)

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver       api.API
	app          *cli.App
	config       utils.ConfigReader
	logControl   logging.LogControl
	exitDisabled bool
}

// New instantiates a new command-line client
func New(driver api.API, config utils.ConfigReader, logControl logging.LogControl) *ServicedCli {
	if config == nil {
		log.Fatal("Missing configuration data")
	}
	defaultOps := api.GetDefaultOptions(config)

	c := &ServicedCli{
		driver:     driver,
		app:        cli.NewApp(),
		config:     config,
		logControl: logControl,
	}

	c.app.Name = "serviced"
	c.app.Usage = "A container-based management system"
	c.app.Version = fmt.Sprintf("%s - %s ", servicedversion.Version, servicedversion.Gitcommit)
	c.app.EnableBashCompletion = true
	c.app.Before = c.cmdInit
	c.app.Flags = []cli.Flag{
		cli.StringFlag{"docker-registry", defaultOps.DockerRegistry, "local docker registry to use"},
		cli.StringSliceFlag{"static-ip", convertToStringSlice(defaultOps.StaticIPs), "static ips for this agent to advertise"},
		cli.StringFlag{"endpoint", defaultOps.Endpoint, fmt.Sprintf("endpoint for remote serviced (example.com:%d)", api.DefaultRPCPort)},
		cli.StringFlag{"outbound", defaultOps.OutboundIP, "outbound ip address"},
		cli.StringFlag{"uiport", defaultOps.UIPort, "port for ui"},
		cli.StringFlag{"nfs-client", defaultOps.NFSClient, "establish agent as an nfs client sharing data, 0 to disable"},
		cli.IntFlag{"listen", config.IntVal("RPC_PORT", api.DefaultRPCPort), fmt.Sprintf("rpc port for serviced (%d)", api.DefaultRPCPort)},
		cli.StringSliceFlag{"docker-dns", convertToStringSlice(defaultOps.DockerDNS), "docker dns configuration used for running containers"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control center service"},
		cli.BoolFlag{"agent", "deprecated"},
		cli.IntFlag{"mux", defaultOps.MuxPort, "multiplexing port"},
		cli.StringFlag{"mux-disable-tls", defaultOps.MuxDisableTLS, "disable TLS for mux connections"},
		cli.StringSliceFlag{"mux-tls-ciphers", convertToStringSlice(defaultOps.MUXTLSCiphers), "list of supported TLS ciphers for MUX"},
		cli.StringFlag{"mux-tls-min-version", string(defaultOps.MUXTLSMinVersion), "mininum TLS version for MUX"},
		cli.StringFlag{"volumes-path", defaultOps.VolumesPath, "path where application data is stored"},
		cli.StringFlag{"isvcs-path", defaultOps.IsvcsPath, "path where internal application data is stored"},
		cli.StringFlag{"backups-path", defaultOps.BackupsPath, "default path where backups are stored"},
		cli.StringFlag{"etc-path", defaultOps.EtcPath, "default path for configuration files"},
		cli.StringFlag{"log-path", defaultOps.LogPath, "path where serviced logs are located"},
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
		cli.IntFlag{"isvcs-zk-id", defaultOps.IsvcsZKID, "zookeeper id when running in a cluster"},
		cli.StringSliceFlag{"isvcs-zk-quorum", convertToStringSlice(defaultOps.IsvcsZKQuorum), "isvcs zookeeper host quorum (e.g. -isvcs-zk-quorum zk1@localhost:2888:3888)"},
		cli.StringFlag{"isvcs-zk-username", defaultOps.IsvcsZKUsername, "isvcs zookeeper username"},
		cli.StringFlag{"isvcs-zk-passwd", defaultOps.IsvcsZKPasswd, "isvcs zookeeper password"},
		cli.StringSliceFlag{"isvcs-env", convertToStringSlice(defaultOps.IsvcsENV), "internal-service environment variable: ISVC:KEY=VAL"},
		cli.StringSliceFlag{"tls-ciphers", convertToStringSlice(defaultOps.TLSCiphers), "list of supported TLS ciphers for HTTP"},
		cli.StringFlag{"tls-min-version", string(defaultOps.TLSMinVersion), "mininum TLS version for HTTP"},

		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", defaultOps.HostStats, "container statistics for host:port"},
		cli.IntFlag{"stats-period", defaultOps.StatsPeriod, "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", defaultOps.MCUsername, "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", defaultOps.MCPasswd, "Password for the Zenoss metric consumer"},
		cli.StringFlag{"cpuprofile", defaultOps.CPUProfile, "write cpu profile to file"},
		cli.IntFlag{"debug-port", defaultOps.DebugPort, "Port on which to listen for profiler connections"},
		cli.IntFlag{"max-rpc-clients", defaultOps.MaxRPCClients, "max number of rpc clients to an endpoint"},
		cli.IntFlag{"rpc-dial-timeout", defaultOps.RPCDialTimeout, "timeout for creating rpc connections"},
		cli.StringFlag{"rpc-cert-verify", defaultOps.RPCCertVerify, "enable verification of rpc server certificate"},
		cli.StringFlag{"rpc-disable-tls", defaultOps.RPCDisableTLS, "disable tls for RPC connections"},
		cli.StringSliceFlag{"rpc-tls-ciphers", convertToStringSlice(defaultOps.RPCTLSCiphers), "list of supported TLS ciphers for RPC"},
		cli.StringFlag{"rpc-tls-min-version", string(defaultOps.RPCTLSMinVersion), "mininum TLS version for RPC"},
		cli.IntFlag{"snapshot-ttl", defaultOps.SnapshotTTL, "snapshot TTL in hours, 0 to disable"},
		cli.IntFlag{"snapshot-space-percent", defaultOps.SnapshotSpacePercent, "percent of tenant volume size that is assumed to be needed to create a snapshot"},
		cli.StringFlag{"controller-binary", defaultOps.ControllerBinary, "path to the container controller binary"},
		cli.StringFlag{"log-driver", defaultOps.DockerLogDriver, "log driver for docker containers"},
		cli.StringSliceFlag{"log-config", convertToStringSlice(defaultOps.DockerLogConfigList), "comma-separated list of key=value settings for docker log driver"},

		cli.IntFlag{"ui-poll-frequency", defaultOps.UIPollFrequency, "frequency in seconds that the UI polls serviced for changes"},
		cli.IntFlag{"storage-stats-update-interval", defaultOps.StorageStatsUpdateInterval, "frequency in seconds that the thin pool usage will be analyzed"},
		cli.IntFlag{"zk-session-timeout", defaultOps.ZKSessionTimeout, "zookeeper session timeout in seconds"},
		cli.IntFlag{"zk-connection-timeout", defaultOps.ZKConnectTimeout, "zookeeper connection timeout in seconds"},
		cli.IntFlag{"zk-per-host-connect-delay", defaultOps.ZKPerHostConnectDelay, "zookeeper per-host delay in seconds"},
		cli.IntFlag{"zk-reconnect-start-delay", defaultOps.ZKReconnectStartDelay, "zookeeper initial reconnect delay in seconds"},
		cli.IntFlag{"zk-reconnect-max-delay", defaultOps.ZKReconnectMaxDelay, "zookeeper max recoonect delay in seconds"},
		cli.IntFlag{"auth-token-expiry", defaultOps.TokenExpiration, "authentication token expiration in seconds"},
		cli.StringFlag{"conntrack-flush", defaultOps.ConntrackFlush, "whether to flush the conntrack table when a service with an assigned IP is started"},
		cli.IntFlag{"service-run-level-timeout", defaultOps.ServiceRunLevelTimeout, "max time in seconds to wait for services to start/stop before moving on to services at the next run level"},

		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
		cli.BoolFlag{"alsologtostderr", "log to standard error as well as files"},
		cli.StringFlag{"logstashurl", defaultOps.LogstashURL, "logstash url and port"},
		cli.StringFlag{"logstash-es", defaultOps.LogstashES, "host and port for logstash elastic search"},
		cli.IntFlag{"logstash-max-days", defaultOps.LogstashMaxDays, "days to keep Logstash data"},
		cli.IntFlag{"logstash-max-size", defaultOps.LogstashMaxSize, "max size of Logstash data to keep in gigabytes"},

		cli.IntFlag{"storage-report-interval", defaultOps.StorageReportInterval, "frequency in seconds to report storage stats to opentsdb"},
		cli.IntFlag{"storage-metric-monitor-window", defaultOps.StorageMetricMonitorWindow, "the amount of time in seconds for which serviced will consider storage availability metrics in order to predict future availability"},
		cli.IntFlag{"storage-lookahead-period", defaultOps.StorageLookaheadPeriod, "the amount of time in the future in seconds serviced should predict storage availability for the purposes of emergency shutdown"},
		cli.StringFlag{"storage-min-free", string(defaultOps.StorageMinimumFreeSpace), "the amount of space the emergency shutdown algorithm should reserve when deciding to shut down"},

		cli.IntFlag{"logstash-cycle-time", defaultOps.LogstashCycleTime, "logstash purging cycle time in hours"},
		cli.IntFlag{"v", defaultOps.Verbosity, "log level for V logs"},
		cli.StringFlag{"stderrthreshold", "", "logs at or above this threshold go to stderr"},
		cli.StringFlag{"vmodule", "", "comma-separated list of pattern=N settings for file-filtered logging"},
		cli.StringFlag{"log_backtrace_at", "", "when logging hits line file:N, emit a stack trace"},
		cli.StringFlag{"config-file", "/etc/default/serviced", "path to config"},
		cli.StringFlag{"allow-loop-back", defaultOps.AllowLoopBack, "allow loop-back device with devicemapper"},
		cli.StringFlag{"backup-min-overhead", defaultOps.BackupMinOverhead, "Minimum free space to allow when calculating backup estimates"},
		cli.Float64Flag{"backup-estimated-compression", defaultOps.BackupEstimatedCompression, "Estimate of compression rate to use when calculating backup estimates"},
		cli.StringFlag{"auth0-domain", defaultOps.Auth0Domain, "Domain configured for tenant in Auth0. Ref: https://auth0.com/docs/getting-started/the-basics#domain"},
		cli.StringFlag{"auth0-audience", defaultOps.Auth0Audience, "Audience configured for application (?) in Auth0."},
		cli.StringSliceFlag{"auth0-group", convertToStringSlice(defaultOps.Auth0Group), "Group(s) configured for application in Auth0. A comma-separated list."},
		cli.StringFlag{"auth0-client-id", defaultOps.Auth0ClientID, "Client ID of Auth0 application"},
		cli.StringFlag{"auth0-scope", defaultOps.Auth0Scope, "Scope to request in Auth0"},
		cli.StringFlag{"keyproxy-json-server", defaultOps.KeyProxyJsonServer, "URL for API key server (cc auth token endpoint)"},
		cli.StringFlag{"keyproxy-listen-port", defaultOps.KeyProxyListenPort, "Port for API key proxy to listen on"},
		cli.BoolFlag{"no-prefix-match", "Make matches on SERVICEID by name strictly 'ends-with' rather than 'contains'"},
	}

	c.initVersion()
	c.initPool()
	c.initConfig()
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
	c.initKey()
	c.initDebug()

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	if err := c.app.Run(args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
	}
}

// cmdInit is executed before EVERY CLI command/subcommand. Any messages output by this
// method are shown to the CLI user. If this method returns an error, then CLI
// processing is halted,
//
// NOTE: Neither this routine, nor the methods it calls, can use glog to report problems.
//       Otherwise, the unit-tests with "-race" will fail.
func (c *ServicedCli) cmdInit(ctx *cli.Context) error {
	options := getRuntimeOptions(c.config, ctx)
	if err := api.ValidateCommonOptions(options); err != nil {
		fmt.Fprintf(os.Stderr, "Invalid option(s) found: %s\n", err)
		return err
	}

	config.LoadOptions(options)

	// Set logging options
	if err := setLogging(&options, ctx, c.logControl); err != nil {
		fmt.Fprintf(os.Stderr, "Unable to set logging options: %s\n", err)
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

// Get all runtime options as a combination of default values, environment variable settings and
// command line overrides.
func getRuntimeOptions(cfg utils.ConfigReader, ctx *cli.Context) config.Options {
	options := config.Options{
		GCloud:                     cfg.BoolVal("GCLOUD", false),
		StartZK:                    cfg.BoolVal("START_ZK", true),
		StartAPIKeyProxy:           cfg.BoolVal("START_API_KEY_PROXY", false),
		BigTableMetrics:            cfg.BoolVal("BIGTABLE_METRICS", false),
		DockerRegistry:             ctx.GlobalString("docker-registry"),
		NFSClient:                  ctx.GlobalString("nfs-client"),
		Endpoint:                   ctx.GlobalString("endpoint"),
		StaticIPs:                  ctx.GlobalStringSlice("static-ip"),
		UIPort:                     service.ScrubPortString(ctx.GlobalString("uiport")),
		RPCPort:                    fmt.Sprintf("%d", ctx.GlobalInt("listen")),
		Listen:                     fmt.Sprintf(":%d", ctx.GlobalInt("listen")),
		DockerDNS:                  ctx.GlobalStringSlice("docker-dns"),
		Master:                     ctx.GlobalBool("master"),
		MuxPort:                    ctx.GlobalInt("mux"),
		MuxDisableTLS:              ctx.GlobalString("mux-disable-tls"),
		MUXTLSCiphers:              ctx.GlobalStringSlice("mux-tls-ciphers"),
		MUXTLSMinVersion:           ctx.GlobalString("mux-tls-min-version"),
		HomePath:                   api.GetDefaultOptions(cfg).HomePath,
		VolumesPath:                ctx.GlobalString("volumes-path"),
		IsvcsPath:                  ctx.GlobalString("isvcs-path"),
		BackupsPath:                ctx.GlobalString("backups-path"),
		EtcPath:                    ctx.GlobalString("etc-path"),
		LogPath:                    ctx.GlobalString("log-path"),
		KeyPEMFile:                 ctx.GlobalString("keyfile"),
		CertPEMFile:                ctx.GlobalString("certfile"),
		Zookeepers:                 ctx.GlobalStringSlice("zk"),
		Mount:                      ctx.GlobalStringSlice("mount"),
		HostAliases:                ctx.GlobalStringSlice("alias"),
		ESStartupTimeout:           ctx.GlobalInt("es-startup-timeout"),
		ReportStats:                ctx.GlobalBool("report-stats"),
		HostStats:                  ctx.GlobalString("host-stats"),
		StatsPeriod:                ctx.GlobalInt("stats-period"),
		MCUsername:                 ctx.GlobalString("mc-username"),
		MCPasswd:                   ctx.GlobalString("mc-password"),
		Verbosity:                  ctx.GlobalInt("v"),
		CPUProfile:                 ctx.GlobalString("cpuprofile"),
		MaxContainerAge:            ctx.GlobalInt("max-container-age"),
		MaxDFSTimeout:              ctx.GlobalInt("max-dfs-timeout"),
		VirtualAddressSubnet:       ctx.GlobalString("virtual-address-subnet"),
		MasterPoolID:               ctx.GlobalString("master-pool-id"),
		OutboundIP:                 ctx.GlobalString("outbound"),
		LogstashES:                 ctx.GlobalString("logstash-es"),
		LogstashMaxDays:            ctx.GlobalInt("logstash-max-days"),
		LogstashMaxSize:            ctx.GlobalInt("logstash-max-size"),
		LogstashCycleTime:          ctx.GlobalInt("logstash-cycle-time"),
		LogstashURL:                ctx.GlobalString("logstashurl"),
		DebugPort:                  ctx.GlobalInt("debug-port"),
		AdminGroup:                 ctx.GlobalString("admin-group"),
		MaxRPCClients:              ctx.GlobalInt("max-rpc-clients"),
		RPCDialTimeout:             ctx.GlobalInt("rpc-dial-timeout"),
		RPCCertVerify:              ctx.GlobalString("rpc-cert-verify"),
		RPCDisableTLS:              ctx.GlobalString("rpc-disable-tls"),
		RPCTLSCiphers:              ctx.GlobalStringSlice("rpc-tls-ciphers"),
		RPCTLSMinVersion:           ctx.GlobalString("rpc-tls-min-version"),
		SnapshotTTL:                ctx.GlobalInt("snapshot-ttl"),
		SnapshotSpacePercent:       ctx.GlobalInt("snapshot-space-percent"),
		StorageArgs:                ctx.GlobalStringSlice("storage-opts"),
		ControllerBinary:           ctx.GlobalString("controller-binary"),
		IsvcsENV:                   ctx.GlobalStringSlice("isvcs-env"),
		StartISVCS:                 ctx.GlobalStringSlice("isvcs-start"),
		IsvcsZKID:                  ctx.GlobalInt("isvcs-zk-id"),
		IsvcsZKQuorum:              ctx.GlobalStringSlice("isvcs-zk-quorum"),
		IsvcsZKUsername:            ctx.GlobalString("isvcs-zk-username"),
		IsvcsZKPasswd:              ctx.GlobalString("isvcs-zk-passwd"),
		TLSCiphers:                 ctx.GlobalStringSlice("tls-ciphers"),
		TLSMinVersion:              ctx.GlobalString("tls-min-version"),
		DockerLogDriver:            ctx.GlobalString("log-driver"),
		DockerLogConfigList:        ctx.GlobalStringSlice("log-config"),
		AllowLoopBack:              ctx.GlobalString("allow-loop-back"),
		UIPollFrequency:            ctx.GlobalInt("ui-poll-frequency"),
		StorageStatsUpdateInterval: ctx.GlobalInt("storage-stats-update-interval"),
		StorageReportInterval:      ctx.GlobalInt("storage-report-interval"),
		ZKSessionTimeout:           ctx.GlobalInt("zk-session-timeout"),
		ZKConnectTimeout:           ctx.GlobalInt("zk-connection-timeout"),
		ZKPerHostConnectDelay:      ctx.GlobalInt("zk-per-host-connect-delay"),
		ZKReconnectStartDelay:      ctx.GlobalInt("zk-reconnect-start-delay"),
		ZKReconnectMaxDelay:        ctx.GlobalInt("zk-reconnect-max-delay"),
		TokenExpiration:            ctx.GlobalInt("auth-token-expiry"),
		ConntrackFlush:             ctx.GlobalString("conntrack-flush"),
		ServiceRunLevelTimeout:     ctx.GlobalInt("service-run-level-timeout"),
		StorageMetricMonitorWindow: ctx.GlobalInt("storage-metric-monitor-window"),
		StorageLookaheadPeriod:     ctx.GlobalInt("storage-lookahead-period"),
		StorageMinimumFreeSpace:    ctx.GlobalString("storage-min-free"),
		BackupEstimatedCompression: ctx.Float64("backup-estimated-compression"),
		BackupMinOverhead:          ctx.String("backup-min-overhead"),
		Auth0Domain:                ctx.String("auth0-domain"),
		Auth0Audience:              ctx.String("auth0-audience"),
		Auth0Group:                 ctx.GlobalStringSlice("auth0-group"),
		Auth0ClientID:              ctx.String("auth0-client-id"),
		Auth0Scope:                 ctx.String("auth0-scope"),
		KeyProxyJsonServer:         ctx.String("keyproxy-json-server"),
		KeyProxyListenPort:         ctx.String("keyproxy-listen-port"),
	}

	// Long story, but due to the way codegangsta handles bools and the way we start system services vs
	// zendev, we need to double-check the environment variables for Master/Agent after all option
	// initialization has been done
	if cfg.StringVal("MASTER", "") == "1" {
		options.Master = true
	}

	// When using the 'serviced server' command, we should always run as an agent
	if ctx.Args().First() == "server" {
		options.Agent = true
	}

	if options.Master {
		fstype := ctx.GlobalString("fstype")
		options.FSType = volume.DriverType(fstype)
	} else {
		options.FSType = volume.DriverTypeNFS
		if options.NFSClient == "0" {
			options.StorageArgs = append(options.StorageArgs, nfs.NetworkDisabled)
		}
	}

	options.Endpoint = getEndpoint(options)

	// Set the logging configuration filename.
	// This is handled in a non-standard way for two reasons:
	// 1) It needs to be set by an environment variable, but not a CLI flag.  This
	//    is so we don't add too many arcane arguemtns to the CLI.
	// 2) The default value differs depending on whther we are running a server or
	//    CLI command.
	logConfigPath := cfg.StringVal("LOG_CONFIG", "")
	if logConfigPath == "" {
		var filename string
		if ctx.Args().First() == "server" {
			filename = "logconfig-server.yaml"
		} else {
			filename = "logconfig-cli.yaml"
		}
		logConfigPath = filepath.Join(options.EtcPath, filename)
	}
	options.LogConfigFilename = logConfigPath

	return options
}

// getEndpoint gets the endpoint to use if the user did not specify one.
// Takes other configuration options into account while determining the default.
//
// TODO: This method is eerily similar to logic in api.ValidateServerOptions(). The two should be reconciled
//       at some point to avoid duplicate/inconsistent code
func getEndpoint(options config.Options) string {
	// Not printing anything in here because it shows up in help, version, etc.
	endpoint := options.Endpoint
	if len(endpoint) == 0 {
		if options.Master || !options.Agent {
			// Master has multiple backup sources: OUTBOUND_IP or query network configuration.
			// No role probably means user is running us on the Master and assumes we know how
			// to connect to ourselves.
			if len(options.OutboundIP) > 0 {
				endpoint = fmt.Sprintf("%s:%s", options.OutboundIP, options.RPCPort)
			} else if ip, err := utils.GetIPAddress(); err == nil {
				endpoint = fmt.Sprintf("%s:%s", ip, options.RPCPort)
			}
		} else {
			// On pure Agent, ENDPOINT is required to know where Master is (we can't just guess)
		}
	}
	return endpoint
}

func setLogging(options *config.Options, ctx *cli.Context, logControl logging.LogControl) error {
	// Rather than immediately returning from the function on error, keep track
	// of the errors and continue.  This allows us to process all arguments,
	// start watchers, etc.
	var errors []error

	if err := logControl.ApplyConfigFromFile(options.LogConfigFilename); err != nil {
		errors = append(errors, err)
	}
	go logControl.WatchConfigFile(options.LogConfigFilename)

	// Set glog verbosity.  Map the glog verbosity level onto logrus levels as best
	// we can, so that the verbosity for both can be controlled with a single argument.
	if ctx.IsSet("v") {
		verbosity := ctx.GlobalInt("v")
		logControl.SetVerbosity(verbosity)
		switch verbosity {
		case 0:
			logControl.SetLevel(logrus.DebugLevel)
		case 1:
			logControl.SetLevel(logrus.InfoLevel)
		case 2:
			logControl.SetLevel(logrus.WarnLevel)
		default:
			logControl.SetLevel(logrus.ErrorLevel)
		}
	}

	if ctx.IsSet("logtostderr") {
		logControl.SetToStderr(ctx.GlobalBool("logtostderr"))
	}

	if ctx.IsSet("alsologtostderr") {
		logControl.SetAlsoToStderr(ctx.GlobalBool("alsologtostderr"))
	}

	if ctx.IsSet("stderrthreshold") {
		if err := logControl.SetStderrThreshold(ctx.GlobalString("stderrthreshold")); err != nil {
			errors = append(errors, err)
		}
	}
	if ctx.IsSet("vmodule") {
		if err := logControl.SetVModule(ctx.GlobalString("vmodule")); err != nil {
			errors = append(errors, err)
		}
	}

	if ctx.IsSet("log_backtrace_at") {
		if err := logControl.SetTraceLocation(ctx.GlobalString("log_backtrace_at")); err != nil {
			errors = append(errors, err)
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
			// set up debug settings
			verbosity := 2
			level := logrus.DebugLevel
			// if we're already not at Info, reset to Info
			if logControl.GetVerbosity() != 0 {
				verbosity = 0
				level = logrus.InfoLevel
			}
			logControl.SetVerbosity(verbosity)
			logControl.SetLevel(level)
			log.WithFields(logrus.Fields{
				"glog-verbosity": verbosity,
				"logrus-level":   level,
			}).Info("Changed logging level")
		}
	}()

	if len(errors) == 0 {
		return nil
	} else {
		// not technically correct, but realistically we only care about the first error
		return errors[0]
	}
}

func init() {
	// Change the representation of the version flag
	cli.VersionFlag = cli.BoolFlag{"version", "print the version"}
}

func convertToStringSlice(list []string) *cli.StringSlice {
	slice := cli.StringSlice(list)
	return &slice
}
