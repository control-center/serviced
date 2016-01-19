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
	"path"
	"reflect"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// getServicePath returns the tenantID and the full path of the service
// TODO: update function to include deploymentID in the service path
func (f *Facade) getServicePath(ctx datastore.Context, serviceID string) (tenantID string, servicePath string, err error) {
	store := f.serviceStore
	svc, err := store.Get(ctx, serviceID)
	if err != nil {
		glog.Errorf("Could not look up service %s: %s", serviceID, err)
		return "", "", err
	}
	if svc.ParentServiceID == "" {
		return serviceID, "/" + serviceID, nil
	}
	tenantID, servicePath, err = f.getServicePath(ctx, svc.ParentServiceID)
	if err != nil {
		return "", "", err
	}
	return tenantID, path.Join(servicePath, serviceID), nil
}

// updateServiceConfigs adds or updates configuration files.  If forceDelete is
// set to true, then remove any extranneous service configurations.
func (f *Facade) updateServiceConfigs(ctx datastore.Context, serviceID string, configFiles []servicedefinition.ConfigFile, forceDelete bool) error {
	tenantID, servicePath, err := f.getServicePath(ctx, serviceID)
	if err != nil {
		return err
	}
	configStore := serviceconfigfile.NewStore()
	svcConfigFiles, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
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
		if err := configStore.Put(ctx, serviceconfigfile.Key(svcConfigFile.ID), svcConfigFile); err != nil {
			glog.Errorf("Could not update service config file %s for service %s: %s", configFile.Filename, serviceID, err)
			return err
		}
	}
	// delete any nonmatching configurations
	if forceDelete {
		for filename, svcConfigFile := range svcConfigFileMap {
			if err := configStore.Delete(ctx, serviceconfigfile.Key(svcConfigFile.ID)); err != nil {
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
	configStore := serviceconfigfile.NewStore()
	svcConfigFiles, err := configStore.GetConfigFiles(ctx, tenantID, servicePath)
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
