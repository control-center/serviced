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

package facade

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"strings"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	log "github.com/Sirupsen/logrus"
)


var (
	logstashConfigLock = &sync.Mutex{}
	ErrLogstashUnchanged = errors.New("logstash config unchanged")
)

//
// ReloadLogstashConfig will create a new logstash configuration based on the union of information from all
// templates and all currently deployed services.  A union of values is used because scenarios like a service migration
// can expand the scope of auditable logs or change log filters without touching the currently loaded templates.
//
// If the new configuration is different from the one currently used by logstash,
// then it will rewrite the logstash.conf file, trusting that logstash will recognize the file change
// and reload the new filter set.
//
// This method should be called anytime the available service templates are modified or deployed services are upgraded.
//
// This method depends on the elasticsearch container being up and running.
//
// Note that the strategy of using a union of fields from the templates and deployed services means that if a template
// says a certain log file should be auditable, but the deployed service does not, then the field will still be
// auditable.  Refactoring to update logstash soley on the basis of deployed services might resolve that problem, but
// it still leaves the constraint that in cases where separate tenant applications have conflicting filters/auditable
// types for the same file, the last one wins.
//
func (f *Facade) ReloadLogstashConfig(ctx datastore.Context) error {
	// serialize updates so that we don't have different threads overwriting the same file.
	logstashConfigLock.Lock()
	defer logstashConfigLock.Unlock()

	templates, err := f.GetServiceTemplates(ctx)
	if err != nil {
		plog.WithError(err).Error("Could not retrieve service templates")
		return err
	}

	tenantIDs, err := f.GetTenantIDs(ctx)
	if err != nil {
		plog.WithError(err).Error("Could not retrieve tenant IDs")
		return err
	}

	services := []service.Service{}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
		if err != nil {
			plog.WithError(err).
				WithField("tenantid", tenantID).Error("Could not retrieve services for tenant")
			return err
		}

		services = append(services, svcs...)
	}

	err = writeLogstashConfiguration(templates, services)
	if err == ErrLogstashUnchanged {
		return nil
	} else if err != nil {
		plog.WithError(err).Error("Could not write logstash configuration: %s", err)
		return err
	}
	return nil
}

type reloadLogstashContainer func(ctx datastore.Context, f FacadeInterface) error

var LogstashContainerReloader reloadLogstashContainer = reloadLogstashContainerImpl

func reloadLogstashContainerImpl(ctx datastore.Context, f FacadeInterface) error {
	return f.ReloadLogstashConfig(ctx)
}


