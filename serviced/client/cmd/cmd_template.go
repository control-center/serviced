package cmd

import (
	"fmt"

	"github.com/codegangsta/cli"
)

// CmdTemplateList is the command-line interaction for serviced template list
// usage: serviced template list
func CmdTemplateList(c *cli.Context) {
	fmt.Println("serviced template list")
}

// CmdTemplateAdd is the command-line interaction for serviced template add
// usage: serviced template add
func CmdTemplateAdd(c *cli.Context) {
	fmt.Println("serviced template add")
}

// CmdTemplateRemove is the command-line interaction for serviced template remove
// usage: serviced template remove TEMPLATEID
func CmdTemplateRemove(c *cli.Context) {
	fmt.Println("serviced template remove TEMPLATEID")
}

// CmdTemplateDeploy is the command-line interaction for serviced template deploy
// usage: serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]
func CmdTemplateDeploy(c *cli.Context) {
	fmt.Println("serviced template deploy TEMPLATEID POOLID DEPLOYMENTID [--manual-assign-ips]")
}

// CmdTemplateCompile is the command-line interaction for serviced template compile
// usage: serviced template compile DIRPATH [[--map IMAGE,IMAGE] ...]
func CmdTemplateCompile(c *cli.Context) {
	fmt.Println("serviced template compile DIRPATH [[--map IMAGE,IMAGE] ...]")
}