// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cmd

import (
	"github.com/codegangsta/cli"
	"github.com/zenoss/serviced/servicedversion"

	"fmt"
)

// Initializer for serviced version
func (c *ServicedCli) initVersion() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "version",
		Usage:       "shows version information",
		Description: "",
		Action: c.cmdVersion,
	})
}

// serviced version
func (c *ServicedCli) cmdVersion(ctx *cli.Context) {
	fmt.Printf("Version:   %s\n", servicedversion.Version)
	fmt.Printf("Gitcommit: %s\n", servicedversion.Gitcommit)
	fmt.Printf("Gitbranch: %s\n", servicedversion.Gitbranch)
	fmt.Printf("Giturl:    %s\n", servicedversion.Giturl)
	fmt.Printf("Date:      %s\n", servicedversion.Date)
	fmt.Printf("Buildtag:  %s\n", servicedversion.Buildtag)
}

