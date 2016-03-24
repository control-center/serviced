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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/cli/api"
	template "github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/servicedversion"
)

// initTemplate is the initializer for serviced template
func (c *ServicedCli) initTemplate() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "template",
		Usage:       "Administers templates",
		Description: "",
		Subcommands: []cli.Command{
			{
				Name:         "list",
				Usage:        "Lists all templates",
				Description:  "serviced template list [TEMPLATEID]",
				BashComplete: c.printTemplatesFirst,
				Action:       c.cmdTemplateList,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "verbose, v",
						Usage: "Show JSON format",
					},
					cli.StringFlag{
						Name:  "show-fields",
						Value: "TemplateID,Name,Description",
						Usage: "Comma-delimited list describing which fields to display",
					},
				},
			}, {
				Name:        "add",
				Usage:       "Add a new template",
				Description: "serviced template add FILE",
				Action:      c.cmdTemplateAdd,
			}, {
				Name:         "remove",
				ShortName:    "rm",
				Usage:        "Remove an existing template",
				Description:  "serviced template remove TEMPLATEID ...",
				BashComplete: c.printTemplatesAll,
				Action:       c.cmdTemplateRemove,
			}, {
				Name:         "deploy",
				Usage:        "Deploys a template's services to a pool",
				Description:  "serviced template deploy TEMPLATEID POOLID DEPLOYMENTID",
				BashComplete: c.printTemplateDeploy,
				Action:       c.cmdTemplateDeploy,
				Flags: []cli.Flag{
					cli.BoolFlag{
						Name:  "manual-assign-ips",
						Usage: "Manually assign IP addresses",
					},
				},
			}, {
				Name:        "compile",
				Usage:       "Convert a directory of service definitions into a template",
				Description: "serviced template compile PATH",
				Action:      c.cmdTemplateCompile,
				Flags: []cli.Flag{
					cli.GenericFlag{
						Name:  "map",
						Value: &api.ImageMap{},
						Usage: "Map a given image name to another (e.g. -map zenoss/zenoss5x:latest,quay.io/zenoss-core:alpha2)",
					},
				},
			},
		},
	})
}

// Returns a list of all available template IDs
func (c *ServicedCli) templates() (data []string) {
	templates, err := c.driver.GetServiceTemplates()
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
func (c *ServicedCli) printTemplatesAll(ctx *cli.Context) {
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
		if template, err := c.driver.GetServiceTemplate(templateID); err != nil {
			fmt.Fprintln(os.Stderr, err)
		} else if template == nil {
			fmt.Fprintln(os.Stderr, "template not found")
		} else if jsonTemplate, err := json.MarshalIndent(template, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal template: %s\n", err)
		} else {
			fmt.Println(string(jsonTemplate))
		}
		return
	}

	templates, err := c.driver.GetServiceTemplates()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
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
		t := NewTable(ctx.String("show-fields"))
		t.Padding = 6
		for _, tmp := range templates {
			t.AddRow(map[string]interface{}{
				"TemplateID":  tmp.ID,
				"Name":        tmp.Name,
				"Description": tmp.Description,
			})
		}
		t.Print()
	}
}

// serviced template add TEMPLATE
func (c *ServicedCli) cmdTemplateAdd(ctx *cli.Context) {
	var input *os.File

	if filepath := ctx.Args().First(); filepath != "" {
		var err error
		if input, err = os.Open(filepath); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return
		}
		defer input.Close()
	} else {
		input = os.Stdin
	}

	if template, err := c.driver.AddServiceTemplate(input); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if template == nil {
		fmt.Fprintln(os.Stderr, "received nil template")
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
		if err := c.driver.RemoveServiceTemplate(id); err != nil {
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
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "deploy")
		return
	}

	cfg := api.DeployTemplateConfig{
		ID:              args[0],
		PoolID:          args[1],
		DeploymentID:    args[2],
		ManualAssignIPs: ctx.Bool("manual-assign-ips"),
	}

	fmt.Fprintln(os.Stderr, "Deploying template - please wait...")
	if svcs, err := c.driver.DeployServiceTemplate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if svcs == nil {
		fmt.Fprintln(os.Stderr, "received nil service definition")
	} else {
		for _, svc := range svcs {
			fmt.Println(svc.ID)
		}
	}
}

type metaTemplate struct {
	template.ServiceTemplate
	ServicedVersion servicedversion.ServicedVersion
	TemplateVersion map[string]string
}

// serviced template compile DIR [[--map IMAGE,IMAGE] ...]
func (c *ServicedCli) cmdTemplateCompile(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "compile")
		return
	}

	cfg := api.CompileTemplateConfig{
		Dir: args[0],
		Map: *ctx.Generic("map").(*api.ImageMap),
	}

	if template, err := c.driver.CompileServiceTemplate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if template == nil {
		fmt.Fprintln(os.Stderr, "received nil template")
	} else {
		cmd := fmt.Sprintf("cd %s && git rev-parse HEAD", args[0])
		commit, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			commit = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git config --get remote.origin.url", args[0])
		repo, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			repo = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git rev-parse --abbrev-ref HEAD", args[0])
		branch, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			branch = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git describe --always", args[0])
		tag, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			tag = []byte("unknown")
		}
		templateVersion := map[string]string{
			"repo":   strings.Trim(string(repo), "\n"),
			"branch": strings.Trim(string(branch), "\n"),
			"tag":    strings.Trim(string(tag), "\n"),
			"commit": strings.Trim(string(commit), "\n"),
		}
		mTemplate := metaTemplate{*template, servicedversion.GetVersion(), templateVersion}
		jsonTemplate, err := json.MarshalIndent(mTemplate, " ", "  ")
		if err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal template: %s\n", err)
		} else {
			fmt.Println(string(jsonTemplate))
		}
	}
}
