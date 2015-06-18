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
	"reflect"
	"strings"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceconfigfile"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/zenoss/glog"
)

// setServiceConfigs fills out the configuration data for the service.
// TODO: this won't work for governed pools
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
	ogconfs := svc.OriginalConfigs
	for name, conf := range confs {
		ogconfs[name] = conf.ConfFile
	}
	svc.ConfigFiles = ogconfs
	return nil
}

// getServiceConfigs returns a map of config files for a service.
func (f *Facade) getServiceConfigs(ctx datastore.Context, tenantID string, servicePath string) (map[string]*serviceconfigfile.SvcConfigFile, error) {
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

// updateServiceConfigs updates config file records for a service.
func (f *Facade) updateServiceConfigs(ctx datastore.Context, svc service.Service, confs map[string]servicedefinition.ConfigFile) error {
	store := serviceconfigfile.NewStore()

	// get the current configs
	if err := f.setServiceConfigs(ctx, &svc); err != nil {
		glog.Errorf("Could not get the current configs for service %s (%s): %s", svc.Name, svc.ID, err)
		return err
	} else if reflect.DeepEqual(confs, svc.ConfigFiles) {
		return nil
	}

	// look for file differences
	tenantID, servicePath, err := f.GetServicePath(ctx, svc.ID)
	if err != nil {
		glog.Errorf("Could not look up path to service %s (%s): %s", svc.ID, svc.Name, err)
		return err
	}
	servicePath = strings.TrimPrefix(servicePath, svc.DeploymentID)

	// add/update conf files
	for name, newconf := range confs {
		// create the conf object to write to the db
		svcconf, err := serviceconfigfile.New(tenantID, servicePath, newconf)
		if err != nil {
			glog.Errorf("Could not create config file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
			return err
		}

		// does this file exist?
		if conf, ok := svc.ConfigFiles[name]; ok {
			delete(svc.ConfigFiles, name)
			if reflect.DeepEqual(newconf, conf) {
				continue
			} else {
				// look up the file id
				if id, err := store.GetFileID(ctx, tenantID, servicePath, name); err != nil {
					glog.Warningf("Could not look up file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
				} else if id != "" {
					svcconf.ID = id
				}
			}
		}

		// write the data
		if err := store.Put(ctx, serviceconfigfile.Key(svcconf.ID), svcconf); err != nil {
			glog.Warningf("Could not update conf %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
		}
	}

	// delete/revert missing files
	for name := range svc.ConfigFiles {
		if id, err := store.GetFileID(ctx, tenantID, servicePath, name); err != nil {
			glog.Warningf("Could not look up file %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
		} else if err := store.Delete(ctx, serviceconfigfile.Key(id)); err != nil {
			glog.Warningf("Could not delete conf %s for service %s (%s): %s", name, svc.Name, svc.ID, err)
		}
	}
	return nil
}