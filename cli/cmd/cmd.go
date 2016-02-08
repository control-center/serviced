// Copyright 2014 The Serviced Authors.
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
	"os/signal"
	"strconv"
	"strings"
	"syscall"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/rpc/rpcutils"
	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/validation"
	"github.com/zenoss/glog"
)

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver       api.API
	app          *cli.App
	exitDisabled bool
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
func configBool(key string, defaultVal bool) bool {
	s := configEnv(key, "")
	if len(s) == 0 {
		return defaultVal
	}

	trues := []string{"1", "true", "t", "yes"}
	if v := strings.ToLower(s); v != "" {
		for _, t := range trues {
			if v == t {
				return true
			}
		}
	}

	falses := []string{"0", "false", "f", "no"}
	if v := strings.ToLower(s); v != "" {
		for _, t := range falses {
			if v == t {
				return false
			}
		}
	}

	return defaultVal
}

func convertToStringSlice(list []string) *cli.StringSlice {
	slice := cli.StringSlice(list)
	return &slice
}

const defaultRPCPort = 4979

func getLocalAgentEndpoint(port int) string {
	ip := configEnv("OUTBOUND_IP", "")
	if ip != "" {
		return fmt.Sprintf("%s:%d", ip, port)
	} else {
		return api.GetAgentIP(port)
	}
}

