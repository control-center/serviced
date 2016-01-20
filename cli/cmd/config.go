// Copyright 2016 The Serviced Authors.
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
	"fmt"

	"github.com/codegangsta/cli"
	"sort"
)

// Initializer for serviced config subcommands
func (c *ServicedCli) initConfig() {
	c.app.Commands = append(c.app.Commands, cli.Command{
		Name:        "config",
		Usage:       "Reports on serviced configuration",
		Description: "serviced config",
		Action:      c.cmdConfig,
	})
}

// serviced config
func (c *ServicedCli) cmdConfig(ctx *cli.Context) {
	configValues := c.config.GetConfigValues()

	// build a sorted list of keys
	keys := []string{}
	for key, _ := range(configValues) {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	for _, key := range(keys) {
		entry, _ := configValues[key]
		fmt.Printf("%s=%s\n", entry.Name, entry.Value)
	}
}
