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
	"fmt"
	"strings"
	"time"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/metrics"
	zkservice "github.com/control-center/serviced/zzk/service"
)

// GetServiceInstances returns the state of all instances for a particular
// service.
func (f *Facade) GetServiceInstances(ctx datastore.Context, since time.Time, serviceID string) ([]service.Instance, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetServiceInstances"))
	logger := plog.WithField("serviceid", serviceID)

	// create an instance map to map instances to their memory usage
	instanceMap := make(map[string]*service.Usage)

	// keep track of the hosts previously looked up
	hostMap := make(map[string]host.Host)

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up service")

		// TODO: expecting wrapped error here
		return nil, err
	}

	logger = logger.WithFields(log.Fields{
		"servicename": svc.Name,
		"poolid":      svc.PoolID,
		"imageid":     svc.ImageID,
	})
	logger.Debug("Loaded service")

	// get the hash of the service image
	imageUUID, err := f.getImageUUID(ctx, svc.ImageID)
	if err != nil {
		return nil, err
	}

	states, err := f.zzk.GetServiceStates(ctx, svc.PoolID, svc.ID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up running instances")
		return nil, err
	}

	logger = logger.WithField("instances", len(states))
	logger.Debug("Found running instances for service")

	metricsreq := make([]metrics.ServiceInstance, len(states))
	insts := make([]service.Instance, len(states))
	for i, state := range states {
		hst, ok := hostMap[state.HostID]
		if !ok {
			if err := f.hostStore.Get(ctx, host.Key(state.HostID), &hst); err != nil {

				logger.WithFields(log.Fields{
					"hostid":     state.HostID,
					"instanceid": state.InstanceID,
				}).WithError(err).Debug("Could not look up host for instance")

				return nil, err
			}
			hostMap[state.HostID] = hst
		}

		inst, err := f.getInstance(ctx, hst, *svc, imageUUID, state)
		if err != nil {
			return nil, err
		}
		metricsreq[i] = metrics.ServiceInstance{ServiceID: inst.ServiceID, InstanceID: inst.InstanceID}
		insts[i] = *inst
		instanceMap[fmt.Sprintf("%s-%d", inst.ServiceID, inst.InstanceID)] = &insts[i].MemoryUsage
	}

	// look up the metrics of all the instances
	if len(metricsreq) > 0 {
		metricsres, err := f.metricsClient.GetInstanceMemoryStats(since, metricsreq...)
		if err != nil {
			logger.WithError(err).Debug("Could not look up memory metrics for instances on service")
		} else {
			for _, metric := range metricsres {
				*instanceMap[fmt.Sprintf("%s-%s", metric.ServiceID, metric.InstanceID)] = service.Usage{
					Cur: metric.Last,
					Max: metric.Max,
					Avg: metric.Average,
				}
			}
		}
	}

	logger.Debug("Loaded instances for service")
	return insts, nil
}

// GetHostInstances returns the state of all instances for a particular host.
func (f *Facade) GetHostInstances(ctx datastore.Context, since time.Time, hostID string) ([]service.Instance, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetHostInstances"))
	logger := plog.WithField("hostid", hostID)

	// create an instance map to map instances to their memory usage
	instanceMap := make(map[string]*service.Usage)

	// keep track of the services previously looked up
	svcMap := make(map[string]service.Service)

	// keep track of the images previously looked up
	imgMap := make(map[string]string)

	var hst host.Host
	err := f.hostStore.Get(ctx, host.Key(hostID), &hst)
	if err != nil {

		logger.WithError(err).Debug("Could not look up host")

		// TODO: expecting wrapped error here
		return nil, err
	}

	logger.Debug("Loaded host")

	states, err := f.zzk.GetHostStates(ctx, hst.PoolID, hst.ID)
	if err != nil {

		logger.WithError(err).Debug("Could not look up running instances")

		return nil, err
	}

	logger = logger.WithField("instances", len(states))
	logger.Debug("Found running instances for services")

	metricsreq := make([]metrics.ServiceInstance, len(states))
	insts := make([]service.Instance, len(states))
	for i, state := range states {

		st8log := logger.WithFields(log.Fields{
			"serviceid":  state.ServiceID,
			"instanceid": state.InstanceID,
		})

		svc, ok := svcMap[state.ServiceID]
		if !ok {
			s, err := f.serviceStore.Get(ctx, state.ServiceID)
			if err != nil {
				st8log.WithError(err).Debug("Could not look up service for instance")
				return nil, err
			}
			svc = *s
			svcMap[state.ServiceID] = svc
		}

		imageUUID, ok := imgMap[svc.ImageID]
		if !ok {
			imageUUID, err = f.getImageUUID(ctx, svc.ImageID)
			if err != nil {
				return nil, err
			}
			imgMap[svc.ImageID] = imageUUID
		}

		inst, err := f.getInstance(ctx, hst, svc, imageUUID, state)
		if err != nil {
			return nil, err
		}
		metricsreq[i] = metrics.ServiceInstance{ServiceID: inst.ServiceID, InstanceID: inst.InstanceID}
		insts[i] = *inst
		instanceMap[fmt.Sprintf("%s-%d", inst.ServiceID, inst.InstanceID)] = &insts[i].MemoryUsage
	}

	if len(metricsreq) > 0 {
		// look up the metrics of all the instances
		metricsres, err := f.metricsClient.GetInstanceMemoryStats(since, metricsreq...)
		if err != nil {
			logger.WithError(err).Debug("Could not look up memory metrics for instances on service")
		} else {
			for _, metric := range metricsres {
				*instanceMap[fmt.Sprintf("%s-%s", metric.ServiceID, metric.InstanceID)] = service.Usage{
					Cur: metric.Last,
					Max: metric.Max,
					Avg: metric.Average,
				}
			}
		}
	}

	logger.Debug("Loaded instances for host")
	return insts, nil
}

