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
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/health"
	zkservice "github.com/control-center/serviced/zzk/service"
)

// GetServiceInstances returns the state of all instances for a particular
// service.
func (f *Facade) GetServiceInstances(ctx datastore.Context, serviceID string) ([]service.Instance, error) {
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		// TODO: error on loading service
		return nil, err
	}
	states, err := f.zzk.GetServiceStates2(svc.PoolID, svc.ID)
	if err != nil {
		// TODO: error on loading states
		return nil, err
	}
	insts := make([]service.Instance, len(states))
	for i, state := range states {
		inst, err := f.getInstance(ctx, state)
		if err != nil {
			// TODO: error on loading state
			return nil, err
		}
		insts[i] = *inst
	}

	return insts, nil
}

// GetHostInstances returns the state of all instances for a particular host.
func (f *Facade) GetHostInstances(ctx datastore.Context, hostID string) ([]service.Instance, error) {
	var hst host.Host
	err := f.hostStore.Get(ctx, host.HostKey(hostID), &hst)
	if err != nil {
		// TODO: error on loading host
		return nil, err
	}
	states, err := f.zzk.GetHostStates(hst.PoolID, hst.ID)
	if err != nil {
		// TODO: error on loading states
		return nil, err
	}
	insts := make([]service.Instance, len(states))
	for i, state := range states {
		inst, err := f.getInstance(ctx, state)
		if err != nil {
			// TODO: error on loading state
			return nil, err
		}
		insts[i] = *inst
	}

	return insts, nil
}

// getInstance calculates the fields of the service instance object.
func (f *Facade) getInstance(ctx datastore.Context, state zkservice.State) (*service.Instance, error) {
	// get the service
	svc, err := f.serviceStore.Get(ctx, state.ServiceID)
	if err != nil {
		// TODO: error on loading service
		return nil, err
	}

	// get the image
	imageID, err := commons.ParseImageID(svc.ImageID)
	if err != nil {
		// TODO: error on loading image
		return nil, err
	}
	imageID.Tag = docker.Latest
	img, err := f.registryStore.Get(ctx, imageID.String())
	if err != nil {
		// TODO: error on searching image registry
		return nil, err
	}

	// get the host
	var hst host.Host
	err = f.hostStore.Get(ctx, host.HostKey(state.HostID), &hst)
	if err != nil {
		// TODO: error on loading host
		return nil, err
	}

	// get the current state
	var curState service.CurrentState
	switch state.DesiredState {
	case service.SVCStop:
		if state.Terminated.After(state.Started) {
			curState = service.Stopped
		} else {
			curState = service.Stopping
		}
	case service.SVCRun:
		if state.Started.After(state.Terminated) && !state.Paused {
			curState = service.Running
		} else {
			curState = service.Starting
		}
	case service.SVCPause:
		if state.Started.After(state.Terminated) {
			if state.Paused {
				curState = service.Paused
			} else {
				curState = service.Pausing
			}
		} else {
			curState = service.Stopped
		}
	default:
		curState = ""
	}

	// get the health status
	hstats := make(map[string]health.Status)
	for name := range svc.HealthChecks {
		key := health.HealthStatusKey{
			ServiceID:       svc.ID,
			InstanceID:      state.InstanceID,
			HealthCheckName: name,
		}
		result, ok := f.hcache.Get(key)
		if ok {
			hstats[name] = result.Status
		} else {
			hstats[name] = health.Unknown
		}
	}

	// TODO: get memory stats
	// TODO: get cpu stats

	inst := &service.Instance{
		ID:           state.InstanceID,
		HostID:       hst.ID,
		HostName:     hst.Name,
		ServiceID:    svc.ID,
		ServiceName:  svc.Name,
		DockerID:     state.DockerID,
		ImageSynced:  img.UUID == state.ImageID,
		DesiredState: state.DesiredState,
		CurrentState: curState,
		HealthStatus: hstats,
	}

	return inst, nil
}
