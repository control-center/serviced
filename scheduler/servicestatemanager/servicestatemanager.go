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

	"fmt"
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/logging"
)

var plog = logging.PackageLogger()

var (
	ErrBadTenantID       = errors.New("Unrecognized tenant ID")
	ErrDuplicateTenantID = errors.New("A tenant with this ID already exists")
	ErrBatchQueueEmpty   = errors.New("Tenant service state queue is empty")
)

// ServiceStateChangeBatch represents a batch of services with the same
// desired state that will be operated on by a ServiceStateManager
type ServiceStateChangeBatch struct {
	services     []*service.Service
	desiredState service.DesiredState
	tenantID     string
	emergency    bool
}

// PendingServiceStateChangeBatch represents a batch of services with the same
// desired state that is the current batch to be processed by a ServiceStateManager
type PendingServiceStateChangeBatch struct {
	services     map[string]CancellableService
	desiredState service.DesiredState
	tenantID     string
	emergency    bool
}

// CancellableService is a service whose scheduling may be canceled by a channel
type CancellableService struct {
	*service.Service
	Cancel chan interface{}
}

type ServiceStateQueue struct {
	sync.RWMutex
	batchQueue   []ServiceStateChangeBatch
	currentBatch PendingServiceStateChangeBatch
	changed      chan bool
	facade       Facade
}

// ServiceStateManager intelligently schedules batches of services with zookeeper
type ServiceStateManager struct {
	sync.RWMutex
	facade                 Facade
	ctx                    datastore.Context
	serviceRunLevelTimeout time.Duration
	tenantQueues           map[string]*ServiceStateQueue
	tenantShutDowns        map[string]chan<- interface{}
}

func (b ServiceStateChangeBatch) String() string {
	svcStr := ""
	for _, svc := range b.services {
		svcStr += fmt.Sprintf(`&service.Service{
			ID: %v,
			DesiredState: %v,
			EmergencyShutdownLevel: %v,
			StartLevel: %v,
		},
		`, svc.ID, svc.DesiredState, svc.EmergencyShutdownLevel, svc.StartLevel)
	}

	return fmt.Sprintf(`ServiceStateChangeBatch{
	services: []*service.Service{
		%v
	},
	desiredState: %v,
	emergency: %v,
}`, svcStr, b.desiredState, b.emergency)
}

// NewServiceStateManager creates a new, initialized ServiceStateManager
func NewServiceStateManager(facade Facade, ctx datastore.Context, runLevelTimeout time.Duration) *ServiceStateManager {
	return &ServiceStateManager{
		RWMutex: sync.RWMutex{},
		facade:  facade,
		ctx:     ctx,
		serviceRunLevelTimeout: runLevelTimeout,
		tenantQueues:           make(map[string]*ServiceStateQueue),
		tenantShutDowns:        make(map[string]chan<- interface{}),
	}
}

func (s *ServiceStateManager) Stop() {
	s.Lock()
	defer s.Unlock()
	for _, cancel := range s.tenantShutDowns {
		close(cancel)
	}
}

func (s *ServiceStateManager) Start() error {
	tenantIDs, err := s.facade.GetTenantIDs(s.ctx)
	if err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		s.AddTenant(tenantID)
	}

	return nil
}

func (s *ServiceStateManager) AddTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.tenantQueues[tenantID]; ok {
		return ErrDuplicateTenantID
	}

	queue := &ServiceStateQueue{
		currentBatch: PendingServiceStateChangeBatch{},
		changed:      make(chan bool),
		facade:       s.facade,
	}
	shutdown := make(chan interface{})
	s.tenantQueues[tenantID] = queue
	s.tenantShutDowns[tenantID] = shutdown

	go s.tenantLoop(tenantID, shutdown)

	return nil
}

// ScheduleServices merges and reconciles a slice of services with the
// ServiceStateChangeBatches in the ServiceStateManager's queue
func (s *ServiceStateManager) ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error {
	s.RLock()
	var queue *ServiceStateQueue
	queue, ok := s.tenantQueues[tenantID]
	s.RUnlock()

	if !ok {
		return ErrBadTenantID
	}

	queue.Lock()
	defer queue.Unlock()

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
		tenantID:     tenantID,
		desiredState: desiredState,
		emergency:    emergency,
	}

	// reconcile the new batch against all batches in queue
	newBatch = queue.reconcileWithBatchQueue(newBatch)

	if len(newBatch.services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	// reconcile with the pending batch
	newBatch = queue.reconcileWithPendingBatch(newBatch)

	if len(newBatch.services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	var err error
	// Merge this into the queue
	if newBatch.emergency {
		err = queue.mergeEmergencyBatch(newBatch)
	} else {
		err = queue.mergeBatch(newBatch)
	}

	// Signal update or exit
	select {
	case queue.changed <- true:
	default:
	}

	return err
}

func (s *ServiceStateManager) RemoveTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()
	cancel, ok := s.tenantShutDowns[tenantID]
	if !ok {
		return ErrBadTenantID
	}

	close(cancel)
	delete(s.tenantShutDowns, tenantID)
	delete(s.tenantQueues, tenantID)

	return nil
}

