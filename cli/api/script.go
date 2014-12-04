// Copyright 2014 The Serviced Authors.
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

package api

import (
	"fmt"
	"os"
	"path"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/script"
)

// ScriptRun
func (a *api) ScriptRun(fileName string, config *script.Config) error {

	initConfig(config, a)
	r, err := script.NewRunnerFromFile(fileName, config)
	if err != nil {
		return err
	}
	return r.Run()
}

func (a *api) ScriptParse(fileName string, config *script.Config) error {
	_, err := script.NewRunnerFromFile(fileName, config)
	return err
}

func initConfig(config *script.Config, a *api) {
	config.Snapshot = a.AddSnapshot
	config.Restore = cliRestore(a)
	config.TenantLookup = cliTenantLookup(a)
	config.SvcIDFromPath = cliServiceIDFromPath(a)
	config.SvcStart = cliServiceStart(a)
}

func cliServiceStart(a *api) script.ServiceStart {
	return func(svcID string) error {
		svcConfig := SchedulerConfig{ServiceID: svcID, AutoLaunch: false}
		if _, err := a.StartService(svcConfig); err != nil {
			return err
		}
		return nil
	}
}

func cliTenantLookup(a *api) script.TenantIDLookup {
	return func(svcID string) (string, error) {
		client, err := a.connectDAO()
		if err != nil {
			return "", err
		}
		var tID string
		err = client.GetTenantId(svcID, &tID)
		if err != nil {
			return "", err
		}
		return tID, nil
	}
}

func cliRestore(a *api) script.SnapshotRestore {
	return func(tenantID, snapshotID string) error {
		client, err := a.connectDAO()
		if err != nil {
			return fmt.Errorf("please restore snaphsot %s manually: %v", snapshotID, err)
		}
		var svcs []service.Service
		serviceRequest := dao.ServiceRequest{
			TenantID: tenantID,
		}
		if err := client.GetServices(serviceRequest, &svcs); err != nil {
			return fmt.Errorf("please restore snaphsot %s manually: %v", snapshotID, err)
		}
		runningServices := make(map[string]string, 0)
		//find all services that should be running
		for _, svc := range svcs {
			if svc.DesiredState == int(service.SVCRun) {
				runningServices[svc.ID] = svc.Name
			}
		}
		//shut the app down recursively
		req := dao.ScheduleServiceRequest{ServiceID: tenantID, AutoLaunch: true}
		var count int
		if err := client.StopService(req, &count); err != nil {
			return fmt.Errorf("could not stop service, please restore snaphsot %s manually: %v", snapshotID, err)
		}

		if err := a.Rollback(snapshotID); err != nil {
			return fmt.Errorf("could not restore snapshot %s, please restore manually: %v", snapshotID, err)
		}
		//try to restart previously running services
		for runningID, name := range runningServices {
			req = dao.ScheduleServiceRequest{ServiceID: runningID, AutoLaunch: false}
			if err := client.StartService(req, &count); err != nil {
				fmt.Fprintf(os.Stderr, "could not restart service %s %s after restore: %v\n", runningID, name, err)
			}
		}
		return nil
	}
}

func cliServiceIDFromPath(a *api) script.ServiceIDFromPath {
	return func(tenantID string, svcPath string) (string, error) {
		client, err := a.connectDAO()
		if err != nil {
			return "", err
		}
		var svcs []service.Service
		serviceRequest := dao.ServiceRequest{
			TenantID: tenantID,
		}
		if err := client.GetServices(serviceRequest, &svcs); err != nil {
			return "", err
		}

		svcMap := make(map[string]service.Service)
		for _, svc := range svcs {
			svcMap[svc.ID] = svc
		}

		// recursively build full path for all services
		pathmap := make(map[string]string) //path to service id
		for _, svc := range svcs {
			fullpath := svc.Name
			parentServiceID := svc.ParentServiceID

			for parentServiceID != "" {
				fullpath = path.Join(svcMap[parentServiceID].Name, fullpath)
				parentServiceID = svcMap[parentServiceID].ParentServiceID
			}
			pathmap[fullpath] = svc.ID
		}
		svcID, found := pathmap[svcPath]
		if !found {
			return "", fmt.Errorf("did not find service %s", svcPath)
		}
		return svcID, nil
	}
}
