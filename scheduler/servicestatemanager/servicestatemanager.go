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
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/logging"
)

var plog = logging.PackageLogger()

var (
	ErrBadTenantID             = errors.New("Unrecognized tenant ID")
	ErrDuplicateTenantID       = errors.New("A tenant with this ID already exists")
	ErrBatchQueueEmpty         = errors.New("Tenant service state queue is empty")
	ErrMismatchedDesiredStates = errors.New("Can't merge batches with different desired states")
)

// ServiceStateChangeBatch represents a batch of services with the same
// desired state that will be operated on by a ServiceStateManager
type ServiceStateChangeBatch struct {
	Services     []*service.Service
	DesiredState service.DesiredState
	Emergency    bool
}

// PendingServiceStateChangeBatch represents a batch of services with the same
// desired state that is the current batch to be processed by a ServiceStateManager
type PendingServiceStateChangeBatch struct {
	Services     map[string]CancellableService
	DesiredState service.DesiredState
	Emergency    bool
}

// CancellableService is a service whose scheduling may be canceled by a channel
type CancellableService struct {
	*service.Service
	Cancel chan interface{}
}

type ServiceStateQueue struct {
	sync.RWMutex
	BatchQueue   []ServiceStateChangeBatch
	CurrentBatch PendingServiceStateChangeBatch
	Changed      chan bool
	Facade       Facade
}

// ServiceStateManager intelligently schedules batches of services with zookeeper
type ServiceStateManager struct {
	sync.RWMutex
	Facade                 Facade
	ctx                    datastore.Context
	ServiceRunLevelTimeout time.Duration
	TenantQueues           map[string]*ServiceStateQueue
	TenantShutDowns        map[string]chan<- int
}

