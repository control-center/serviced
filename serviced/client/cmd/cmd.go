package cmd

import (
	"fmt"

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
	c := &ServicedCli{
		driver: driver,
		app:    cli.NewApp(),
	}

	c.app.Name = "serviced"
	c.app.Usage = "A container-based management system"
	c.app.Flags = []cli.Flag{
		cli.BoolFlag{"cmp", ""},
	}
	c.app.Action = cmdDefault

	c.initProxy()
	c.initPool()
	c.initHost()
	c.initTemplate()
	c.initService()
	c.initSnapshot()

	return c
}

// Run builds the command-line interface for serviced and runs.
func (c *ServicedCli) Run(args []string) {
	c.app.Run(args)
}

func cmdDefault(c *cli.Context) {
	if c.Bool("cmp") {
		for _, command := range c.App.Commands {
			fmt.Println(command.Name)
			if command.ShortName != "" {
				fmt.Println(command.ShortName)
			}
		}
	} else if c.Args().First() != "" {
		cli.ShowCommandHelp(c, c.Args().First())
	} else {
		cli.ShowAppHelp(c)
	}
}
