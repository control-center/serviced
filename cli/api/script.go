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
	"math"
	"path"
	"strings"
	"time"

	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/script"
)

// ScriptRun
func (a *api) ScriptRun(fileName string, config *script.Config, stopChan chan struct{}) error {

	initConfig(config, a)
	r, err := script.NewRunnerFromFile(fileName, config)
	if err != nil {
		return err
	}

	return r.Run(stopChan)
}

func (a *api) ScriptParse(fileName string, config *script.Config) error {
	_, err := script.NewRunnerFromFile(fileName, config)
	return err
}

func initConfig(config *script.Config, a *api) {
	config.Snapshot = func(serviceID, message string, tag string) (string, error) {
		return a.AddSnapshot(SnapshotConfig{ServiceID: serviceID, Message: message, Tag: tag})
	}
	config.Restore = a.Rollback
	config.TenantLookup = cliTenantLookup(a)
	config.SvcIDFromPath = cliServiceIDFromPath(a)
	config.SvcStart = cliServiceControl(a.StartService)
	config.SvcStop = cliServiceControl(a.StopService)
	config.SvcRestart = cliServiceControl(a.RestartService)
	config.SvcMigrate = cliServiceMigrate(a)
	config.SvcWait = cliServiceWait(a)
	config.Commit = func(containerID string) (string, error) {
		return a.AddSnapshot(SnapshotConfig{DockerID: containerID})
	}
	config.SvcUse = cliServiceUse(a)
}

func cliServiceUse(a *api) script.ServiceUse {
	return func(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error) {
		client, err := a.connectMaster()
		if err != nil {
			return "", err
		}
		resp, err := client.ServiceUse(serviceID, imageID, registry, replaceImgs, noOp)
		if err != nil {
			return "", err
		}
		return resp, nil
	}
}

func cliServiceControl(svcControlMethod ServiceStateController) script.ServiceControl {
	return func(svcID string, recursive bool) error {
		svcConfig := SchedulerConfig{ServiceID: svcID, AutoLaunch: recursive}
		if _, err := svcControlMethod(svcConfig); err != nil {
			return err
		}
		return nil
	}
}

func cliServiceWait(a *api) script.ServiceWait {
	return func(svcIDs []string, state script.ServiceState, timeout uint32) error {
		client, err := a.connectDAO()
		if err != nil {
			return err
		}

		var desiredState service.DesiredState
		switch state {
		case "started":
			desiredState = service.SVCRun
		case "stopped":
			desiredState = service.SVCStop
		case "paused":
			desiredState = service.SVCPause
		default:
			return fmt.Errorf("Unknown service state %s", state)
		}
		if timeout == 0 {
			timeout = math.MaxUint32
		}
		wsr := dao.WaitServiceRequest{ServiceIDs: svcIDs,
			DesiredState: desiredState,
			Timeout:      time.Duration(timeout) * time.Second}
		if err = client.WaitService(wsr, nil); err != nil {
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
			pathmap[strings.ToLower(fullpath)] = svc.ID
		}
		svcID, found := pathmap[strings.ToLower(svcPath)]
		if !found {
			return "", fmt.Errorf("did not find service %s", svcPath)
		}
		return svcID, nil
	}
}

func cliServiceMigrate(a API) script.ServiceMigrate {
	return func(svcID string, scriptFile string, sdkVersion string) error {
		if _, err := a.RunEmbeddedMigrationScript(svcID, scriptFile, false, sdkVersion); err != nil {
			return fmt.Errorf("Migration failed for service %s: %s", svcID, err)
		}
		return nil
	}
}
