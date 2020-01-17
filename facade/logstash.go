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

    "github.com/control-center/serviced/config"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/docker/pkg/version"
)


var (
	logstashConfigLock = &sync.Mutex{}
	ErrLogstashUnchanged = errors.New("logstash config unchanged")
)

type serviceLogInfo struct {
	ID         string               // service ID
	Name       string		// service name
	Version    string		// version of parent tenant application
	LogConfigs []servicedefinition.LogConfig
}

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

	logFilters, err := f.GetLogFilters(ctx)
	if err != nil {
		plog.WithError(err).Error("Could not retrieve service logFilters")
		return err
	}

	tenantIDs, err := f.GetTenantIDs(ctx)
	if err != nil {
		plog.WithError(err).Error("Could not retrieve tenant IDs")
		return err
	}

	// serviceLogs is a unique list of services with log files, such that if there are two or more
	// copies of the same service, then only the most recent is kept in the list. In other words,
	// in cases where two or more versions of a particular service are deployed, we only use
	// the most recent version to decide which log filters to install
	serviceLogs := map[string]serviceLogInfo{}
	for _, tenantID := range tenantIDs {
		svcs, err := f.GetServices(ctx, dao.ServiceRequest{TenantID: tenantID})
		if err != nil {
			plog.WithError(err).
				WithField("tenantid", tenantID).Error("Could not retrieve services for tenant")
			return err
		}

		tenantVersion := ""
		for _, svc := range svcs {
			if svc.ID == tenantID {
				tenantVersion = svc.Version
				break
			}
		}

		addServiceLogs(tenantVersion, svcs, serviceLogs)
	}

	filterSection := ""
	logFiles := []string{} 	// a list of unique application log file names
	auditLogSection := ""
	auditableTypes := []string{} // a list of unique log types where IsAudit=true

	plog.Debugf("Checking %d services", len(serviceLogs))
	for _, logInfo := range serviceLogs {
		filterSection += getFilterSection(logInfo, logFilters, &logFiles)
		auditLogSection += getAuditLogSection(logInfo.LogConfigs, &auditableTypes)
	}
	plog.Debugf("after checking services, auditLogSection=%s", auditLogSection)

	err = writeLogstashConfiguration(filterSection, auditLogSection)
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

func addServiceLogs(tenantVersion string, svcs []service.Service, serviceLogs map[string]serviceLogInfo){
	for _, svc := range svcs {
		if len(svc.LogConfigs) == 0 {
			continue
		}

		logInfo, ok := serviceLogs[svc.Name]
		if ok && logInfo.GreaterThanOrEqual(tenantVersion) {
			continue
		}
		serviceLogs[svc.Name] = serviceLogInfo{
			ID:         svc.ID,
			Name:       svc.Name,
			Version:    tenantVersion,
			LogConfigs: svc.LogConfigs,
		}
	}
}

