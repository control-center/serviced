package cmd

import (
	"github.com/zenoss/cli"
	"github.com/zenoss/serviced/serviced/client/api"
)

// ServicedCli is the client ui for serviced
type ServicedCli struct {
	driver api.API
	app    *cli.App
}

// New instantiates a new command-line client
func New(driver api.API) *ServicedCli {

	cli.CommandHelpTemplate = `NAME:
    {{.Name}} - {{.Usage}}                                                         
                                                                                     
USAGE:                                                                            
    command {{.Name}} [command options] {{range .Args}}{{.}} {{end}}               
	                                                                                    
DESCRIPTION:                                                                      
	{{.Description}}                                                               
		                                                                                   
OPTIONS:                                                                          
	{{range .Flags}}{{.}}                                                          
	{{end}}                                                                        
`

	c := &ServicedCli{
		driver: driver,
		app:    cli.NewApp(),
	}

	c.app.Name = "serviced"
	c.app.Usage = "A container-based management system"
	c.app.Before = c.cmdInit
	c.app.Flags = []cli.Flag{
		cli.StringFlag{"port", api.GetAgentIP() + ":4979", "port for remote serviced (example.com:8080)"},
		cli.StringFlag{"listen", ":4979", "port for local serviced (example.com:8080)"},
		cli.StringFlag{"docker-dns", api.GetDockerDNS(), "docker dns configuration used for running containers (comma separated list)"},
		cli.BoolFlag{"master", "run in master mode, i.e., the control plane service"},
		cli.BoolFlag{"agent", "run in agent mode, i.e., a host in a resource pool"},
		cli.IntFlag{"mux", 22250, "multiplexing port"},
		cli.BoolFlag{"tls", "enable TLS"},
		cli.StringFlag{"var", api.GetVarpath(), "path to store serviced data"},
		cli.StringFlag{"keyfile", "", "path to private key file (defaults to compiled in private key)"},
		cli.StringFlag{"certfile", "", "path to public certificate file (defaults to compiled in public cert)"},
		cli.StringSliceFlag{"zk", "", "Specify a zookeeper instance to connect to (e.g. -zk localhost:2181)"},
		cli.GenericFlag{"mount", &api.VolumeMap{}, "bind mount: dockerImage,hostPath,containerPath"},
		cli.StringFlag{"vfs", "rsync", "filesystem for container volumes"},
		cli.StringSliceFlag{"alias", "list of aliases for this host, e.g., localhost"},
		cli.IntVar{"es-start-timeout", api.GetESStartupTimeout(), "time to wait on elasticsearch startup before bailing"},

		cli.BoolTFlag{"report-stats", "report container statistics"},
		cli.StringFlag{"host-stats", "127.0.0.1:8443", "container statistics for host:port"},
		cli.IntFlag{"stats-period", 60, "Period (seconds) for container statistics reporting"},
		cli.StringFlag{"mc-username", "scott", "Username for Zenoss metric consumer"},
		cli.StringFlag{"mc-password", "tiger", "Password for the Zenoss metric consumer"},
	}

	c.initPool()
	c.initHost()
	c.initTemplate()
	c.initService()
	c.initSnapshot()

	return c
}

// Builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	c.app.Run(args)
}

// Starts the server if no subcommands are called
func (c *ServicedCli) cmdInit(ctx *cli.Context) error {

	options := api.Options{
		Port:             ctx.GlobalString("port"),
		Listen:           ctx.GlobalString("listen"),
		DockerDNS:        ctx.GlobalString("docker-dns"),
		Master:           ctx.GlobalBool("master"),
		Agent:            ctx.GlobalBool("agent"),
		MuxPort:          ctx.GlobalInt("mux"),
		TLS:              ctx.GlobalBool("tls"),
		VarPath:          ctx.GlobalString("var"),
		KeyPEMFile:       ctx.GlobalString("keyfile"),
		CertPEMFile:      ctx.GlobalString("certfile"),
		Zookeepers:       ctx.GlobalStringSlice("zk"),
		Mount:            ctx.GlobalGeneric("mount"),
		VFS:              ctx.GlobalString("vfs"),
		Alias:            ctx.GlobalStringSlice("alias"),
		ESStartupTimeout: ctx.GlobalInt("es-startup-timeout"),
		ReportStats:      ctx.GlobalBool("report-stats"),
		HostStats:        ctx.GlobalString("host-stats"),
		StatsPeriod:      ctx.GlobalInt("stats-period"),
		MCUsername:       ctx.GlobalString("mc-username"),
		MCPassword:       ctx.GlobalString("mc-password"),
	}

	// Start server mode
	if (options.master || options.agent) && len(ctx.Args()) == 0 {
		c.driver.StartServer(options)
		return fmt.Errorf("server started")
	}

	return nil
}
