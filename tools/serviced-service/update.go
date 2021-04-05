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

	"github.com/jessevdk/go-flags"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/logfilter"
	definition "github.com/control-center/serviced/domain/servicedefinition"
	template "github.com/control-center/serviced/domain/servicetemplate"
)

// Update is the subcommand for updating definitions
type Update struct {
	Args struct {
		TemplateFile flags.Filename `positional-arg-name:"TEMPLATE_FILE" description:"Compiled service template file (to get log filters)"`
		ServicesFile flags.Filename `positional-arg-name:"SERVICES_FILE" description:"Deployed service definitions file"`
		UpdateFile   flags.Filename `positional-arg-name:"UPDATE_FILE" description:"Update file (produced by servicemigration)"`
	} `positional-args:"yes" required:"yes"`
}

// Execute compiles the template directory
func (c *Update) Execute(args []string) error {
	App.initializeLogging()
	templatefile := string(c.Args.TemplateFile)
	servicesfile := string(c.Args.ServicesFile)
	updatefile := string(c.Args.UpdateFile)

	template, err := LoadTemplate(templatefile)
	if err != nil {
		return err
	}
	filters := extractLogFilters(template)

	services, err := loadDefinitions(servicesfile)
	if err != nil {
		return err
	}

	updates, err := loadUpdates(updatefile)
	if err != nil {
		return err
	}

	migrator := NewMigrationContext(services, filters)

	var migratedServices ServiceList
	migratedServices, err = migrator.Migrate(updates)
	if err != nil {
		return err
	}

	marshaled, err := json.MarshalIndent(migratedServices, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal updated services: %s", err)
	}
	fmt.Println(string(marshaled))

	return nil
}

func extractLogFilters(tmpl *template.ServiceTemplate) map[string]logfilter.LogFilter {
	var queue = []definition.ServiceDefinition{}
	for _, svc := range tmpl.Services {
		queue = append(queue, svc)
	}
	var svc, child definition.ServiceDefinition

	var filters = make(map[string]logfilter.LogFilter)
	for len(queue) > 0 {
		svc, queue = queue[0], queue[1:]
		for name, value := range svc.LogFilters {
			filters[name] = logfilter.LogFilter{
				Name:    name,
				Filter:  value,
				Version: tmpl.Version,
			}
		}
		for _, child = range svc.Services {
			queue = append(queue, child)
		}
	}
	return filters
}

func loadDefinitions(filepath string) (*ServiceList, error) {
	var input *os.File
	var err error

	if input, err = os.Open(filepath); err != nil {
		return nil, err
	}
	defer input.Close()

	var services = NewServiceList()
	if err = json.NewDecoder(input).Decode(services); err != nil {
		return nil, fmt.Errorf("Could not read service definitions: %s", err)
	}
	return services, nil
}

func loadUpdates(filepath string) (dao.ServiceMigrationRequest, error) {
	var input *os.File
	var err error
	var updates dao.ServiceMigrationRequest

	if input, err = os.Open(filepath); err != nil {
		return updates, err
	}
	defer input.Close()

	if err = json.NewDecoder(input).Decode(&updates); err != nil {
		return updates, fmt.Errorf("Could not read service updates: %s", err)
	}

	return updates, nil
}