// getInstance calculates the fields of the service instance object.
func (f *Facade) getInstance(ctx datastore.Context, hst host.Host, svc service.Service, imageUUID string, state zkservice.State) (*service.Instance, error) {
	logger := plog.WithFields(log.Fields{
		"hostid":     state.HostID,
		"serviceid":  state.ServiceID,
		"instanceid": state.InstanceID,
	})

	// get the current state
	var curState service.InstanceCurrentState
	curState = service.InstanceCurrentState(state.CurrentStateContainer.Status)

	if svc.EmergencyShutdown {
		if curState == service.StateStopped {
			curState = service.StateEmergencyStopped
		} else if curState == service.StateStopping {
			curState = service.StateEmergencyStopping
		}
	}

	logger.Debug("Calulated service status")

	svch := service.BuildServiceHealth(svc)

	inst := &service.Instance{
		InstanceID:    state.InstanceID,
		HostID:        hst.ID,
		HostName:      hst.Name,
		ServiceID:     svc.ID,
		ServiceName:   svc.Name,
		ContainerID:   state.ContainerID,
		ImageSynced:   imageUUID == state.ImageUUID,
		DesiredState:  state.DesiredState,
		CurrentState:  curState,
		HealthStatus:  f.getInstanceHealth(svch, state.InstanceID),
		RAMCommitment: int64(svc.RAMCommitment.Value),
		Scheduled:     state.Scheduled,
		Started:       state.Started,
		Terminated:    state.Terminated,
	}
	logger.Debug("Loaded service instance")

	return inst, nil
}

// getImageUUID returns the hash of the latest image given the name
func (f *Facade) getImageUUID(ctx datastore.Context, imageName string) (string, error) {
	logger := plog.WithField("imagename", imageName)

	if imageName == "" {
		logger.Debug("Image name not specified")
		return "", nil
	}

	imageID, err := commons.ParseImageID(imageName)
	if err != nil {
		logger.WithError(err).Debug("Could not parse service image")
		return "", err
	}
	imageID.Tag = docker.Latest
	logger = logger.WithField("imagename", imageID.String())
	logger.Debug("Parsed service image")

	imageData, err := f.registryStore.Get(ctx, imageID.String())
	if err != nil {
		logger.WithError(err).Debug("Could not look up service image in image registry")
		return "", err
	}

	logger.WithField("imageuuid", imageData.UUID).Debug("Loaded image uuid")
	return imageData.UUID, nil
}

