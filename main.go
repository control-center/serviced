// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package main


import (
	"os"

	"github.com/control-center/serviced/servicedversion"
	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/cli/cmd"
)

var Version string
var Date string
var Gitcommit string
var Gitbranch string
var Giturl string
var Buildtag string


func main() {
	servicedversion.Version = Version
	servicedversion.Date = Date
	servicedversion.Gitcommit = Gitcommit
	servicedversion.Gitbranch = Gitbranch
	servicedversion.Giturl = Giturl
	servicedversion.Buildtag = Buildtag
	cmd.New(api.New()).Run(os.Args)
}
