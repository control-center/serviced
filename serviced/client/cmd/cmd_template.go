package cmd

import (
	"fmt"

	"github.com/zenoss/cli"
)

// initTemplate is the initializer for serviced template
func (c *ServicedCli) initTemplate() {
	cmd := c.app.AddSubcommand(cli.Command{
		Name:  "template",
		Usage: "Administers templates.",
	})
	cmd.Commands = []cli.Command{
		{
			Name:   "list",
			Usage:  "Lists all templates.",
			Action: c.cmdTemplateList,
		}, {
			Name:   "add",
			Usage:  "Adds a new template.",
			Action: c.cmdTemplateAdd,
		}, {
			Name:      "remove",
			ShortName: "rm",
			Usage:     "Removes an existing template.",
			Action:    c.cmdTemplateRemove,
		}, {
			Name:   "deploy",
			Usage:  "Deploys template into a given pool.",
			Action: c.cmdTemplateDeploy,
		}, {
			Name:   "compile",
			Usage:  "Reads a given directory of service definitions to compile to a json struct.",
			Action: c.cmdTemplateCompile,
		},
	}
}

// cmdTemplateList is the command-line interaction for serviced template list
// usage: serviced template list
func (c *ServicedCli) cmdTemplateList(ctx *cli.Context) {
	fmt.Println("serviced template list")
}

// cmdTemplateAdd is the command-line interaction for serviced template add
// usage: serviced template add
func (c *ServicedCli) cmdTemplateAdd(ctx *cli.Context) {
	fmt.Println("serviced template add")
}

// cmdTemplateRemove is the command-line interaction for serviced template remove
// usage: serviced template remove TEMPLATEID
func (c *ServicedCli) cmdTemplateRemove(ctx *cli.Context) {
	fmt.Println("serviced template remove TEMPLATEID")
}

// cmdTemplateDeploy is the command-line interaction for serviced template deploy
// usage: serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]
func (c *ServicedCli) cmdTemplateDeploy(ctx *cli.Context) {
	fmt.Println("serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]")
}

// cmdTemplateCompile is the command-line interaction for serviced template compile
// usage: serviced template compile DIRPATH [[--map IMAGE,IMAGE] ...]
func (c *ServicedCli) cmdTemplateCompile(ctx *cli.Context) {
	fmt.Println("serviced template compile DIRPATH [[--map IMAGE,IMAGE] ...]")
}