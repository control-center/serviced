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
			BashComplete: c.printTemplatesFirst,

			Args: []string{
				"[TEMPLATEID]",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"verbose, v", "Show JSON format"},
			},
		}, {
			Name:   "add",
			Usage:  "Adds a new template.",
			Action: c.cmdTemplateAdd,

			Args: []string{
				"< TEMPLATE",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"file, f", "Template file name"},
			},
		}, {
			Name:         "remove",
			ShortName:    "rm",
			Usage:        "Removes an existing template.",
			Action:       c.cmdTemplateRemove,
			BashComplete: c.printTemplatesAll,

			Args: []string{
				"TEMPLATEID ...",
			},
		}, {
			Name:         "deploy",
			Usage:        "Deploys template into a given pool.",
			Action:       c.cmdTemplateDeploy,
			BashComplete: c.printTemplateDeploy,

			Args: []string{
				"TEMPLATEID", "POOLID", "DEPLOYMENTID",
			},
			Flags: []cli.Flag{
				cli.BoolFlag{"manual-assign-ips", "Manually assign IP addresses to services requiring an external IP address"},
			},
		}, {
			Name:         "compile",
			Usage:        "Reads a given directory of service definitions to compile to a json struct.",
			Action:       c.cmdTemplateCompile,
			BashComplete: c.printTemplates,

			Args: []string{
				"DIR",
			},
			Flags: []cli.Flag{
				cli.GenericSliceFlag{"map", "Map a given image name to another (e.g. -map zenoss/zenoss5x->quay.io/zenoss-core:alpha2)"},
			},
		},
	}
}

// Returns a list of all available template IDs
func (c *ServicedCli) templates() (data []string) {
	templates, err := c.driver.ListTemplates()
	if err != nil || templates == nil || len(templates) == 0 {
		return
	}

	data = make([]string, len(templates))
	for i, t := range templates {
		data[i] = t.ID
	}

	return
}

// Bash-completion command that prints the list of templates as the first
// argument
func (c *ServicedCli) printTemplatesFirst(ctx *cli.Context) {
	if len(ctx.Args()) > 0 {
		return
	}

	for _, t := range c.templates() {
		fmt.Println(t)
	}
}

// Bash-completion command that prints the command options for
// serviced template deploy
func (c *ServicedCli) printTemplateDeploy(ctx *cli.Context) {
	var output []string

	switch len(ctx.Args()) {
	case 0:
		output = c.templates()
	case 1:
		output = c.pools()
	}

	for _, o := range output {
		fmt.Println(o)
	}
}

// Bash-completion command that prints the list of templates as all arguments
func (c *ServicedCli) printTemplatesAll(ctx cli.Context) {
	args := ctx.Args()

	for _, t := range c.templates() {
		for _, a := range args {
			if t == a {
				goto next
			}
		}
		fmt.Println(t)
	next:
	}
}

// serviced template list [--verbose, -v] [TEMPLATEID]
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

// serviced template add < [TEMPLATE]
func (c *ServicedCli) cmdTemplateAdd(ctx *cli.Context) {
	var input *os.File

	args := ctx.Args()

	if ctx.String("file") != "" {
		if input, err = os.Open(args[0]); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	if template, err := c.driver.AddTemplate(input); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println(template.ID)
	}
}

// serviced template remove TEMPLATEID ...
func (c *ServicedCli) cmdTemplateRemove(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "remove")
		return
	}

	for _, id := range args {
		if err := c.driver.RemoveTemplate(id); err != nil {
			fmt.Fprintf(os.Stderr, "%s: %s\n", id, err)
		} else {
			fmt.Println(id)
		}
	}
}

// serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]
func (c *ServicedCli) cmdTemplateDeploy(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 3 {
		fmt.Println("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "deploy")
		return
	}

	cfg := api.DeployTemplateConfig{
		ID:              args[0],
		PoolID:          args[1],
		DeploymentID:    args[2],
		ManualAssignIPs: ctx.BoolFlag("manual-assign-ips"),
	}

	if err := c.driver.DeployTemplate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else {
		fmt.Println(cfg.DeploymentID)
	}
}

// serviced template compile DIR [[--map IMAGE->IMAGE] ...]
func (c *ServicedCli) cmdTemplateCompile(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Println("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "compile")
		return
	}

	cfg := api.CompileTemplateConfig{
		Dir: args[0],
		Map: ctx.GenericSlice("map"),
	}

	if template, err := c.driver.CompileTemplate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if jsonTemplate, err := json.MarshalTemplate(template, " ", "  "); err != nil {
		fmt.Println(string(jsonTemplate))
	}
}