// GetAggregateServices returns the aggregated states of a bulk of services
func (f *Facade) GetAggregateServices(ctx datastore.Context, since time.Time, serviceIDs []string) ([]service.AggregateService, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetAggregateServices"))
	logger := plog.WithField("serviceids", strings.Join(serviceIDs, ","))

	// Create an instance map to map instances to their memory usage.  This is
	// so that we only have to make one call to query service to get metrics
	// for all the instances.
	instanceMap := make(map[string]*service.Usage)

	// Set up an array containing all the metrics to collect
	var metricsreq []metrics.ServiceInstance

	// Results are for saving the metric data that will be returned to the
	// caller.
	results := make([]service.AggregateService, len(serviceIDs))

	for i, serviceID := range serviceIDs {
		svclog := logger.WithField("serviceid", serviceID)

		svc, err := f.serviceStore.GetServiceHealth(ctx, serviceID)
		if datastore.IsErrNoSuchEntity(err) {

			// If the service is not found, set the NotFound boolean to true
			// and continue
			results[i] = service.AggregateService{
				ServiceID: serviceID,
				NotFound:  true,
			}
			svclog.Debug("Service not found")
			continue
		} else if err != nil {
			svclog.WithError(err).Debug("Could not retrieve service")
			return nil, err
		}

		// Get all the state ids running on that service
		stateIDs, err := f.zzk.GetServiceStateIDs(svc.PoolID, svc.ID)
		if err != nil {
			svclog.WithError(err).Debug("Could not retrieve instances for service")
			return nil, err
		}

		// set up the aggregated service object
		results[i] = service.AggregateService{
			ServiceID:         serviceID,
			Name:              svc.Name,
			DesiredState:      service.DesiredState(svc.DesiredState),
			CurrentState:      service.ServiceCurrentState(svc.CurrentState),
			Status:            make([]service.StatusInstance, len(stateIDs)),
			NotFound:          false,
			EmergencyShutdown: svc.EmergencyShutdown,
			RAMCommitment:     int64(svc.RAMCommitment.Value),
		}

		// set up the status of each instance
		for j, stateID := range stateIDs {

			// report the instance id and the health
			results[i].Status[j] = service.StatusInstance{
				InstanceID:   stateID.InstanceID,
				HealthStatus: f.getInstanceHealth(svc, stateID.InstanceID),
			}

			// append a request to the metrics query for this instance
			metricsreq = append(metricsreq, metrics.ServiceInstance{
				ServiceID:  serviceID,
				InstanceID: stateID.InstanceID,
			})

			// add the memory usage response
			instanceMap[fmt.Sprintf("%s-%d", serviceID, stateID.InstanceID)] = &results[i].Status[j].MemoryUsage
		}
	}

	// look up the metrics of all the instances
	metricsres, err := f.metricsClient.GetInstanceMemoryStats(since, metricsreq...)
	if err != nil {
		logger.WithError(err).Debug("Could not look up memory metrics for instances on service")
	} else {
		for _, metric := range metricsres {
			*instanceMap[fmt.Sprintf("%s-%s", metric.ServiceID, metric.InstanceID)] = service.Usage{
				Cur: metric.Last,
				Max: metric.Max,
				Avg: metric.Average,
			}
		}
	}

	logger.Debug("Loaded aggregate service instance data")
	return results, nil
}

// getInstanceHealth returns the health of the instance of a given service
func (f *Facade) getInstanceHealth(svch *service.ServiceHealth, instanceID int) map[string]health.Status {
	hstats := make(map[string]health.Status)
	for name := range svch.HealthChecks {
		key := health.HealthStatusKey{
			ServiceID:       svch.ID,
			InstanceID:      instanceID,
			HealthCheckName: name,
		}
		result, ok := f.hcache.Get(key)
		if ok {
			hstats[name] = result.Status
		} else {
			hstats[name] = health.Unknown
		}
	}
	return hstats
}

// GetHostStrategyInstances returns the strategy objects of all the instances
// running on a host.
func (f *Facade) GetHostStrategyInstances(ctx datastore.Context, hosts []host.Host) ([]*service.StrategyInstance, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.GetHostStrategyInstances"))

	svcMap := make(map[string]service.StrategyInstance)
	insts := make([]*service.StrategyInstance, 0)

	for _, host := range hosts {
		logger := plog.WithField("hostid", host.ID)

		states, err := f.zzk.GetHostStates(ctx, host.PoolID, host.ID)
		if err != nil {

			logger.WithError(err).Debug("Could not look up running instances")
			return nil, err
		}

		logger.WithField("instances", len(states))
		logger.Debug("Found running instances for services")

		for _, state := range states {

			inst, ok := svcMap[state.ServiceID]
			if !ok {
				// FIXME: replace with new service-strategy-store that
				// returns service.StrategyInstance instead of a full Service obj
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
			insts = append(insts, &inst)
		}

		logger.Debug("Loaded instances for host")
	}

	return insts, nil
}

// StopServiceInstance stops a particular service instance
func (f *Facade) StopServiceInstance(ctx datastore.Context, serviceID string, instanceID int) error {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.StopServiceInstance"))
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not look up service")
		return err
	}

	if err := f.zzk.StopServiceInstance(svc.PoolID, svc.ID, instanceID); err != nil {
		logger.WithError(err).Debug("Could not stop service instance")
		return err
	}

	logger.Debug("Stopped service instance")
	return nil
}

// LocateServiceInstance returns host and container information about a service instance
func (f *Facade) LocateServiceInstance(ctx datastore.Context, serviceID string, instanceID int) (*service.LocationInstance, error) {
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.LocateServiceInstance"))
	logger := plog.WithFields(log.Fields{
		"serviceid":  serviceID,
		"instanceid": instanceID,
	})

	svc, err := f.serviceStore.Get(ctx, serviceID)
	if err != nil {
		logger.WithError(err).Debug("Could not look up service")
		return nil, err
	}

	state, err := f.zzk.GetServiceState(ctx, svc.PoolID, svc.ID, instanceID)
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
	defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.SendDockerAction"))
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