func (b ServiceStateChangeBatch) String() string {
	svcStr := ""
	for _, svc := range b.Services {
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
}`, svcStr, b.DesiredState, b.Emergency)
}

// NewServiceStateManager creates a new, initialized ServiceStateManager
func NewServiceStateManager(facade Facade, ctx datastore.Context, runLevelTimeout time.Duration) *ServiceStateManager {
	return &ServiceStateManager{
		RWMutex: sync.RWMutex{},
		Facade:  facade,
		ctx:     ctx,
		ServiceRunLevelTimeout: runLevelTimeout,
		TenantQueues:           make(map[string]*ServiceStateQueue),
		TenantShutDowns:        make(map[string]chan<- int),
	}
}

func (s *ServiceStateManager) Shutdown() {
	s.Lock()
	defer s.Unlock()
	var wg sync.WaitGroup
	for _, cancel := range s.TenantShutDowns {
		wg.Add(1)
		go func(c chan<- int) {
			c <- 0
			wg.Done()
		}(cancel)
	}

	// Wait for all loops to exit
	wg.Wait()
}

func (s *ServiceStateManager) Start() error {
	tenantIDs, err := s.Facade.GetTenantIDs(s.ctx)
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

	if _, ok := s.TenantQueues[tenantID]; ok {
		return ErrDuplicateTenantID
	}

	queue := &ServiceStateQueue{
		CurrentBatch: PendingServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.Facade,
	}
	shutdown := make(chan int)
	s.TenantQueues[tenantID] = queue
	s.TenantShutDowns[tenantID] = shutdown

	go s.tenantLoop(tenantID, shutdown)

	return nil
}

func (s *ServiceStateManager) RemoveTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()
	cancel, ok := s.TenantShutDowns[tenantID]
	if !ok {
		return ErrBadTenantID
	}

	// blocks until the cancel is received
	cancel <- 0
	delete(s.TenantShutDowns, tenantID)
	delete(s.TenantQueues, tenantID)

	return nil
}

// ScheduleServices merges and reconciles a slice of services with the
// ServiceStateChangeBatches in the ServiceStateManager's queue
func (s *ServiceStateManager) ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error {
	s.RLock()
	var queue *ServiceStateQueue
	queue, ok := s.TenantQueues[tenantID]
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
		Services:     svcs,
		DesiredState: desiredState,
		Emergency:    emergency,
	}

	// reconcile the new batch against all batches in queue
	newBatch = queue.reconcileWithBatchQueue(newBatch)

	if len(newBatch.Services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	// reconcile with the pending batch
	newBatch = queue.reconcileWithPendingBatch(newBatch)

	if len(newBatch.Services) == 0 {
		// this is no longer a useful batch
		return nil
	}

	var err error
	// Merge this into the queue
	if newBatch.Emergency {
		err = queue.mergeEmergencyBatch(newBatch)
	} else {
		err = queue.mergeBatch(newBatch)
	}

	// Signal update or exit
	select {
	case queue.Changed <- true:
	default:
	}

	return err
}

func (s *ServiceStateQueue) reconcileWithBatchQueue(new ServiceStateChangeBatch) ServiceStateChangeBatch {
	var newBatchQueue []ServiceStateChangeBatch
	updated := new
	for _, batch := range s.BatchQueue {
		updated = batch.reconcile(updated)
		if len(batch.Services) > 0 {
			newBatchQueue = append(newBatchQueue, batch)
		}
	}

	s.BatchQueue = newBatchQueue

	plog.WithFields(logrus.Fields{
		"newDesiredState": new.DesiredState,
		"newEmergency":    new.Emergency,
	}).Debug("finished reconcile with batch queue")
	return updated
}

func (b *ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	oldSvcs := make([]*service.Service, len(b.Services))
	copy(oldSvcs, b.Services)
	newSvcs := []*service.Service{}
	for _, newSvc := range newBatch.Services {
		addNew := true
		// make a copy of the (possibly) updated oldSvcs and iterate over it
		updatedSvcs := make([]*service.Service, len(oldSvcs))
		copy(updatedSvcs, oldSvcs)
		for i, svc := range updatedSvcs {
			if newSvc.ID == svc.ID {
				if b.Emergency {
					// this service is going to be stopped, so don't bother queuing it
					addNew = false
				} else if newBatch.Emergency {
					// newBatch is going to be brought to the front of the queue on merge,
					// so we can take this service out of the old batch
					oldSvcs = append(oldSvcs[:i], oldSvcs[i+1:]...)
				} else if b.DesiredState != newBatch.DesiredState {
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

	b.Services = oldSvcs

	plog.WithFields(logrus.Fields{
		"existingDesiredState": b.DesiredState,
		"existingEmergency":    b.Emergency,
		"newDesiredState":      newBatch.DesiredState,
		"newEmergency":         newBatch.Emergency,
	}).Debug("finished reconcile")

	return ServiceStateChangeBatch{
		Services:     newSvcs,
		DesiredState: newBatch.DesiredState,
		Emergency:    newBatch.Emergency,
	}
}

func (s *ServiceStateQueue) reconcileWithPendingBatch(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	var newSvcs []*service.Service
	for _, newSvc := range newBatch.Services {
		addNew := true
		for _, pendingSvc := range s.CurrentBatch.Services {
			if newSvc.ID == pendingSvc.ID {
				if newBatch.Emergency {
					// this service is going to be stopped,
					// so cancel the one in currentBatch
					s.cancelPending(pendingSvc.ID)
				} else if s.CurrentBatch.DesiredState != newBatch.DesiredState {
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
		Services:     newSvcs,
		DesiredState: newBatch.DesiredState,
		Emergency:    newBatch.Emergency,
	}
}

func (s *ServiceStateQueue) mergeEmergencyBatch(newBatch ServiceStateChangeBatch) error {
	if !newBatch.Emergency {
		return s.mergeBatch(newBatch)
	}
	// find the last emergency batch in the queue
	lastEmergencyBatch := -1
	for i, batch := range s.BatchQueue {
		if batch.Emergency {
			lastEmergencyBatch = i
		} else {
			break
		}
	}

	// merge newBatch with the emergency batches at the front of the queue
	// Build the list to pass to mergeBatches without using go's append because append may screw up the original slice (no joke)
	emergencyBatches := make([]ServiceStateChangeBatch, lastEmergencyBatch+2)
	copy(emergencyBatches, s.BatchQueue[:lastEmergencyBatch+1])
	emergencyBatches[lastEmergencyBatch+1] = newBatch

	emergencyBatches, err := MergeBatches(emergencyBatches)
	if err != nil {
		return err
	}

	s.BatchQueue = append(emergencyBatches, s.BatchQueue[lastEmergencyBatch+1:]...)

	return nil
}

func (s *ServiceStateQueue) mergeBatch(newBatch ServiceStateChangeBatch) error {
	if newBatch.Emergency {
		return s.mergeEmergencyBatch(newBatch)
	}

	// We want to merge this batch with all consecutive batches at the END of the queue that have the same desired state
	lastBatchToMerge := len(s.BatchQueue)
	for i := len(s.BatchQueue) - 1; i >= 0; i-- {
		if s.BatchQueue[i].DesiredState == newBatch.DesiredState {
			lastBatchToMerge = i
		} else {
			break
		}
	}

	// merge this with any other batches matching desired state at the end of the queue
	// Build the list to pass to mergeBatches without using go's append because append may screw up the original slice (no joke)
	newBatches := make([]ServiceStateChangeBatch, len(s.BatchQueue[lastBatchToMerge:])+1)
	copy(newBatches, s.BatchQueue[lastBatchToMerge:])
	newBatches[len(newBatches)-1] = newBatch
	newBatches, err := MergeBatches(newBatches)
	if err != nil {
		return err
	}
	s.BatchQueue = append(s.BatchQueue[:lastBatchToMerge], newBatches...)
	return nil
}

func MergeBatches(batches []ServiceStateChangeBatch) ([]ServiceStateChangeBatch, error) {
	if len(batches) < 1 {
		return batches, nil
	}

	var fullServiceList []*service.Service

	// Make sure all of the batches we're merging have the same desiredState
	// and emergency status
	desiredState := batches[0].DesiredState
	emergency := batches[0].Emergency
	for _, b := range batches {
		if b.DesiredState != desiredState || b.Emergency != emergency {
			return nil, ErrMismatchedDesiredStates
		}
		fullServiceList = append(fullServiceList, b.Services...)
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
				Services:     newSvcs,
				DesiredState: desiredState,
				Emergency:    emergency,
			})
			newSvcs = []*service.Service{svc}
		}
		previousEmergencyLevel = currentEmergencyLevel
		previousStartLevel = currentStartLevel
	}

	// Add the last batch
	newBatches = append(newBatches, ServiceStateChangeBatch{
		Services:     newSvcs,
		DesiredState: desiredState,
		Emergency:    emergency,
	})

	return newBatches, nil
}

// DrainQueue blocks until the queue is empty.
func (s *ServiceStateManager) DrainQueue(tenantID string) error {
	s.RLock()
	queue, ok := s.TenantQueues[tenantID]
	s.RUnlock()

	if !ok {
		return ErrBadTenantID
	}

	for {
		empty := func() bool {
			queue.RLock()
			defer queue.RUnlock()
			return len(queue.BatchQueue) == 0 && len(queue.CurrentBatch.Services) == 0
		}()

		if empty {
			return nil
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ServiceStateManager) tenantLoop(tenantID string, cancel <-chan int) {
	logger := plog.WithField("tenantid", tenantID)
	s.RLock()
	queue := s.TenantQueues[tenantID]
	s.RUnlock()
	for {
		select {
		case <-cancel:
			return
		default:
		}
		batch, err := queue.getNextBatch()
		if err == nil {
			batchLogger := logger.WithFields(
				logrus.Fields{
					"emergency":    batch.Emergency,
					"desiredstate": batch.DesiredState,
				})
			// Schedule services for this batch
			var services []*service.Service
			for _, svc := range batch.Services {
				services = append(services, svc.Service)
				if batch.Emergency {
					// Set EmergencyShutdown to true for this service and update the database
					svc.EmergencyShutdown = true
					uerr := s.Facade.UpdateService(s.ctx, *svc.Service)
					if uerr != nil {
						batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
					}
				}
			}

			_, serr := s.Facade.ScheduleServiceBatch(s.ctx, services, tenantID, batch.DesiredState)
			if serr != nil {
				batchLogger.WithError(serr).Error("Error scheduling services to stop")
			}

			// Wait on this batch, with cancel option
			desiredState := batch.DesiredState
			if desiredState == service.SVCRestart {
				desiredState = service.SVCRun
			}
			if err := queue.waitServicesWithTimeout(desiredState, batch.Services, s.ServiceRunLevelTimeout); err != nil {
				batchLogger.WithError(err).Error("Error waiting for service batch to reach desired state")
			}
		} else {
			if err != ErrBatchQueueEmpty {
				logger.WithError(err).Error("Error getting next batch")
			}
			select {
			case <-cancel:
				return
			case <-queue.Changed:
			}
		}
	}
}

func (q *ServiceStateQueue) getNextBatch() (b PendingServiceStateChangeBatch, err error) {
	q.Lock()
	defer q.Unlock()
	if len(q.BatchQueue) > 0 {
		b = PendingServiceStateChangeBatch{
			Services:     make(map[string]CancellableService),
			DesiredState: q.BatchQueue[0].DesiredState,
			Emergency:    q.BatchQueue[0].Emergency,
		}

		for _, svc := range q.BatchQueue[0].Services {
			b.Services[svc.ID] = CancellableService{svc, make(chan interface{})}
		}

		q.BatchQueue = q.BatchQueue[1:]
	} else {
		err = ErrBatchQueueEmpty
	}

	q.CurrentBatch = b
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
	if svc, ok := s.CurrentBatch.Services[serviceID]; ok {
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
			err := s.Facade.WaitSingleService(svcArg.Service, dstate, svcArg.Cancel)
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