func (s *ServiceStateQueue) reconcileWithBatchQueue(new ServiceStateChangeBatch) ServiceStateChangeBatch {
	var newBatchQueue []ServiceStateChangeBatch
	updated := new
	for _, batch := range s.batchQueue {
		updated = batch.reconcile(updated)
		if len(batch.services) > 0 {
			newBatchQueue = append(newBatchQueue, batch)
		}
	}

	s.batchQueue = newBatchQueue

	plog.WithFields(logrus.Fields{
		"newDesiredState": new.desiredState,
		"newEmergency":    new.emergency,
	}).Debug("finished reconcile with batch queue")
	return updated
}

func (b *ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	if b.tenantID != newBatch.tenantID {
		// Nothing to do
		return newBatch
	}

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

func (s *ServiceStateQueue) reconcileWithPendingBatch(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
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

func (s *ServiceStateQueue) mergeEmergencyBatch(newBatch ServiceStateChangeBatch) error {
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

func (s *ServiceStateQueue) mergeBatch(newBatch ServiceStateChangeBatch) error {
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
	if len(batches) < 1 {
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

// DrainQueue blocks until the queue is empty.
func (s *ServiceStateManager) DrainQueue(tenantID string) error {
	s.RLock()
	queue, ok := s.tenantQueues[tenantID]
	s.RUnlock()

	if !ok {
		return ErrBadTenantID
	}

	for {
		empty := func() bool {
			queue.RLock()
			defer queue.RUnlock()
			return len(queue.batchQueue) == 0 && len(queue.currentBatch.services) == 0
		}()

		if empty {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ServiceStateManager) tenantLoop(tenantID string, cancel <-chan interface{}) {
	logger := plog.WithField("tenantid", tenantID)
	s.RLock()
	queue := s.tenantQueues[tenantID]
	s.RUnlock()
	for {
		batch, err := queue.getNextBatch()
		if err == nil {
			batchLogger := logger.WithFields(
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

			_, serr := s.facade.ScheduleServiceBatch(s.ctx, services, tenantID, batch.desiredState)
			if serr != nil {
				batchLogger.WithError(serr).Error("Error scheduling services to stop")
			}

			// Wait on this batch, with cancel option
			desiredState := batch.desiredState
			if desiredState == service.SVCRestart {
				desiredState = service.SVCRun
			}
			if err := queue.waitServicesWithTimeout(desiredState, batch.services, s.serviceRunLevelTimeout); err != nil {
				batchLogger.WithError(err).Error("Error waiting for service batch to reach desired state")
			}
		} else {
			if err != ErrBatchQueueEmpty {
				logger.WithError(err).Error("Error getting next batch")
			}

			switch {
			case <-cancel:
				return
			case <-queue.changed:
			}
		}
	}
}

func (q *ServiceStateQueue) getNextBatch() (b PendingServiceStateChangeBatch, err error) {
	q.Lock()
	defer q.Unlock()
	if len(q.batchQueue) > 0 {
		b = PendingServiceStateChangeBatch{
			services:     make(map[string]CancellableService),
			desiredState: q.batchQueue[0].desiredState,
			emergency:    q.batchQueue[0].emergency,
		}

		for _, svc := range q.batchQueue[0].services {
			b.services[svc.ID] = CancellableService{svc, make(chan interface{})}
		}

		q.batchQueue = q.batchQueue[1:]
	} else {
		err = ErrBatchQueueEmpty
	}

	q.currentBatch = b
	return
}

func (s *ServiceStateQueue) waitServicesWithTimeout(dstate service.DesiredState, services map[string]CancellableService, timeout time.Duration) error {
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
		s.RLock()
		for _, svc := range services {
			s.cancelPending(svc.ID)
		}
		s.RUnlock()
		return errors.New("Timeout waiting for services")
	}
}

func (s *ServiceStateQueue) cancelPending(serviceID string) {
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

func (s *ServiceStateQueue) waitServicesWithCancel(dstate service.DesiredState, services map[string]CancellableService) error {
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
