// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package dao

import (
	"fmt"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
)

func resourcesDir() string {
	homeDir := os.Getenv("SERVICED_HOME")
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		return path.Clean(path.Join(path.Dir(filename), "..", "isvcs", "resources"))

	}
	return path.Clean(path.Join(homeDir, "isvcs/resources"))
}

// WriteConfigurationFile takes a map of ServiceTemplates and writes them to the
// appropriate place in the logstash.conf.
func WriteConfigurationFile(templates map[string]*servicetemplate.ServiceTemplate) error {
	// the definitions are a map of filter name to content
	// they are found by recursively going through all the service definitions
	filterDefs := make(map[string]string)
	for _, template := range templates {
		subFilterDefs := getFilterDefinitions(template.Services)
		for name, value := range subFilterDefs {
			filterDefs[name] = value
		}
	}

	// filters will be a syntactically correct logstash filters section
	filters := ""

	for _, template := range templates {
		filters += getFilters(template.Services, filterDefs)
	}
	configFile := resourcesDir() + "/logstash/logstash.conf"
	err := writeLogStashConfigFile(filters, configFile)
	if err != nil {
		return err
	}
	return nil
}

func getFilterDefinitions(services []servicedefinition.ServiceDefinition) map[string]string {
	filterDefs := make(map[string]string)
	for _, service := range services {
		for name, value := range service.LogFilters {
			filterDefs[name] = value
		}

		if len(service.Services) > 0 {
			subFilterDefs := getFilterDefinitions(service.Services)
			for name, value := range subFilterDefs {
				filterDefs[name] = value
			}
		}
	}
	return filterDefs
}

func getFilters(services []servicedefinition.ServiceDefinition, filterDefs map[string]string) string {
	filters := ""
	for _, service := range services {
		for _, config := range service.LogConfigs {
			for _, filtName := range config.Filters {
				filters += fmt.Sprintf("\nif [type] == \"%s\" \n {\n  %s \n}", config.Type, filterDefs[filtName])
			}
		}
		if len(service.Services) > 0 {
			subFilts := getFilters(service.Services, filterDefs)
			filters += subFilts
		}
	}
	return filters
}

// This method writes out the config file for logstash. It uses
// the logstash.conf.template and does a variable replacement.
func writeLogStashConfigFile(filters string, outputPath string) error {
	// read the log configuration template
	templatePath := resourcesDir() + "/logstash/logstash.conf.template"

	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// the newlines in the string below are deliberate so that the filter
	// syntax is correct
	filterSection := `
filter {
# NOTE the filters are generated from the service definitions
` + string(filters) + `
}
`
	newContents := strings.Replace(string(contents), "${FILTER_SECTION}", filterSection, 1)
	newBytes := []byte(newContents)
	// generate the filters section
	// write the log file
	return ioutil.WriteFile(outputPath, newBytes, 0644)
}
