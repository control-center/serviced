// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package facade

import (
	"errors"
	"reflect"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// GetServiceConfigs returns the config files for a service
func (f *Facade) GetServiceConfigs(ctx datastore.Context, serviceID string) ([]service.Config, error) {
	logger := plog.WithField("serviceid", serviceID)

	tenantID, servicePath, err := f.getServicePath(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not trace service path")
		return nil, err
	}

	logger = logger.WithFields(log.Fields{
		"tenantid":    tenantID,
		"servicepath": servicePath,
	})

	files, err := f.configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		logger.WithError(err).Debug("Could not load existing configs for service")
		return nil, err
	}

	confs := make([]service.Config, len(files))
	for i, file := range files {
		confs[i] = service.Config{
			ID:       file.ID,
			Filename: file.ConfFile.Filename,
		}
	}

	logger.WithField("count", len(files)).Debug("Loaded config files for service")
	return confs, nil
}

// GetServiceConfig returns a config file
func (f *Facade) GetServiceConfig(ctx datastore.Context, fileID string) (*servicedefinition.ConfigFile, error) {
	logger := plog.WithField("fileid", fileID)

	file := &serviceconfigfile.SvcConfigFile{}
	if err := f.configStore.Get(ctx, serviceconfigfile.Key(fileID), file); err != nil {
		logger.WithError(err).Debug("Could not get service config file")
		return nil, err
	}

	return &file.ConfFile, nil
}

// AddServiceConfig creates a config file for a service
func (f *Facade) AddServiceConfig(ctx datastore.Context, serviceID string, conf servicedefinition.ConfigFile) error {
	logger := plog.WithFields(log.Fields{
		"serviceid": serviceID,
		"filename":  conf.Filename,
	})

	tenantID, servicePath, err := f.getServicePath(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not trace service path")
		return err
	}

	logger = logger.WithFields(log.Fields{
		"tenantid":    tenantID,
		"servicepath": servicePath,
	})

	// make sure the file does not already exist
	file, err := f.configStore.GetConfigFile(ctx, tenantID, servicePath, conf.Filename)
	if err != nil {
		logger.WithError(err).Debug("Could not search for service config file")
		return err
	}

	if file != nil {
		logger.WithField("fileid", file.ID).Debug("File already exists for service")
		return errors.New("config file exists")
	}

	// initialize the database record for the file
	file, err = serviceconfigfile.New(tenantID, servicePath, conf)
	if err != nil {
		logger.WithError(err).Debug("Could not initialize service config file record for the database")
		return err
	}

	// write the record into the database
	if err := f.configStore.Put(ctx, serviceconfigfile.Key(file.ID), file); err != nil {
		logger.WithField("fileid", file.ID).WithError(err).Debug("Could not add record to the database")
		return err
	}

	logger.Debug("Created new service config file")
	return nil
}

// UpdateServiceConfig updates an existing service config file
func (f *Facade) UpdateServiceConfig(ctx datastore.Context, fileID string, conf servicedefinition.ConfigFile) error {
	logger := plog.WithFields(log.Fields{
		"fileid":   fileID,
		"filename": conf.Filename,
	})

	file := &serviceconfigfile.SvcConfigFile{}
	if err := f.configStore.Get(ctx, serviceconfigfile.Key(fileID), file); err != nil {
		logger.WithError(err).Debug("Could not get service config file")
		return err
	}

	// update the database record for the file
	file.ConfFile = conf

	// write the record into the database
	if err := f.configStore.Put(ctx, serviceconfigfile.Key(fileID), file); err != nil {
		logger.WithError(err).Debug("Could not update record in database")
		return err
	}

	logger.Debug("Updated service config file")
	return nil
}

// DeleteServiceConfig deletes a service config file
func (f *Facade) DeleteServiceConfig(ctx datastore.Context, fileID string) error {
	logger := plog.WithField("fileid", fileID)

	if err := f.configStore.Delete(ctx, serviceconfigfile.Key(fileID)); err != nil {
		logger.WithError(err).Debug("Could not delete service config file")
		return err
	}

	logger.Debug("Deleted service config file")
	return nil
}

