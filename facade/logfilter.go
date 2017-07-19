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

)
import (
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/Sirupsen/logrus"
)

func (f *Facade) UpdateLogFilters(ctx datastore.Context, serviceTemplate *servicetemplate.ServiceTemplate) error {
	logger := plog.WithFields(logrus.Fields{
		"templateid": serviceTemplate.ID,
		"templatename": serviceTemplate.Name,
		"templateversion": serviceTemplate.Version,
	})
	action := "find"
	filterDefs := getFilterDefinitions(serviceTemplate.Services)
	for name, value := range filterDefs {
		logFilter, err := f.logFilterStore.Get(ctx, name, serviceTemplate.Version)
		if err == nil {
			logFilter.Filter = value
			err = f.logFilterStore.Put(ctx, logFilter)
			action = "update"
		} else if err != nil && datastore.IsErrNoSuchEntity(err) {
			newFilter := &logfilter.LogFilter{
				Name:    name,
				Version: serviceTemplate.Version,
				Filter:  value,
			}
			err = f.logFilterStore.Put(ctx, newFilter)
			action = "add"
		}
		if err != nil {
			logger.WithError(err).WithFields(logrus.Fields{
				"action": action,
				"filtername": name,
			}).Error("Failed to add/update log filter")
			return err
		}
	}

	return nil
}

func (f *Facade) RemoveLogFilters(ctx datastore.Context, serviceTemplate *servicetemplate.ServiceTemplate) error {
	logger := plog.WithFields(logrus.Fields{
		"templateid": serviceTemplate.ID,
		"templatename": serviceTemplate.Name,
		"templateversion": serviceTemplate.Version,
	})
	filterDefs := getFilterDefinitions(serviceTemplate.Services)
	for name, _ := range filterDefs {
		err := f.logFilterStore.Delete(ctx, name, serviceTemplate.Version)
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
