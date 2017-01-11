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

package servicestatemanager

import (
	"errors"
	"sort"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/logging"
)

var plog = logging.PackageLogger()

// ServiceStateChangeBatch represents a batch of services with the same
// desired state that will be operated on by a ServiceStateManager
type ServiceStateChangeBatch struct {
	services     []*service.Service
	desiredState service.DesiredState
	emergency    bool
}

// PendingServiceStateChangeBatch represents a batch of services with the same
// desired state that is the current batch to be processed by a ServiceStateManager
type PendingServiceStateChangeBatch struct {
	services     map[string]CancellableService
	desiredState service.DesiredState
	emergency    bool
}

// CancellableService is a service whose scheduling may be canceled by a channel
type CancellableService struct {
	*service.Service
	Cancel chan interface{}
}

// ServiceStateManager intelligently schedules batches of services with zookeeper
type ServiceStateManager struct {
	sync.RWMutex
	facade                 Facade
	ctx                    datastore.Context
	tenantID               string
	serviceRunLevelTimeout time.Duration
	drainQueue             sync.Mutex
	batchQueue             []ServiceStateChangeBatch
	currentBatch           PendingServiceStateChangeBatch
	changed                chan bool
}

// NewServiceStateManager creates a new, initialized ServiceStateManager
func NewServiceStateManager(facade Facade, ctx datastore.Context, tenantID string, runLevelTimeout time.Duration) *ServiceStateManager {
	return &ServiceStateManager{
		RWMutex:                sync.RWMutex{},
		facade:                 facade,
		ctx:                    ctx,
		tenantID:               tenantID,
		serviceRunLevelTimeout: runLevelTimeout,
		batchQueue:             []ServiceStateChangeBatch{},
		changed:                make(chan bool, 1),
	}
}

