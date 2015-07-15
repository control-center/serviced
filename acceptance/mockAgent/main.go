// Copyright 2015 The Serviced Authors.
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
package main

import (
	"os"

	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/servicedversion"
)

// These variables are initialized by compile-time properties defined in the make
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

	newAgent(cli.NewApp()).run(os.Args)
}