// writeLogstashConfiguration takes an array of LogFilter and writes them to the
// appropriate place in the logstash.conf.
// This is required before logstash startup
//
// This method returns nil of logstash configuration was replaced,
// ErrLogstashUnchanged if the configuration is unchanged, or other errors if there was an I/O problem
func writeLogstashConfiguration(filterSection, auditLogSection string) error {

	logstashDir := getLogstashConfigDirectory()
	newConfigFile := filepath.Join(logstashDir, "logstash.conf.new")
	originalFile :=filepath.Join(logstashDir, "logstash.conf")
	logger := plog.WithFields(log.Fields{
		"newconfigfile": newConfigFile,
		"currentconfigfile": originalFile,
	})

	err := writeLogStashConfigFile(filterSection, auditLogSection, newConfigFile)
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


func getFilterSection(logInfo serviceLogInfo, logFilters []*logfilter.LogFilter, logFiles *[]string) string {
	filterSection := ""
	for _, config := range logInfo.LogConfigs {
		for _, filterName := range config.Filters {
			filterValue, ok := findNewestFilter(filterName, logInfo, logFilters)
			if !ok {
				plog.WithFields(log.Fields{
					"serviceid": logInfo.ID,
					"servicename": logInfo.Name,
					"filter": filterName,
				}).Warn("service log filter not found")
				continue
			}
			//  CC-3669: do not write duplicate filters for the same log file
			if !utils.StringInSlice(config.Path, *logFiles) {
				// Ruby Regex used in logstash conf uses / as a special character so we escape it.
				path := strings.Replace(config.Path, "/", "\\/", -1)
				filterSection += fmt.Sprintf("\n%s\n  if [file] =~ \"%s\" {\n%s\n  }\n",
					"  # Regex pattern used must overlap golang glob format to be valid in filebeat",
					path,
					indent(filterValue, "    "))
				*logFiles = append(*logFiles, config.Path)
			}
		}
	}
	return filterSection
}

// Finds the newest match for the named filter by version.
// If an exact match is found, use it. Otherwise, return the newest version of the named filter
func findNewestFilter(filterName string, logInfo serviceLogInfo, logFilters []*logfilter.LogFilter) (string, bool) {
	matchingFilters := []*logfilter.LogFilter{}
	for _, filter := range logFilters {
		if filterName == filter.Name {
			matchingFilters = append(matchingFilters, filter)
		}
	}

	var closest *logfilter.LogFilter
	var filterVersion version.Version
	svcVersion := version.Version(logInfo.Version)
	for _, filter := range matchingFilters {
		filterVersion = version.Version(filter.Version)
		if svcVersion.Equal(filterVersion) {
			closest = filter
			break
		} else if closest == nil || version.Version(closest.Version).LessThan(filterVersion) {
			closest = filter
		}
	}
	if closest == nil {
		return "", false
	}
	if !svcVersion.Equal(filterVersion)  {
		plog.WithFields(log.Fields{
			"serviceid": logInfo.ID,
			"servicename": logInfo.Name,
			"serviceversion": logInfo.Version,
			"filter": filterName,
			"filterversion": closest.Version,
		}).Warn("Unable to find exact match for service log filter version")
	}
	return closest.Filter, true
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
func writeLogStashConfigFile(filterSection string, auditLogSection string, outputPath string) error {
	// read the log configuration template
	templatePath := filepath.Join(getLogstashConfigDirectory(), "logstash.conf.template")

	contents, err := ioutil.ReadFile(templatePath)
	if err != nil {
		return err
	}

	// the newlines in the string below are deliberate so that the filter
	// syntax is correct
	filterDefinition := `
filter {
  mutate {
    rename => {
      "source" => "file"
    }

    convert => {
      "[fields][instance]" => "string"
      "[fields][ccWorkerID]" => "string"
      "[fields][poolid]" => "string"
    }

    # Save the time each message was received by logstash as rcvd_datetime
    add_field => [ "rcvd_datetime", "%{@timestamp}" ]
  }
# NOTE the filters are generated from the service definitions
` + string(filterSection) + `
}
`

    stdoutSection := `
        stdout { codec => "json_lines" }
`

	newContents := strings.Replace(string(contents), "${FILTER_SECTION}", filterDefinition, 1)
	if len(auditLogSection) > 0 {
		newContents = strings.Replace(string(newContents),"${AUDITLOG_SECTION}", auditLogSection, 1)
	}
	if config.GetOptions().LogstashStdout {
	    newContents = strings.Replace(string(newContents),"${STDOUT_SECTION}", stdoutSection, 1)
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

// Returns true if serviceLogInfo.Version is >= version.
func (sli *serviceLogInfo) GreaterThanOrEqual(value string) bool {
	if sli.Version == "" {
		return false
	} else if value == "" {
		return true
	}

	a := version.Version(sli.Version)
	b := version.Version(value)
	return a.GreaterThanOrEqualTo(b)
}
