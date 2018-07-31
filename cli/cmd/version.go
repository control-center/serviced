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
	"github.com/codegangsta/cli"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/servicedversion"

	"encoding/json"
	"fmt"
	"os"
	"github.com/control-center/serviced/config"
)

// Initializer for serviced version
func (c *ServicedCli) initVersion() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "version",
		Usage:       "shows version information",
		Description: "",
		Action:      c.cmdVersion,
		Flags: []cli.Flag{
			cli.BoolFlag{
				Name:  "verbose, v",
				Usage: "Show JSON format",
			},
		},
	})
}

// serviced version
func (c *ServicedCli) cmdVersion(ctx *cli.Context) {

	var versionInfo = map[string]string{
		"Version":            servicedversion.Version,
		"GoVersion":          servicedversion.GoVersion,
		"Gitcommit":          servicedversion.Gitcommit,
		"Gitbranch":          servicedversion.Gitbranch,
		"Date":               servicedversion.Date,
		"Release":            servicedversion.Release,
		"IsvcsImage":         fmt.Sprintf("%s:%s", isvcs.IMAGE_REPO, isvcs.IMAGE_TAG),
		"IsvcsZKImage":       fmt.Sprintf("%s:%s", isvcs.ZK_IMAGE_REPO, isvcs.ZK_IMAGE_TAG),
	}

	// Only populate the 'IsvcsApiProxyImage' value if the api proxy is configured to start.
	if config.GetOptions().StartAPIKeyProxy {
		versionInfo["IsvcsApiProxyImage"] = fmt.Sprintf("%s:%s", isvcs.API_KEY_PROXY_REPO, isvcs.API_KEY_PROXY_TAG)
	}

	if ctx.Bool("verbose") {
		if jsonVersion, err := json.MarshalIndent(versionInfo, " ", "  "); err != nil {
			fmt.Fprintf(os.Stderr, "failed to marshal version info: %s", err)
			os.Exit(1)
		} else {
			fmt.Println(string(jsonVersion))
		}
	} else {
		fmt.Printf("Version:    %s\n", versionInfo["Version"])
		fmt.Printf("GoVersion:  %s\n", versionInfo["GoVersion"])
		fmt.Printf("Gitcommit:  %s\n", versionInfo["Gitcommit"])
		fmt.Printf("Gitbranch:  %s\n", versionInfo["Gitbranch"])
		fmt.Printf("Date:       %s\n", versionInfo["Date"])
		fmt.Printf("Buildtag:   %s\n", versionInfo["Buildtag"])
		fmt.Printf("Release:    %s\n", versionInfo["Release"])
		images := []string{
			versionInfo["IsvcsImage"], versionInfo["IsvcsZKImage"],
		}
		if image, exists := versionInfo["IsvcsApiProxyImage"]; exists {
			images = append(images, image)
		}
		fmt.Printf("IsvcsImages: %v\n", images)
	}
}