// writeLogstashConfiguration takes a map of ServiceTemplates and writes them to the
// appropriate place in the logstash.conf.
// This is required before logstash startup
//
// This method returns nil of logstash configuration was replaced,
// ErrLogstashUnchanged if the configuration is unchanged, or other errors if there was an I/O problem
func writeLogstashConfiguration(templates map[string]servicetemplate.ServiceTemplate, services []service.Service) error {
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

	typeFilter := []string{}
	auditableTypes := []string{}
	auditLogSection := ""
	plog.Debugf("Checking %d templates", len(templates))
	for _, template := range templates {
		filters += getFiltersFromTemplates(template.Services, filterDefs, &typeFilter)
		auditLogSection = getAuditLogSectionFromTemplates(template.Services, &auditableTypes)
	}
	plog.Debugf("after templates, auditLogSection=%s", auditLogSection)

	plog.Debugf("Checking %d services", len(services))
	for _, svc := range services {
		filters += getFilters(svc.LogConfigs, filterDefs, &typeFilter)
		auditLogSection += getAuditLogSection(svc.LogConfigs,  &auditableTypes)
	}
	plog.Debugf("after services, auditLogSection=%s", auditLogSection)

	logstashDir := getLogstashConfigDirectory()
	newConfigFile := filepath.Join(logstashDir, "logstash.conf.new")
	originalFile :=filepath.Join(logstashDir, "logstash.conf")
	logger := plog.WithFields(log.Fields{
		"newconfigfile": newConfigFile,
		"currentconfigfile": originalFile,
	})

	err := writeLogStashConfigFile(filters, auditLogSection, newConfigFile)
	if err != nil {
		logger.WithError(err).Error("Unable to create new logstash config file")
		return err
	}

	originalContents, err := ioutil.ReadFile(originalFile)
	if err != nil {
		logger.WithError(err).Error("Unable to read current logstash config file")
		return err
	}

	newContents, err := ioutil.ReadFile(newConfigFile)
	if err != nil {
		logger.WithError(err).Error("Unable to read new logstash config file")
		return err
	}

	// Now compare the new config to the current config, and
	// only replace the current config if they are different
	if bytes.Equal(originalContents, newContents) {
		return ErrLogstashUnchanged
	} else if err := os.Rename(newConfigFile, originalFile); err != nil {
		logger.WithError(err).Error("Unable to replace current logstash config file")
		return err
	}
	plog.WithField("currentconfigfile", originalFile).Info("Updated logstash configuration")
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

func getFiltersFromTemplates(services []servicedefinition.ServiceDefinition, filterDefs map[string]string, typeFilter *[]string) string {
	filters := ""
	for _, svc := range services {
		filters += getFilters(svc.LogConfigs, filterDefs, typeFilter)
		if len(svc.Services) > 0 {
			subFilts := getFiltersFromTemplates(svc.Services, filterDefs, typeFilter)
			filters += subFilts
		}
	}
	return filters
}

func getFilters(configs []servicedefinition.LogConfig, filterDefs map[string]string, typeFilter *[]string) string {
	filters := ""
	for _, config := range configs {
		for _, filtName := range config.Filters {
			//do not write duplicate types, logstash doesn't handle this
			if !utils.StringInSlice(config.Type, *typeFilter) {
				filters += fmt.Sprintf("\n  if [file] == \"%s\" {\n    %s\n  }\n",
					config.Path, indent(filterDefs[filtName], "    "))
				*typeFilter = append(*typeFilter, config.Type)
			}
		}
	}
	return filters
}

func getAuditLogSectionFromTemplates(services []servicedefinition.ServiceDefinition, auditTypes *[]string) string {
	auditSection := ""
	for _, svc := range services {
		auditSection += getAuditLogSection(svc.LogConfigs, auditTypes)
		if len(svc.Services) > 0 {
			subServiceOutput := getAuditLogSectionFromTemplates(svc.Services, auditTypes)
			auditSection += subServiceOutput
		}
	}
	return auditSection
}

func getAuditLogSection(configs []servicedefinition.LogConfig, auditTypes *[]string) string {
	auditSection := ""
	fileSection := `file {
  path => "%s"
  codec => line { format => "%%{message}"}
}`
	auditLogFile := filepath.Join(utils.LOGSTASH_LOCAL_SERVICED_LOG_DIR, "application-audit.log")
	fileSection = fmt.Sprintf(fileSection, auditLogFile)
	for _, config := range configs {
		if config.IsAudit {
			if !utils.StringInSlice(config.Type, *auditTypes){
				plog.Infof("found type %q enabled for audit, file=%s", config.Type, config.Path)
				auditSection += fmt.Sprintf("\n        if [fields][type] == \"%s\" {\n%s        }",
					config.Type, indent(fileSection, "            "))
				*auditTypes = append(*auditTypes, config.Type)
			}
		}
	}
	return auditSection
}

// This method writes out the config file for logstash. It uses
// the logstash.conf.template and does a variable replacement.
func writeLogStashConfigFile(filters string, auditLogSection string, outputPath string) error {
	// read the log configuration template
	templatePath := filepath.Join(getLogstashConfigDirectory(), "logstash.conf.template")

	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// the newlines in the string below are deliberate so that the filter
	// syntax is correct
	filterSection := `
filter {
  mutate {
    rename => {
      "source" => "file"
    }
  }
# NOTE the filters are generated from the service definitions
` + string(filters) + `
}
`
	newContents := strings.Replace(string(contents), "${FILTER_SECTION}", filterSection, 1)
	if len(auditLogSection) > 0 {
		newContents = strings.Replace(string(newContents),"${AUDITLOG_SECTION}", auditLogSection, 1)
	}
	newBytes := []byte(newContents)
	// generate the filters section
	// write the log file
	return ioutil.WriteFile(outputPath, newBytes, 0644)
}

func indent(src, tab string) string {
	result := ""
	lines := strings.Split(src, "\n")
	for _, line := range lines {
		result += tab + line + "\n"
	}
	return result
}

func getLogstashConfigDirectory() string {
	return filepath.Join(utils.ResourcesDir(), "logstash")
}
