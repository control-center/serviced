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

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/health"
	zkservice "github.com/control-center/serviced/zzk/service2"
)

// GetServiceInstances returns the state of all instances for a particular
// service.
func (f *Facade) GetServiceInstances(ctx datastore.Context, serviceID string) ([]service.Instance, error) {
	logger := plog.WithField("serviceid", serviceID)

	hostMap := make(map[string]host.Host)

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up service")

		// TODO: expecting wrapped error here
		return nil, err
	}

	logger.Debug("Loaded service")

	states, err := f.zzk.GetServiceStates2(svc.PoolID, svc.ID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up running instances")
		return nil, err
	}

	logger = logger.WithField("instances", len(states))
	logger.Debug("Found running instances for service")

	insts := make([]service.Instance, len(states))
	for i, state := range states {
		hst, ok := hostMap[state.HostID]
		if !ok {
			if err := f.hostStore.Get(ctx, host.HostKey(state.HostID), &hst); err != nil {

				logger.WithFields(log.Fields{
					"hostid":     state.HostID,
					"instanceid": state.InstanceID,
				}).WithError(err).Debug("Could not look up host for instance")

				return nil, err
			}
			hostMap[state.HostID] = hst
		}

		inst, err := f.getInstance(ctx, hst, *svc, state)
		if err != nil {
			return nil, err
		}
		insts[i] = *inst
	}

	logger.Debug("Loaded instances for service")

	return insts, nil
}

// GetHostInstances returns the state of all instances for a particular host.
func (f *Facade) GetHostInstances(ctx datastore.Context, hostID string) ([]service.Instance, error) {
	logger := plog.WithField("hostid", hostID)

	svcMap := make(map[string]service.Service)

	var hst host.Host
	err := f.hostStore.Get(ctx, host.HostKey(hostID), &hst)
	if err != nil {

		logger.WithError(err).Debug("Could not look up host")

		// TODO: expecting wrapped error here
		return nil, err
	}

	logger.Debug("Loaded host")

	states, err := f.zzk.GetHostStates(hst.PoolID, hst.ID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up running instances")

		return nil, err
	}

	logger = logger.WithField("instances", len(states))
	logger.Debug("Found running instances for services")

	insts := make([]service.Instance, len(states))
	for i, state := range states {

		svc, ok := svcMap[state.ServiceID]
		if !ok {
			s, err := f.serviceStore.Get(ctx, state.ServiceID)
			if err != nil {

				logger.WithFields(log.Fields{
					"serviceid":  state.ServiceID,
					"instanceid": state.InstanceID,
				}).WithError(err).Debug("Could not look up service for instance")

				return nil, err
			}
			svc = *s
			svcMap[state.ServiceID] = svc
		}

		inst, err := f.getInstance(ctx, hst, svc, state)
		if err != nil {
			return nil, err
		}
		insts[i] = *inst
	}

	logger.Debug("Loaded instances for host")

	return insts, nil
}

// getInstance calculates the fields of the service instance object.
func (f *Facade) getInstance(ctx datastore.Context, hst host.Host, svc service.Service, state zkservice.State) (*service.Instance, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":     state.HostID,
		"serviceid":  state.ServiceID,
		"instanceid": state.InstanceID,
	})

	// check the image
	imageSynced, err := func(imageName, imageUUID string) (bool, error) {
		imgLogger := logger.WithField("imagename", imageName)

		imageID, err := commons.ParseImageID(imageName)
		if err != nil {

			imgLogger.WithError(err).Debug("Could not parse service image")
			return false, err
		}
		imgLogger.Debug("Parsed service image")

		imageID.Tag = docker.Latest
		imageData, err := f.registryStore.Get(ctx, imageID.String())
		if err != nil {

			imgLogger.WithError(err).Debug("Could not look up service image in registry")

			// TODO: expecting wrapped error here
			return false, err
		}
		imgLogger.Debug("Loaded service image from registry")

		return imageData.UUID == imageUUID, nil
	}(svc.ImageID, state.ImageID)

	if err != nil {
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
	logger.Debug("Calulated service status")

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
	logger.Debug("Loaded service health")

	inst := &service.Instance{
		ID:           state.InstanceID,
		HostID:       hst.ID,
		HostName:     hst.Name,
		ServiceID:    svc.ID,
		ServiceName:  svc.Name,
		ContainerID:  state.ContainerID,
		ImageSynced:  imageSynced,
		DesiredState: state.DesiredState,
		CurrentState: curState,
		HealthStatus: hstats,
		Scheduled:    state.Scheduled,
		Started:      state.Started,
		Terminated:   state.Terminated,
	}
	logger.Debug("Loaded service instance")

	return inst, nil
}

