package cmd

import (
	"fmt"
	"os"

	"github.com/zenoss/cli"
)

// Initializer for serviced backup and serviced restore
func (c *ServicedCli) initBackup() {
	c.app.Commands = append(
		c.app.Commands,
		cli.Command{
			Name:        "backup",
			Usage:       "Dump all templates and services to a tgz file",
			Description: "serviced service backup DIRPATH",
			Action:      c.cmdBackup,
		},
		cli.Command{
			Name:        "restore",
			Usage:       "Restore services from a tgz file",
			Description: "serviced service restore FILEPATH",
			Action:      c.cmdRestore,
		},
	)
}

// serviced service backup DIRPATH
func (c *ServicedCli) cmdBackup(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "backup")
		return
	}

	if path, err := c.driver.Backup(args[0]); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if path == "" {
		fmt.Fprintln(os.Stderr, "received nil path to backup file")
	} else {
		fmt.Println(path)
	}
}

// serviced service restore FILEPATH
func (c *ServicedCli) cmdRestore(ctx *cli.Context) {
	args := ctx.Args()
	if len(args) < 1 {
		fmt.Printf("Incorrect Usage.\n\n")
		cli.ShowCommandHelp(ctx, "restore")
		return
	}

	err := c.driver.Restore(args[0])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
}