// ScheduleServices merges and reconciles a slice of services with the
// ServiceStateChangeBatches in the ServiceStateManager's queue
func (s *ServiceStateManager) ScheduleServices(svcs []*service.Service, desiredState service.DesiredState, emergency bool) error {
	s.Lock()
	defer s.Unlock()

	s.drainQueue.Lock()
	defer s.drainQueue.Unlock()

	// Merge with oldBatch batchQueue
	// 1. If this is emergency, merge with other emergencies and move to front of the queue
	// 2. If any service in this batch is currently in the "pending" batch:
	//    A. If the desired states are the same, leave it pending and remove it from this batch
	//    B. If the desired states are different, cancel the pending request and leave it in this batch
	// 3. If this and the last N batches at the end of the queue all have the same desired state, merge and re-group them
	// 4. If any service in this batch also appears in an earlier batch:
	//    A. If the desired state is the same, leave it in the earlier batch and remove it here
	//    B. If the desired state is different, delete it from the earlier batch and leave it in the new one

	// create a batch from the args
	newBatch := ServiceStateChangeBatch{
		services:     svcs,
		desiredState: desiredState,
		emergency:    emergency,
	}

	// reconcile the new batch against all batches in queue
	newBatch = s.reconcileWithBatchQueue(newBatch)

	if len(newBatch.services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	// reconcile with the pending batch
	newBatch = s.reconcileWithPendingBatch(newBatch)

	if len(newBatch.services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	var err error
	// Merge this into the queue
	if newBatch.emergency {
		err = s.mergeEmergencyBatch(newBatch)
	} else {
		err = s.mergeBatch(newBatch)
	}

	// Signal update or exit
	select {
	case s.changed <- true:
	default:
	}

	return err
}

func (s *ServiceStateManager) reconcileWithBatchQueue(new ServiceStateChangeBatch) ServiceStateChangeBatch {
	updated := new
	for n, batch := range s.batchQueue {
		updated = batch.reconcile(updated)
		if len(batch.services) == 0 {
			// this is no longer a useful batch, purge it from batchQueue
			s.batchQueue = append(s.batchQueue[:n], s.batchQueue[n+1:]...)
		} else {
			s.batchQueue[n] = batch
		}
	}

	plog.WithFields(logrus.Fields{
		"newDesiredState": new.desiredState,
		"newEmergency":    new.emergency,
	}).Debug("finished reconcile with batch queue")
	return updated
}

func (b *ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	oldSvcs := make([]*service.Service, len(b.services))
	copy(oldSvcs, b.services)
	newSvcs := []*service.Service{}
	for _, newSvc := range newBatch.services {
		addNew := true
		// make a copy of the (possibly) updated oldSvcs and iterate over it
		updatedSvcs := make([]*service.Service, len(oldSvcs))
		copy(updatedSvcs, oldSvcs)
		for i, svc := range updatedSvcs {
			if newSvc.ID == svc.ID {
				if b.emergency {
					// this service is going to be stopped, so don't bother queuing it
					addNew = false
				} else if newBatch.emergency {
					// newBatch is going to be brought to the front of the queue on merge,
					// so we can take this service out of the old batch
					oldSvcs = append(oldSvcs[:i], oldSvcs[i+1:]...)
				} else if b.desiredState != newBatch.desiredState {
					// this service has a newer desired state than it does in b,
					// so we can take this service out of old batch
					oldSvcs = append(oldSvcs[:i], oldSvcs[i+1:]...)
				} else {
					// this service exists in b with the same desired state,
					// so don't add it, because it would be redundant
					addNew = false
				}
				break
			}
		}
		if addNew {
			newSvcs = append(newSvcs, newSvc)
		}
	}

	b.services = oldSvcs

	plog.WithFields(logrus.Fields{
		"existingDesiredState": b.desiredState,
		"existingEmergency":    b.emergency,
		"newDesiredState":      newBatch.desiredState,
		"newEmergency":         newBatch.emergency,
	}).Debug("finished reconcile")

	return ServiceStateChangeBatch{
		services:     newSvcs,
		desiredState: newBatch.desiredState,
		emergency:    newBatch.emergency,
	}
}

func (s *ServiceStateManager) reconcileWithPendingBatch(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	var newSvcs []*service.Service
	for _, newSvc := range newBatch.services {
		addNew := true
		for _, pendingSvc := range s.currentBatch.services {
			if newSvc.ID == pendingSvc.ID {
				if newBatch.emergency {
					// this service is going to be stopped,
					// so cancel the one in currentBatch
					s.cancelPending(pendingSvc.ID)
				} else if s.currentBatch.desiredState != newBatch.desiredState {
					// this service has a newer desired state than it does in currentBatch,
					// so cancel the one in currentBatch
					s.cancelPending(pendingSvc.ID)
				} else {
					// this service exists in currentBatch with the same desired state,
					// so don't add it, because it would be redundant
					addNew = false
				}
				break
			}
		}
		if addNew {
			newSvcs = append(newSvcs, newSvc)
		}
	}

	return ServiceStateChangeBatch{
		services:     newSvcs,
		desiredState: newBatch.desiredState,
		emergency:    newBatch.emergency,
	}
}

func (s *ServiceStateManager) mergeEmergencyBatch(newBatch ServiceStateChangeBatch) error {
	if !newBatch.emergency {
		return s.mergeBatch(newBatch)
	}
	// find the last emergency batch in the queue
	lastEmergencyBatch := -1
	for i, batch := range s.batchQueue {
		if batch.emergency {
			lastEmergencyBatch = i
		} else {
			break
		}
	}

	// merge newBatch with the emergency batches at the front of the queue
	newBatches, err := mergeBatches(append(s.batchQueue[:lastEmergencyBatch+1], newBatch))
	if err != nil {
		return err
	}
	s.batchQueue = append(newBatches, s.batchQueue[lastEmergencyBatch+1:]...)
	return nil
}

func (s *ServiceStateManager) mergeBatch(newBatch ServiceStateChangeBatch) error {
	if newBatch.emergency {
		return s.mergeEmergencyBatch(newBatch)
	}
	// merge this with any other batches matching desired state at the end of the queue
	lastBatchToMerge := len(s.batchQueue)
	for i := len(s.batchQueue) - 1; i >= 0; i-- {
		if s.batchQueue[i].desiredState == newBatch.desiredState {
			lastBatchToMerge = i
		} else {
			break
		}
	}

	newBatches, err := mergeBatches(append(s.batchQueue[lastBatchToMerge:], newBatch))
	if err != nil {
		return err
	}
	s.batchQueue = append(s.batchQueue[:lastBatchToMerge], newBatches...)
	return nil
}

func mergeBatches(batches []ServiceStateChangeBatch) ([]ServiceStateChangeBatch, error) {
	if len(batches) <= 1 {
		return batches, nil
	}

	var fullServiceList []*service.Service

	// Make sure all of the batches we're merging have the same desiredState
	// and emergency status
	desiredState := batches[0].desiredState
	emergency := batches[0].emergency
	for _, b := range batches {
		if b.desiredState != desiredState || b.emergency != emergency {
			// TODO make this a real error
			return nil, errors.New("Can't merge batches with different desired states")
		}
		fullServiceList = append(fullServiceList, b.services...)
	}

	// Sort the full list based on desired state
	if desiredState == service.SVCRun {
		// Sort the services by start level
		sort.Sort(service.ByStartLevel{
			Services: fullServiceList,
		})
	} else if desiredState == service.SVCStop && emergency {
		sort.Sort(service.ByEmergencyShutdown{
			Services: fullServiceList,
		})
	} else if desiredState == service.SVCStop {
		sort.Sort(service.ByReverseStartLevel{
			Services: fullServiceList,
		})
	} else if desiredState == service.SVCRestart {
		// TODO: We need to handle this properly
		sort.Sort(service.ByStartLevel{
			Services: fullServiceList,
		})
	} else if desiredState == service.SVCPause {
		sort.Sort(service.ByReverseStartLevel{
			Services: fullServiceList,
		})
	}

	// regroup the services by level
	previousEmergencyLevel := fullServiceList[0].EmergencyShutdownLevel
	previousStartLevel := fullServiceList[0].StartLevel

	newBatches := []ServiceStateChangeBatch{}
	newSvcs := []*service.Service{}

	for _, svc := range fullServiceList {
		currentEmergencyLevel := svc.EmergencyShutdownLevel
		currentStartLevel := svc.StartLevel
		var sameBatch bool

		if emergency {
			// this service should be in the same batch if it has the same emergency level
			sameBatch = currentEmergencyLevel == previousEmergencyLevel
			if sameBatch && currentEmergencyLevel == 0 {
				// For emergency shutdown level 0, we group by reverse start level
				sameBatch = currentStartLevel == previousStartLevel
			}
		} else {
			// this service should be in the same batch if it has the same start level
			sameBatch = currentStartLevel == previousStartLevel
		}

		if sameBatch {
			// add it to newSvcs
			newSvcs = append(newSvcs, svc)
		} else {
			// append our batch from the previous level, start a new batch with svc
			newBatches = append(newBatches, ServiceStateChangeBatch{
				services:     newSvcs,
				desiredState: desiredState,
				emergency:    emergency,
			})
			newSvcs = []*service.Service{svc}
		}
		previousEmergencyLevel = currentEmergencyLevel
		previousStartLevel = currentStartLevel
	}

	// Add the last batch
	newBatches = append(newBatches, ServiceStateChangeBatch{
		services:     newSvcs,
		desiredState: desiredState,
		emergency:    emergency,
	})

	return newBatches, nil
}

// Mergeable compares two ServiceStateChangeBatch to determine if they can merge
func (b ServiceStateChangeBatch) Mergeable(batch ServiceStateChangeBatch) bool {
	return b.desiredState == batch.desiredState && b.emergency == batch.emergency
}

// Mergeable compares a PendingServiceStateChangeBatch to a ServiceStateChangeBatch
// to determine if they can merge
func (b PendingServiceStateChangeBatch) Mergeable(batch ServiceStateChangeBatch) bool {
	return b.desiredState == batch.desiredState && b.emergency == batch.emergency
}

// DrainQueue blocks until the queue is empty.  Call LockQueue first
func (s *ServiceStateManager) DrainQueue() {
	for {
		if len(s.batchQueue) == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

// LockQueue locks the drainQueue
func (s *ServiceStateManager) LockQueue() {
	s.drainQueue.Lock()
}

// UnlockQueue unlocks the drainQueue
func (s *ServiceStateManager) UnlockQueue() {
	s.drainQueue.Unlock()
}

func (s *ServiceStateManager) getNextBatch() (PendingServiceStateChangeBatch, error) {
	s.Lock()
	defer s.Unlock()
	if len(s.batchQueue) > 0 {
		b := PendingServiceStateChangeBatch{
			services:     make(map[string]CancellableService),
			desiredState: s.batchQueue[0].desiredState,
			emergency:    s.batchQueue[0].emergency,
		}

		for _, svc := range s.batchQueue[0].services {
			b.services[svc.ID] = CancellableService{svc, make(chan interface{})}
		}

		s.batchQueue = s.batchQueue[1:]
		s.currentBatch = b
		return b, nil
	}

	return PendingServiceStateChangeBatch{}, errors.New("batchQueue empty")
}

func (s *ServiceStateManager) mainloop(cancel <-chan interface{}) {
	for {
		batch, err := s.getNextBatch()
		if err == nil {
			batchLogger := plog.WithFields(
				logrus.Fields{
					"emergency":    batch.emergency,
					"desiredstate": batch.desiredState,
				})
			// Schedule services for this batch
			var services []*service.Service
			for _, svc := range batch.services {
				services = append(services, svc.Service)
				if batch.emergency {
					// Set EmergencyShutdown to true for this service and update the database
					svc.EmergencyShutdown = true
					uerr := s.facade.UpdateService(s.ctx, *svc.Service)
					if uerr != nil {
						batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
					}
				}
			}

			_, serr := s.facade.ScheduleServiceBatch(s.ctx, services, s.tenantID, batch.desiredState)
			if serr != nil {
				batchLogger.WithError(serr).Error("Error scheduling services to stop")
			}

			// Wait on this batch, with cancel option
			desiredState := batch.desiredState
			if desiredState == service.SVCRestart {
				desiredState = service.SVCRun
			}
			if err := s.waitServicesWithTimeout(desiredState, batch.services, s.serviceRunLevelTimeout); err != nil {
				batchLogger.WithError(err).Error("Error waiting for service batch to reach desired state")
			}
		} else {
			switch {
			case <-cancel:
				return
			case <-s.changed:
			}
		}
	}
}

func (s *ServiceStateManager) waitServicesWithTimeout(dstate service.DesiredState, services map[string]CancellableService, timeout time.Duration) error {
	done := make(chan error)
	cancel := make(chan interface{})
	defer close(cancel)

	go func() {
		err := s.waitServicesWithCancel(dstate, services)
		select {
		case done <- err:
		case <-cancel:
		}
	}()

	timer := time.NewTimer(timeout)
	select {
	case err := <-done:
		return err
	case <-timer.C:
		// close all channels
		s.Lock()
		for _, svc := range services {
			s.cancelPending(svc.ID)
		}
		s.Unlock()
		return errors.New("Timeout waiting for services")
	}
}

func (s *ServiceStateManager) cancelPending(serviceID string) {
	if svc, ok := s.currentBatch.services[serviceID]; ok {
		// Make sure it isn't already closed
		select {
		case <-svc.Cancel:
			return
		default:
			close(svc.Cancel)
		}
	}
}

func (s *ServiceStateManager) waitServicesWithCancel(dstate service.DesiredState, services map[string]CancellableService) error {
	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(svcArg CancellableService) {
			err := s.facade.WaitSingleService(svcArg.Service, dstate, svcArg.Cancel)
			plog.WithError(err).WithFields(logrus.Fields{
				"serviceid":    svcArg.ID,
				"desiredstate": dstate,
			}).Error("Failed to wait for a single service")
			wg.Done()
		}(svc)
	}
	wg.Wait()
	return nil
}