// GetHostStrategyInstances returns the strategy objects of all the instances
// running on a host.
func (f *Facade) GetHostStrategyInstances(ctx datastore.Context, hostIDs ...string) ([]service.StrategyInstance, error) {

	svcMap := make(map[string]service.StrategyInstance)
	insts := make([]service.StrategyInstance, 0)

	for _, hostID := range hostIDs {
		logger := plog.WithField("hostid", hostID)

		var hst host.Host
		err := f.hostStore.Get(ctx, host.HostKey(hostID), &hst)
		if err != nil {

			logger.WithError(err).Debug("Could not look up host")

			// TODO: expecting wrapped error here
			return nil, err
		}

		logger.Debug("Loaded host")

		states, err := f.zzk.GetHostStates(hst.PoolID, hst.ID)
		if err != nil {

			logger.WithError(err).Debug("Could not look up running instances")
			return nil, err
		}

		logger.WithField("instances", len(states))
		logger.Debug("Found running instances for services")

		for _, state := range states {

			inst, ok := svcMap[state.ServiceID]
			if !ok {
				s, err := f.serviceStore.Get(ctx, state.ServiceID)
				if err != nil {

					logger.WithFields(log.Fields{
						"serviceid":  state.ServiceID,
						"instanceid": state.InstanceID,
					}).WithError(err).Debug("Could not look up service for instance")

					return nil, err
				}
				inst = service.StrategyInstance{
					ServiceID:     s.ID,
					CPUCommitment: int(s.CPUCommitment),
					RAMCommitment: s.RAMCommitment.Value,
					HostPolicy:    s.HostPolicy,
				}
				svcMap[state.ServiceID] = inst
			}

			inst.HostID = state.HostID
			insts = append(insts, inst)
		}

		logger.Debug("Loaded instances for host")
	}

	return insts, nil
}

// StopServiceInstance stops a particular service instance
func (f *Facade) StopServiceInstance(ctx datastore.Context, serviceID string, instanceID int) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not look up service")
		return err
	}

	if err := f.zzk.StopServiceInstance2(svc.PoolID, svc.ID, instanceID); err != nil {
		logger.WithError(err).Debug("Could not stop service instance")
		return err
	}

	logger.Debug("Stopped service instance")
	return nil
}

// LocateServiceInstance returns host and container information about a service
// instance
func (f *Facade) LocateServiceInstance(ctx datastore.Context, serviceID string, instanceID int) (*service.LocationInstance, error) {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not look up service")
		return nil, err
	}

	state, err := f.zzk.GetServiceState(svc.PoolID, svc.ID, instanceID)
	if err != nil {
		logger.WithError(err).Debug("Could not locate service instance")
		return nil, err
	}

	logger.Debug("Found service instance")
	return &service.LocationInstance{
		HostID:      state.HostID,
		HostIP:      state.HostIP,
		ContainerID: state.ContainerID,
	}, nil
}

// SendDockerAction locates a service instance and sends an action to it
func (f *Facade) SendDockerAction(ctx datastore.Context, serviceID string, instanceID int, action string, args []string) error {
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceID": instanceID,
		"action":     action,
		"args":       args,
	})

	// get the service
	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not look up service")
		return err
	}

	// evaluate the service actions template
	get := func(serviceID string) (service.Service, error) {
		s, err := f.serviceStore.Get(ctx, serviceID)
		if err != nil {
			return service.Service{}, nil
		}
		return *s, nil
	}

	getchild := func(parentID, childName string) (service.Service, error) {
		s, err := f.serviceStore.FindChildService(ctx, svc.DeploymentID, parentID, childName)
		if err != nil {
			return service.Service{}, nil
		}
		return *s, nil
	}

	if err := svc.EvaluateActionsTemplate(get, getchild, instanceID); err != nil {
		logger.WithError(err).Debug("Could not evaluate service actions template")
		return err
	}

	// find the service action
	command, ok := svc.Actions[action]
	if !ok {
		logger.Debug("Command not found for action")
		return errors.New("command not found for action")
	}

	// send the command
	if err := f.zzk.SendDockerAction(svc.PoolID, serviceID, instanceID, command, args); err != nil {
		logger.WithError(err).Debug("Unable to send docker action")
		return err
	}

	logger.Debug("Submitted docker action")
	return nil
}
