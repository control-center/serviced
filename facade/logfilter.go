// Copyright 2017 The Serviced Authors.
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
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
)

// UpdateLogFilters updates the log filters from a service template.
func (f *Facade) UpdateLogFilters(ctx datastore.Context, serviceTemplate *servicetemplate.ServiceTemplate) error {
	logger := plog.WithFields(logrus.Fields{
		"templateid":      serviceTemplate.ID,
		"templatename":    serviceTemplate.Name,
		"templateversion": serviceTemplate.Version,
	})
	action := "find"
	filterDefs := getFilterDefinitions(serviceTemplate.Services)
	for name, value := range filterDefs {
		logFilter, err := f.logfilterStore.Get(ctx, name, serviceTemplate.Version)
		if err == nil {
			logFilter.Filter = value
			err = f.logfilterStore.Put(ctx, logFilter)
			action = "update"
		} else if err != nil && datastore.IsErrNoSuchEntity(err) {
			newFilter := &logfilter.LogFilter{
				Name:    name,
				Version: serviceTemplate.Version,
				Filter:  value,
			}
			err = f.logfilterStore.Put(ctx, newFilter)
			action = "add"
		}
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"action":     action,
				"filtername": name,
			}).Error("Failed to add/update log filter")
			return err
		}
		logger.WithFields(logrus.Fields{
			"action":        action,
			"filtername":    name,
			"filterversion": serviceTemplate.Version,
		}).Debug("Saved LogFilter")
	}

	return nil
}

// RemoveLogFilters removes the log filters specified in a service template.
func (f *Facade) RemoveLogFilters(ctx datastore.Context, serviceTemplate *servicetemplate.ServiceTemplate) error {
	logger := plog.WithFields(logrus.Fields{
		"templateid":      serviceTemplate.ID,
		"templatename":    serviceTemplate.Name,
		"templateversion": serviceTemplate.Version,
	})
	filterDefs := getFilterDefinitions(serviceTemplate.Services)
	for name := range filterDefs {
		err := f.logfilterStore.Delete(ctx, name, serviceTemplate.Version)
		// ignore not-found errors, but stop on anything other failure
		if err != nil && !datastore.IsErrNoSuchEntity(err) {
			logger.WithError(err).WithFields(logrus.Fields{
				"filtername": name,
			}).Error("Failed to remove log filter")
			return err
		}
	}
	return nil
}

// BootstrapLogFilters bootstraps the LogFilter store in cases where templates were added to the
// system in some prior CC version which did not have a separate store for LogFilters.
// For cases like that, this code creates new records in the LogFilter store for each logfilter
// found in an existing service template.
func (f *Facade) BootstrapLogFilters(ctx datastore.Context) (bool, error) {
	logFiltersCreated := false
	templates, err := f.GetServiceTemplates(ctx)
	if err != nil {
		plog.WithError(err).Error("Could not retrieve service templates")
		return false, err
	}

	for _, template := range templates {
		logger := plog.WithFields(logrus.Fields{
			"templateid":      template.ID,
			"templatename":    template.Name,
			"templateversion": template.Version,
		})
		filterDefs := getFilterDefinitions(template.Services)
		for name, value := range filterDefs {
			if _, err := f.logfilterStore.Get(ctx, name, template.Version); err == nil {
				continue
			} else if !datastore.IsErrNoSuchEntity(err) {
				logger.WithError(err).
					WithField("filtername", name).
					Error("Could not retrieve log filter")
				return false, err
			} else {
				err = f.logfilterStore.Put(ctx, &logfilter.LogFilter{
					Name:    name,
					Version: template.Version,
					Filter:  value,
				})
				if err != nil {
					logger.WithError(err).WithFields(logrus.Fields{
						"filtername": name,
					}).Error("Failed to add log filter")
					return false, err
				}
				logFiltersCreated = true
				logger.WithField("filtername", name).Info("Added log filter")
			}
		}
	}

	return logFiltersCreated, nil
}

// GetLogFilters returns an array of LogFilter objects.
func (f *Facade) GetLogFilters(ctx datastore.Context) ([]*logfilter.LogFilter, error) {
	return f.logfilterStore.GetLogFilters(ctx)
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