// New instantiates a new command-line client
func New(driver api.API) *ServicedCli {
	var (
		rpcPort          = configInt("RPC_PORT", defaultRPCPort)
		agentEndpoint    = getLocalAgentEndpoint(rpcPort)
		varPath          = api.GetVarPath()
		esStartupTimeout = configInt("ES_STARTUP_TIMEOUT", isvcs.DEFAULT_ES_STARTUP_TIMEOUT_SECONDS)
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

	defaultDockerRegistry := docker.DEFAULT_REGISTRY
	if hostname, err := os.Hostname(); err == nil {
		defaultDockerRegistry = fmt.Sprintf("%s:5000", hostname)
	}

	zks := cli.StringSlice{}
	if len(configEnv("ZK", "")) > 0 {
		zks = cli.StringSlice(strings.Split(configEnv("ZK", ""), ","))
	}

	isvcs_env := cli.StringSlice{}
	if env := configEnv("ISVCS_ENV", ""); len(env) != 0 {
		isvcs_env = append(isvcs_env, env)
	}
	for i := 0; ; i++ {
		if env := configEnv(fmt.Sprintf("ISVCS_ENV_%d", i), ""); len(env) != 0 {
			isvcs_env = append(isvcs_env, env)
		} else {
			break
		}
	}

	/* TODO: 1.1
	remotezks := cli.StringSlice{}
	if len(configEnv("REMOTE_ZK", "")) > 0 {
		zks = cli.StringSlice(strings.Split(configEnv("REMOTE_ZK", ""), ","))
	}
	*/

	aliases := cli.StringSlice{}
	if len(configEnv("VHOST_ALIASES", "")) > 0 {
		aliases = cli.StringSlice(strings.Split(configEnv("VHOST_ALIASES", ""), ","))
	}

	defaultAdminGroup := "sudo"
	if utils.Platform == utils.Rhel {
		defaultAdminGroup = "wheel"
	}

	defaultTlsCiphers := utils.GetDefaultCiphers()

	c.app.Flags = []cli.Flag{
		cli.StringFlag{"docker-registry", configEnv("DOCKER_REGISTRY", defaultDockerRegistry), "local docker registry to use"},
		cli.StringSliceFlag{"static-ip", &staticIps, "static ips for this agent to advertise"},
		cli.StringFlag{"endpoint", configEnv("ENDPOINT", agentEndpoint), fmt.Sprintf("endpoint for remote serviced (example.com:%d)", defaultRPCPort)},
		cli.StringFlag{"outbound", configEnv("OUTBOUND_IP", ""), "outbound ip address"},
		cli.StringFlag{"uiport", configEnv("UI_PORT", ":443"), "port for ui"},
		cli.StringFlag{"nfs-client", configEnv("NFS_CLIENT", "1"), "establish agent as an nfs client sharing data, 0 to disable"},
		cli.IntFlag{"listen", rpcPort, fmt.Sprintf("rpc port for serviced (%d)", defaultRPCPort)},
		cli.StringSliceFlag{"docker-dns", &dockerDNS, "docker dns configuration used for running containers"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control center service"},
		cli.BoolFlag{"agent", "run in agent mode, i.e., a host in a resource pool"},
		cli.IntFlag{"mux", configInt("MUX_PORT", 22250), "multiplexing port"},
		cli.StringFlag{"var", configEnv("VARPATH", varPath), "path to store serviced data"},
		cli.StringFlag{"keyfile", configEnv("KEY_FILE", ""), "path to private key file (defaults to compiled in private key)"},
		cli.StringFlag{"certfile", configEnv("CERT_FILE", ""), "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringSliceFlag{"zk", &zks, "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181)"},
		// TODO: 1.1
		// cli.StringSliceFlag{"remote-zk", &remotezks, "Specify a zookeeper instance to connect to (e.g. -remote-zk remote:2181)"},
		cli.StringSliceFlag{"mount", &cli.StringSlice{}, "bind mount: DOCKER_IMAGE,HOST_PATH[,CONTAINER_PATH]"},
		cli.StringFlag{"fstype", configEnv("FS_TYPE", "rsync"), "driver for underlying file system"},
		cli.StringSliceFlag{"alias", &aliases, "list of aliases for this host, e.g., localhost"},
		cli.IntFlag{"es-startup-timeout", esStartupTimeout, "time (in seconds) to wait on elasticsearch startup before bailing"},
		cli.IntFlag{"max-container-age", configInt("MAX_CONTAINER_AGE", 60*60*24), "maximum age (seconds) of a stopped container before removing"},
		cli.IntFlag{"max-dfs-timeout", configInt("MAX_DFS_TIMEOUT", 60*5), "max timeout to perform a dfs snapshot"},
		cli.StringFlag{"virtual-address-subnet", configEnv("VIRTUAL_ADDRESS_SUBNET", "10.3"), "/16 subnet for virtual addresses"},
		cli.StringFlag{"master-pool-id", configEnv("MASTER_POOLID", "default"), "master's pool ID"},
		cli.StringFlag{"admin-group", configEnv("ADMIN_GROUP", defaultAdminGroup), "system group that can log in to control center"},
		cli.StringSliceFlag{"tls-ciphers", convertToStringSlice(defaultTlsCiphers), "list of supported tls ciphers"},
		cli.StringFlag{"tls-min-version", string(utils.DefaultTLSMinVersion), "mininum tls version"},
		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", configEnv("STATS_PORT", "127.0.0.1:8443"), "container statistics for host:port"},
		cli.IntFlag{"stats-period", configInt("STATS_PERIOD", 10), "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", "scott", "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", "tiger", "Password for the Zenoss metric consumer"},
		cli.StringFlag{"cpuprofile", "", "write cpu profile to file"},
		cli.StringSliceFlag{"isvcs-env", &isvcs_env, "internal-service environment variable: ISVC:KEY=VAL"},
		cli.IntFlag{"debug-port", configInt("DEBUG_PORT", 6006), "Port on which to listen for profiler connections"},
		cli.IntFlag{"max-rpc-clients", configInt("MAX_RPC_CLIENTS", 3), "max number of rpc clients to an endpoint"},
		cli.IntFlag{"rpc-dial-timeout", configInt("RPC_DIAL_TIMEOUT", 30), "timeout for creating rpc connections"},
		cli.IntFlag{"snapshot-ttl", configInt("SNAPSHOT_TTL", 12), "snapshot TTL in hours, 0 to disable"},

		// Reimplementing GLOG flags :(
		cli.BoolTFlag{"logtostderr", "log to standard error instead of files"},
		cli.BoolFlag{"alsologtostderr", "log to standard error as well as files"},
		cli.StringFlag{"logstashurl", configEnv("LOG_ADDRESS", "127.0.0.1:5042"), "logstash url and port"},
		cli.StringFlag{"logstash-es", configEnv("LOGSTASH_ES", "127.0.0.1:9100"), "host and port for logstash elastic search"},
		cli.IntFlag{"logstash-max-days", configInt("LOGSTASH_MAX_DAYS", 14), "days to keep Logstash data"},
		cli.IntFlag{"logstash-max-size", configInt("LOGSTASH_MAX_SIZE", 10), "max size of Logstash data to keep in gigabytes"},
		cli.IntFlag{"v", configInt("LOG_LEVEL", 0), "log level for V logs"},
		cli.StringFlag{"stderrthreshold", "", "logs at or above this threshold go to stderr"},
		cli.StringFlag{"vmodule", "", "comma-separated list of pattern=N settings for file-filtered logging"},
		cli.StringFlag{"log_backtrace_at", "", "when logging hits line file:N, emit a stack trace"},
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

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	err := c.app.Run(args)
	if err != nil {
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
		VarPath:              ctx.GlobalString("var"),
		KeyPEMFile:           ctx.GlobalString("keyfile"),
		CertPEMFile:          ctx.GlobalString("certfile"),
		Zookeepers:           ctx.GlobalStringSlice("zk"),
		RemoteZookeepers:     ctx.GlobalStringSlice("remote-zk"),
		Mount:                ctx.GlobalStringSlice("mount"),
		FSType:               ctx.GlobalString("fstype"),
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
		TLSCiphers:           ctx.GlobalStringSlice("tls-ciphers"),
		TLSMinVersion:        ctx.GlobalString("tls-min-version"),
	}
	if os.Getenv("SERVICED_MASTER") == "1" {
		options.Master = true
	}
	if os.Getenv("SERVICED_AGENT") == "1" {
		options.Agent = true
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

	if options.Master {
		fmt.Println("This master has been configured to be in pool: " + options.MasterPoolID)
	}

	// Start server mode
	if (options.Master || options.Agent) && len(ctx.Args()) == 0 {
		rpcutils.RPC_CLIENT_SIZE = options.MaxRPCClients
		c.driver.StartServer()
		return fmt.Errorf("running server mode")
	}

	glog.V(2).Infof("setting supported tls ciphers %s", options.TLSCiphers)
	if err := utils.SetCiphers(options.TLSCiphers); err != nil {
		return fmt.Errorf("unable to set TLSCiphers %v", err)
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
	// 0 and 2.  If the log level is anything but 0, we set it to 0, and on
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
