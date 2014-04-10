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
