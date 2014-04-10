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
			Name:         "list",
			Usage:        "Lists all templates.",
			Action:       c.cmdTemplateList,
			BashComplete: c.printTemplates,
		}, {
			Name:   "add",
			Usage:  "Adds a new template.",
			Action: c.cmdTemplateAdd,
		}, {
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing template.",
			Action:       c.cmdTemplateRemove,
			BashComplete: c.printTemplates,
		}, {
			Name:         "deploy",
			Usage:        "Deploys template into a given pool.",
			Action:       c.cmdTemplateDeploy,
			BashComplete: c.printTemplates,
		}, {
			Name:         "compile",
			Usage:        "Reads a given directory of service definitions to compile to a json struct.",
			Action:       c.cmdTemplateCompile,
			BashComplete: c.printTemplates,
		},
	}
}

// printTemplates is the default completion for each serviced template subcommand
// usage: serviced template COMMAND --generate-bash-completion
func (c *ServicedCli) printTemplates(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	templates, err := c.driver.ListTemplates()
	if err != nil || templates == nil || len(templates) == 0 {
		return
	}

	for _, t := range templates {
		fmt.Println(t.ID)
	}
}

// cmdTemplateList is the command-line interaction for serviced template list
// usage: serviced template list
func (c *ServicedCli) cmdTemplateList(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		templateID := ctx.Args()[0]
		if template, err := c.driver.GetTemplate(templateID); err != nil {
			fmt.Fprintf(os.Stderr, "error trying to retrieve template: %s\n", err)
		} else if template == nil {
			fmt.Fprintln(os.Stderr, "template not found")
		} else if jsonTemplate, err := json.MarshalIndent(template, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal template: %s\n", err)
		} else {
			fmt.Println(string(jsonTemplate))
		}
		return
	}

	templates, err := c.driver.ListTemplates()
	if err != nil {
		fmt.Fprintf(os.Stderr, "error trying to retrieve templates: %s\n", err)
		return
	} else if templates == nil || len(templates) == 0 {
		fmt.Fprintln(os.Stderr, "no templates found")
		return
	}

	if ctx.Bool("verbose") {
		if jsonTemplate, err := json.MarshalIndent(templates, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal template list: %s\n", err)
		} else {
			fmt.Println(string(jsonTemplate))
		}
	} else {
		tableTemplate := newTable(0, 8, 2)
		tableTemplate.PrintRow("TEMPLATE ID", "NAME", "DESCRIPTION")
		for _, t := range templates {
			tableTemplate.PrintRow(t.ID, t.Name, t.Description)
		}
		table.Template.Flush()
	}
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