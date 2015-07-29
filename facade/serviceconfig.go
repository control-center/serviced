// Copyright 2015 The Serviced Authors.
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
	"path"
	"reflect"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// GetServicePath returns the tenantID and path to a service, starting from the
// deployment id.
func (f *Facade) GetServicePath(ctx datastore.Context, serviceID string) (string, string, error) {
	glog.V(2).Infof("Facade.GetServicePath: %s", serviceID)
	store := f.serviceStore

	var getParentPath func(string) (string, string, error)
	getParentPath = func(string) (string, string, error) {
		svc, err := store.Get(ctx, serviceID)
		if err != nil {
			glog.Errorf("Could not look up service %s: %s", serviceID, err)
			return "", "", err
		}
		if svc.ParentServiceID == "" {
			return svc.ID, path.Join(svc.DeploymentID, svc.ID), nil
		}

		t, p, err := getParentPath(svc.ParentServiceID)
		if err != nil {
			return "", "", err
		}
		return t, path.Join(p, svc.ID), nil
	}
	return getParentPath(serviceID)
}

// setServiceConfigs fills out the configuration data for the service.
func (f *Facade) setServiceConfigs(ctx datastore.Context, svc *service.Service) error {
	tenantID, servicePath, err := f.GetServicePath(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not look up path to service %s (%s): %s", svc.ID, svc.Name, err)
		return err
	}

	// trim the deployment id and get the current configs
	servicePath = strings.TrimPrefix(servicePath, svc.DeploymentID)
	confs, err := f.getServiceConfigs(ctx, tenantID, servicePath)
	if err != nil {
		glog.Errorf("Could not get configs for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	// fill out the configs
	ogconfs := make(map[string]servicedefinition.ConfigFile)
	if len(confs) == 0 {
		// if there are no confs stored in the db, use the original conf data
		for name, conf := range svc.OriginalConfigs {
			ogconfs[name] = conf
		}
	} else {
		for name, conf := range confs {
			ogconfs[name] = conf.ConfFile
		}
	}
	svc.ConfigFiles = ogconfs
	return nil
}

// getServiceConfigs returns a map of config files for a service.
func (f *Facade) getServiceConfigs(ctx datastore.Context, tenantID, servicePath string) (map[string]*serviceconfigfile.SvcConfigFile, error) {
	store := serviceconfigfile.NewStore()
	confs, err := store.GetConfigFiles(ctx, tenantID, servicePath)
	if err != nil {
		return nil, err
	}

	confmap := make(map[string]*serviceconfigfile.SvcConfigFile)
	for _, conf := range confs {
		confmap[conf.ConfFile.Filename] = conf
	}
	return confmap, nil
}

// updateServiceConfigs updates the config file records for a service.  If
// migrate is set to true then it is expected that the old original configs
// will be passed in, and the new original configs will be on the service.  If
// migrate is set to false, the confs will be the new configs as defined by the
// user.
func (f *Facade) updateServiceConfigs(ctx datastore.Context, svc service.Service, confs map[string]servicedefinition.ConfigFile, migrate bool) error {
	store := serviceconfigfile.NewStore()
	// check to see if we are migrating the service configs
	if migrate {
		if reflect.DeepEqual(svc.OriginalConfigs, confs) {
			return nil
		}
		// swap the config files so that the old original configs are on the
		// service.
		svc.OriginalConfigs, confs = confs, svc.OriginalConfigs
	}

	tenantID, servicePath, err := f.GetServicePath(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not look up path to service %s (%s): %s", svc.ID, svc.Name, err)
		return err
	}
	servicePath = strings.TrimPrefix(servicePath, svc.DeploymentID)
	curconfs, err := f.getServiceConfigs(ctx, tenantID, servicePath)
	if err != nil {
		glog.Errorf("Could not look up configs for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	}

	// add/update conf files
	for name, newconf := range confs {
		// does this file already exist?
		if curconf, ok := curconfs[name]; ok {
			if migrate {
				// migrate the file if it hasn't changed since the default
				if reflect.DeepEqual(svc.OriginalConfigs[name], curconf.ConfFile) && !reflect.DeepEqual(svc.OriginalConfigs[name], newconf) {
					curconf.ConfFile = newconf
					glog.V(1).Infof("Migrating config file %s for service %s (%s)", name, svc.Name, svc.ID)
					if err := store.Put(ctx, serviceconfigfile.Key(curconf.ID), curconf); err != nil {
						glog.Warningf("Could not migrate config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
					}
				}
			} else {
				// update the file if there are changes since the original
				if !reflect.DeepEqual(newconf, curconf.ConfFile) {
					curconf.ConfFile = newconf
					glog.V(1).Infof("Updating config file %s for service %s (%s)", name, svc.Name, svc.ID)
					if err := store.Put(ctx, serviceconfigfile.Key(curconf.ID), curconf); err != nil {
						glog.Warningf("Could not update config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
					}
				}
			}
			delete(curconfs, name)
		} else {
			glog.V(1).Infof("Adding config file %s for service %s (%s)", name, svc.Name, svc.ID)
			if svcconf, err := serviceconfigfile.New(tenantID, servicePath, newconf); err != nil {
				glog.Warningf("Could not initialize config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
			} else if err := store.Put(ctx, serviceconfigfile.Key(svcconf.ID), svcconf); err != nil {
				glog.Warningf("Could not create config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
			}
		}
	}
	// delete/revert remaining files
	for name, curconf := range curconfs {
		glog.V(1).Infof("Deleting config file %s for service %s (%s)", name, svc.Name, svc.ID)
		if err := store.Delete(ctx, serviceconfigfile.Key(curconf.ID)); err != nil {
			glog.Warningf("Could not delete config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
		}
	}
	return nil
}