// getServicePath returns the tenantID and the full path of the service
// TODO: update function to include deploymentID in the service path
func (f *Facade) getServicePath(ctx datastore.Context, serviceID string) (tenantID string, servicePath string, err error) {
	gs := func(id string) (service.Service, error) {
		return f.getService(ctx, id)
	}
	return f.serviceCache.GetServicePath(serviceID, gs)
}

// updateServiceConfigs adds or updates configuration files.  If forceDelete is
// set to true, then remove any extranneous service configurations.
func (f *Facade) updateServiceConfigs(ctx datastore.Context, serviceID string, configFiles []servicedefinition.ConfigFile, forceDelete bool) error {
	tenantID, servicePath, err := f.getServicePath(ctx, serviceID)
	if err != nil {
		return err
	}
	svcConfigFiles, err := f.configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		glog.Errorf("Could not load existing configs for service %s: %s", serviceID, err)
		return err
	}
	svcConfigFileMap := make(map[string]*serviceconfigfile.SvcConfigFile)
	for _, svcConfigFile := range svcConfigFiles {
		svcConfigFileMap[svcConfigFile.ConfFile.Filename] = svcConfigFile
	}
	for _, configFile := range configFiles {
		svcConfigFile, ok := svcConfigFileMap[configFile.Filename]
		if ok {
			delete(svcConfigFileMap, configFile.Filename)
			// do not update database if there are no configuration changes
			if reflect.DeepEqual(svcConfigFile.ConfFile, configFile) {
				glog.V(1).Infof("Skipping config file %s", configFile.Filename)
				continue
			}
			svcConfigFile.ConfFile = configFile
			glog.Infof("Updating config file %s for service %s", configFile.Filename, serviceID)
		} else {
			svcConfigFile, err = serviceconfigfile.New(tenantID, servicePath, configFile)
			if err != nil {
				glog.Errorf("Could not create new service config file %s for service %s: %s", configFile.Filename, serviceID, err)
				return err
			}
			glog.Infof("Adding config file %s for service %s", configFile.Filename, serviceID)
		}
		if err := f.configStore.Put(ctx, serviceconfigfile.Key(svcConfigFile.ID), svcConfigFile); err != nil {
			glog.Errorf("Could not update service config file %s for service %s: %s", configFile.Filename, serviceID, err)
			return err
		}
	}
	// delete any nonmatching configurations
	if forceDelete {
		for filename, svcConfigFile := range svcConfigFileMap {
			if err := f.configStore.Delete(ctx, serviceconfigfile.Key(svcConfigFile.ID)); err != nil {
				glog.Errorf("Could not delete service config file %s for service %s: %s", filename, serviceID, err)
				return err
			}
			glog.Infof("Deleting config file %s from service %s", filename, serviceID)
		}
	}
	return nil
}

// fillServiceConfigs sets the configuration files on the service
func (f *Facade) fillServiceConfigs(ctx datastore.Context, svc *service.Service) error {
	tenantID, servicePath, err := f.getServicePath(ctx, svc.ID)
	if err != nil {
		return err
	}
	svcConfigFiles, err := f.configStore.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		glog.Errorf("Could not load existing configs for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}
	svc.ConfigFiles = make(map[string]servicedefinition.ConfigFile)
	for _, configFile := range svc.OriginalConfigs {
		svc.ConfigFiles[configFile.Filename] = configFile
		glog.V(1).Infof("Copying original config file %s from service %s (%s)", configFile.Filename, svc.Name, svc.ID)
	}
	for _, svcConfigFile := range svcConfigFiles {
		filename, configFile := svcConfigFile.ConfFile.Filename, svcConfigFile.ConfFile
		svc.ConfigFiles[filename] = configFile
		glog.V(1).Infof("Loading config file %s for service %s (%s)", filename, svc.Name, svc.ID)
	}
	return nil
}
