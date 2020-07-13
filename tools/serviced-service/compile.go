// Copyright 2020 The Serviced Authors.
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
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/jessevdk/go-flags"

	"github.com/control-center/serviced/cli/api"
	template "github.com/control-center/serviced/domain/servicetemplate"
	version "github.com/control-center/serviced/servicedversion"
)

// Compile is the subcommand for compiling a directory of service templates
type Compile struct {
	MapArgs     []string `long:"map" description:"Map a given image name to another"`
	ExcludeArgs []string `long:"exclude" short:"x" description:"Exclude service or service folder"`
	Args        struct {
		Path flags.Filename `positional-arg-name:"PATH" description:"Path to template directory"`
	} `positional-args:"yes" required:"yes"`
}

// Execute compiles the template directory
func (c *Compile) Execute(args []string) error {
	App.initializeLogging()
	path := string(c.Args.Path)
	log.Info("Path ", path)
	imageMaps := api.ImageMap{}
	for _, arg := range c.MapArgs {
		if err := imageMaps.Set(arg); err != nil {
			return err
		}
	}
	for src, target := range imageMaps {
		log.Info("Replace ", src, " with ", target)
	}
	if err := cmdCompile(path, imageMaps, c.ExcludeArgs); err != nil {
		return err
	}
	return nil
}

type metaTemplate struct {
	template.ServiceTemplate
	ServicedVersion version.ServicedVersion
	TemplateVersion map[string]string
}

// serviced-service compile DIR [[--map IMAGE,IMAGE] ...]
func cmdCompile(path string, maps api.ImageMap, exclusions []string) error {
	cfg := api.CompileTemplateConfig{
		Dir: path,
		Map: maps,
	}

	driver := api.New()

	if template, err := driver.CompileServiceTemplate(cfg); err != nil {
		fmt.Fprintln(os.Stderr, err)
	} else if template == nil {
		fmt.Fprintln(os.Stderr, "received nil template")
	} else {
		cmd := fmt.Sprintf("cd %s && git rev-parse HEAD", path)
		commit, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			commit = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git config --get remote.origin.url", path)
		repo, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			repo = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git rev-parse --abbrev-ref HEAD", path)
		branch, err := exec.Command("sh", "-c", cmd).Output()
		if err != nil {
			branch = []byte("unknown")
		}
		cmd = fmt.Sprintf("cd %s && git describe --always", path)
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
		mTemplate := metaTemplate{*template, version.GetVersion(), templateVersion}
		jsonTemplate, err := json.MarshalIndent(mTemplate, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal template: %s", err)
		}
		fmt.Println(string(jsonTemplate))
	}
	return nil
}
