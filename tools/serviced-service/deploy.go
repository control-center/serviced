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
	"strings"

	"github.com/jessevdk/go-flags"

	"github.com/control-center/serviced/domain/service"
	definition "github.com/control-center/serviced/domain/servicedefinition"
	template "github.com/control-center/serviced/domain/servicetemplate"
)

// Deploy is the subcommand for converting a service template into service definitions.
type Deploy struct {
	TenantID string `long:"tenant-id" default:"" description:"Tenant ID"`
	Args     struct {
		File flags.Filename `positional-arg-name:"TEMPLATE_FILE" description:"Template file"`
	} `positional-args:"yes" required:"yes"`
}

// Execute converts a template file into service definitions.
func (c *Deploy) Execute(args []string) error {
	App.initializeLogging()
	filepath := string(c.Args.File)

	var err error

	var tmpl *template.ServiceTemplate
	tmpl, err = LoadTemplate(filepath)
	if err != nil {
		return err
	}

	var services = NewServiceList()
	err = deploy(services, c.TenantID, "default", "test", "", false, tmpl.Services[0])
	if err != nil {
		return err
	}

	marshaled, err := json.MarshalIndent(services, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal template: %s", err)
	}
	fmt.Println(string(marshaled))

	return nil
}

type deployInfo struct {
	parentID string
	template definition.ServiceDefinition
}

func deploy(
	services *ServiceList,
	tenantID, poolID, deploymentID, parentID string,
	overwrite bool,
	sd definition.ServiceDefinition,
) error {
	var err error
	var queue = []deployInfo{}

	queue = append(queue, deployInfo{parentID: parentID, template: sd})

	var info deployInfo
	var newsvc, oldsvc *service.Service
	var child definition.ServiceDefinition

	for len(queue) > 0 {
		info, queue = queue[0], queue[1:]
		newsvc, err = service.BuildService(
			info.template, info.parentID, poolID, int(service.SVCStop), deploymentID,
		)
		if err != nil {
			return fmt.Errorf(
				"Could not build service from template %s: %s", info.template.Name, err,
			)
		}
		if tenantID == "" {
			tenantID = newsvc.ID
		}

		// Create a fake deployed image ID
		if info.template.ImageID != "" {
			newsvc.ImageID = makeImageID(tenantID, info.template)
		}

		oldsvc, err = services.FindChild(newsvc.ParentServiceID, newsvc.Name)
		if err != nil {
			return err
		}
		if oldsvc != nil {
			if overwrite {
				newsvc.ID = oldsvc.ID
				newsvc.CreatedAt = oldsvc.CreatedAt

			} else {
				path, err := services.GetServicePath(oldsvc.ID)
				if err != nil {
					return err
				}
				return fmt.Errorf("Service already exists: %s", path)
			}
		} else {
			err = services.Append(*newsvc) // Copy the service into the ServiceList
			if err != nil {
				return err
			}
			err = newsvc.EvaluateEndpointTemplates(
				services.GetService,
				services.FindChildService,
				0,
			)
			if err != nil {
				return fmt.Errorf(
					"Unable to evaluate endpoint templates for service %s: %s",
					info.template.Name, err,
				)
			}
		}

		// Add the service template's child services to the queue for processing.
		for _, child = range info.template.Services {
			queue = append(queue, deployInfo{parentID: newsvc.ID, template: child})
		}
	}
	return nil
}

func makeImageID(tenantID string, tmpl definition.ServiceDefinition) string {
	imageName := strings.Split(tmpl.ImageID, ":")[0]
	parts := strings.Split(imageName, "/")
	imageBaseName := parts[len(parts)-1]
	return fmt.Sprintf("localhost:5000/%s/%s:latest", tenantID, imageBaseName)
}
