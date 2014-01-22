/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"fmt"
	"github.com/zenoss/serviced/dao"
	"io/ioutil"
	"strings"
)

func WriteConfigurationFile(templates map[string]*dao.ServiceTemplate) error {
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

func getFilterDefinitions(services []dao.ServiceDefinition) map[string]string {
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

func getFilters(services []dao.ServiceDefinition, filterDefs map[string]string) string {
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
	configPath := outputPath

	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}
	newContents := strings.Replace(string(contents), "${FILTER_SECTION}", filters, 1)
	newBytes := []byte(newContents)
	// generate the filters section
	// write the log file
	err = ioutil.WriteFile(configPath, newBytes, 0644)
	if err != nil {
		return err
	}
	return nil
}
