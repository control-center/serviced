package cmd

import (
	"github.com/zenoss/serviced/serviced/api"
)

func ExampleServicedCli_cmdTemplateList() {
	New(api.New()).Run([]string{"serviced", "template", "list"})

	// Output:
	// serviced template list
}

func ExampleServicedCli_cmdTemplateAdd() {
	New(api.New()).Run([]string{"serviced", "template", "add"})

	// Output:
	// serviced template add
}

func ExampleServicedCli_cmdTemplateRemove() {
	New(api.New()).Run([]string{"serviced", "template", "remove"})
	New(api.New()).Run([]string{"serviced", "template", "rm"})

	// Output:
	// serviced template remove TEMPLATEID
	// serviced template remove TEMPLATEID
}

func ExampleServicedCli_cmdTemplateDeploy() {
	New(api.New()).Run([]string{"serviced", "template", "deploy"})

	// Output:
	// serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]
}

func ExampleServicedCli_cmdTemplateCompile() {
	New(api.New()).Run([]string{"serviced", "template", "compile"})

	// Output:
	// serviced template compile DIRPATH [[--map IMAGE,IMAGE] ...]
}
