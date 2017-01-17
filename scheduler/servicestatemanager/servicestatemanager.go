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
	// ErrBadTenantID occurs when the tenant ID is not contained in the service state manager
	ErrBadTenantID = errors.New("Unrecognized tenant ID")
	// ErrDuplicateTenantID occurs when the tenant ID is already contained in the service state manager
	ErrDuplicateTenantID = errors.New("A tenant with this ID already exists")
	// ErrBatchQueueEmpty occurs when the tenant's queue is empty
	ErrBatchQueueEmpty = errors.New("Tenant service state queue is empty")
	// ErrMismatchedDesiredStates occurs when to batches try to merge with different desired states
	ErrMismatchedDesiredStates = errors.New("Can't merge batches with different desired states")
	// ErrMissingQueue occurs when queue doesn't exist for the tenant with the desired state of a batch
	ErrMissingQueue = errors.New("No queue found for tenant ID and desired state")
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

// ServiceStateQueue is a queue with a tenantLoop processing it's batches
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
	TenantQueues           map[string]map[service.DesiredState]*ServiceStateQueue
	TenantShutDowns        map[string]chan<- int
}

func (p PendingServiceStateChangeBatch) getBatch() ServiceStateChangeBatch {
	var svcs []*service.Service
	for _, svc := range p.Services {
		svcs = append(svcs, svc.Service)
	}
	return ServiceStateChangeBatch{
		Services:     svcs,
		DesiredState: p.DesiredState,
		Emergency:    p.Emergency,
	}
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
		TenantQueues:           make(map[string]map[service.DesiredState]*ServiceStateQueue),
		TenantShutDowns:        make(map[string]chan<- int),
	}
}

// Shutdown properly cancels pending services in tenantLoop
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

// Start gets tenants from the facade and adds them to the service state manager
func (s *ServiceStateManager) Start() error {
	tenantIDs, err := s.Facade.GetTenantIDs(s.ctx)
	if err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		s.AddTenant(tenantID)
		plog.WithFields(logrus.Fields{
			"tenantid": tenantID,
		}).Info("Added tenant to ServiceStateManager")
	}

	plog.WithFields(logrus.Fields{
		"tenantids": tenantIDs,
	}).Info("Started ServiceStateManager")
	return nil
}

// AddTenant adds a queue for a tenant and starts the processing loop for it
func (s *ServiceStateManager) AddTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.TenantQueues[tenantID]; ok {
		return ErrDuplicateTenantID
	}

	shutdown := make(chan int)
	s.TenantQueues[tenantID] = make(map[service.DesiredState]*ServiceStateQueue)
	s.TenantQueues[tenantID][service.SVCRun] = &ServiceStateQueue{
		CurrentBatch: PendingServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.Facade,
	}
	s.TenantQueues[tenantID][service.SVCStop] = &ServiceStateQueue{
		CurrentBatch: PendingServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.Facade,
	}
	s.TenantShutDowns[tenantID] = shutdown
	for _, q := range s.TenantQueues[tenantID] {
		go s.tenantLoop(tenantID, q, shutdown)
	}

	return nil
}

// RemoveTenant cancels the pending batches in queue for the tenant and deletes it from the service state manager
func (s *ServiceStateManager) RemoveTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()
	cancel, ok := s.TenantShutDowns[tenantID]
	if !ok {
		return ErrBadTenantID
	}

	// blocks until the cancel is received
	for range s.TenantQueues[tenantID] {
		cancel <- 0
	}

	delete(s.TenantShutDowns, tenantID)
	delete(s.TenantQueues, tenantID)

	return nil
}

// ScheduleServices merges and reconciles a slice of services with the
// ServiceStateChangeBatches in the ServiceStateManager's queue
func (s *ServiceStateManager) ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error {
	plog.Info("doing servicestatemanager scheduleservices")
	var err error

	plog.Info("getting ssm lock")
	s.RLock()
	plog.Info("got ssm lock")
	var queues map[service.DesiredState]*ServiceStateQueue
	queues, ok := s.TenantQueues[tenantID]
	s.RUnlock()
	plog.Info("unlocking ssm lock")
	if !ok {
		return ErrBadTenantID
	}

	plog.Info("getting queue lock")
	plog.Info("got queue lock")

	// Merge with oldBatch batchQueue
	// 1. If this is emergency, merge with other emergencies and move to front of the queue
	// 2. If any service in this batch is currently in the "pending" batch:
	//    A. If the desired states are the same, leave it pending and remove it from this batch
	//    B. If the desired states are different, cancel the pending request and leave it in this batch
	// 3. If this and the last N batches at the end of the queue all have the same desired state, merge and re-group them
	// 4. If any service in this batch also appears in an earlier batch:
	//    A. If the desired state is the same, leave it in the earlier batch and remove it here
	//    B. If the desired state is different, delete it from the earlier batch and leave it in the new one
	var newBatch ServiceStateChangeBatch
	var expeditedBatch ServiceStateChangeBatch
	var expeditedServices []*service.Service

	for _, queue := range queues {
		func(q *ServiceStateQueue) {
			q.Lock()
			defer q.Unlock()
			// reconcile the new batch against all batches in q
			newBatch, expeditedBatch = q.reconcileWithBatchQueue(ServiceStateChangeBatch{
				Services:     svcs,
				DesiredState: desiredState,
				Emergency:    emergency,
			})
			expeditedServices = append(expeditedServices, expeditedBatch.Services...)

			if len(newBatch.Services) == 0 {
				// this is no longer a useful batch
				return
			}

			// reconcile with the pending batch
			newBatch = q.reconcileWithPendingBatch(newBatch)
		}(queue)
	}

	expeditedBatch.Services = expeditedServices

	queueDesiredState := newBatch.DesiredState
	switch newBatch.DesiredState {
	case service.SVCRestart:
		queueDesiredState = service.SVCRun
	case service.SVCPause:
		queueDesiredState = service.SVCStop
	}

	// Merge this into the queue
	queue, ok := queues[queueDesiredState]
	if !ok {
		return ErrMissingQueue
	}
	queue.Lock()
	defer queue.Unlock()
	if len(expeditedBatch.Services) > 0 {
		s.processBatch(tenantID, expeditedBatch)
	}
	if len(newBatch.Services) == 0 {
		return nil
	}
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

	plog.WithFields(logrus.Fields{
		"services":  svcs,
		"emergency": emergency,
	}).Info("Added services to queue and signaled change")

	return err
}

func (s *ServiceStateQueue) reconcileWithBatchQueue(new ServiceStateChangeBatch) (ServiceStateChangeBatch, ServiceStateChangeBatch) {
	plog.Info("reconciling with batch queue")
	var newBatchQueue []ServiceStateChangeBatch
	var expedited []*service.Service
	expeditedBatch := ServiceStateChangeBatch{
		DesiredState: new.DesiredState,
		Emergency:    new.Emergency,
	}
	updated := new
	for _, batch := range s.BatchQueue {
		updated, expedited = batch.reconcile(updated)
		if len(batch.Services) > 0 {
			newBatchQueue = append(newBatchQueue, batch)
		}
		if len(expedited) > 0 {
			expeditedBatch.Services = append(expeditedBatch.Services, expedited...)
		}
	}

	s.BatchQueue = newBatchQueue

	plog.WithFields(logrus.Fields{
		"newDesiredState": new.DesiredState,
		"newEmergency":    new.Emergency,
	}).Debug("finished reconcile with batch queue")
	return updated, expeditedBatch
}

func (b *ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) (ServiceStateChangeBatch, []*service.Service) {
	plog.Info("reconciling")
	oldSvcs := make([]*service.Service, len(b.Services))
	var expedited []*service.Service
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
					oldSvcs = append(oldSvcs[:i], oldSvcs[i+1:]...)
					expedited = append(expedited, newSvc)
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
	}, expedited
}

func (s *ServiceStateQueue) reconcileWithPendingBatch(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	plog.Info("reconciling with pending")
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
	plog.Info("Merging an emergency batch")
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

	plog.Info("Merging a normal batch")
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

// MergeBatches sorts the services is batches by StartLevel, desiredState, and emergency,
// creates a new batch from the services and returns is
func MergeBatches(batches []ServiceStateChangeBatch) ([]ServiceStateChangeBatch, error) {
	plog.Info("Merging batches")
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

// DrainQueues blocks until the queues are empty for tenantID
func (s *ServiceStateManager) DrainQueues(tenantID string) {
	plog.Info("Draining the queue")
	s.RLock()
	var wg sync.WaitGroup
	for _, queue := range s.TenantQueues[tenantID] {
		wg.Add(1)
		go func(q *ServiceStateQueue) {
			s.drainQueue(q)
			wg.Done()
		}(queue)
	}
	s.RUnlock()
	wg.Wait()
}

func (s *ServiceStateManager) drainQueue(queue *ServiceStateQueue) {
	plog.Info("Draining the queue")

	for {
		empty := func() bool {
			queue.RLock()
			defer queue.RUnlock()
			return len(queue.BatchQueue) == 0 && len(queue.CurrentBatch.Services) == 0
		}()

		if empty {
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ServiceStateManager) tenantLoop(tenantID string, queue *ServiceStateQueue, cancel <-chan int) {
	logger := plog.WithField("tenantid", tenantID)
	for {
		logger.Info("In tenant loop")
		select {
		case <-cancel:
			return
		default:
		}
		batch, err := queue.getNextBatch()
		if err == nil {
			s.processBatch(tenantID, batch.getBatch())
			// Wait on this batch, with cancel option
			desiredState := batch.DesiredState
			if desiredState == service.SVCRestart {
				desiredState = service.SVCRun
			}
			if err := queue.waitServicesWithTimeout(desiredState, batch.Services, s.ServiceRunLevelTimeout); err != nil {
				logger.WithFields(logrus.Fields{
					"emergency":    batch.Emergency,
					"desiredstate": desiredState,
				}).WithError(err).Error("Error waiting for service batch to reach desired state")
			}
		} else {
			if err != ErrBatchQueueEmpty {
				logger.WithError(err).Error("Error getting next batch")
			}
			select {
			case <-cancel:
				return
			case <-queue.Changed:
				logger.Info("Got a job in queue")
			}
		}
	}
}

func (s *ServiceStateManager) processBatch(tenantID string, batch ServiceStateChangeBatch) {
	batchLogger := plog.WithFields(
		logrus.Fields{
			"tenantid":     tenantID,
			"emergency":    batch.Emergency,
			"desiredstate": batch.DesiredState,
		})
	// Schedule services for this batch
	var services []*service.Service
	for _, svc := range batch.Services {
		services = append(services, svc)
		if batch.Emergency {
			// Set EmergencyShutdown to true for this service and update the database
			svc.EmergencyShutdown = true
			uerr := s.Facade.UpdateService(s.ctx, *svc)
			if uerr != nil {
				batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
			}
		}
	}

	_, serr := s.Facade.ScheduleServiceBatch(s.ctx, services, tenantID, batch.DesiredState)
	batchLogger.Info("Scheduled service batch on facade")
	if serr != nil {
		batchLogger.WithError(serr).Error("Error scheduling services")
	}
}

func (s *ServiceStateQueue) getNextBatch() (b PendingServiceStateChangeBatch, err error) {
	s.Lock()
	defer s.Unlock()
	if len(s.BatchQueue) > 0 {
		b = PendingServiceStateChangeBatch{
			Services:     make(map[string]CancellableService),
			DesiredState: s.BatchQueue[0].DesiredState,
			Emergency:    s.BatchQueue[0].Emergency,
		}

		for _, svc := range s.BatchQueue[0].Services {
			b.Services[svc.ID] = CancellableService{svc, make(chan interface{})}
		}

		s.BatchQueue = s.BatchQueue[1:]
	} else {
		err = ErrBatchQueueEmpty
	}

	s.CurrentBatch = b
	plog.Info("Digested batch")
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
			if err := s.Facade.WaitSingleService(svcArg.Service, dstate, svcArg.Cancel); err != nil {
				plog.WithError(err).WithFields(logrus.Fields{
					"serviceid":    svcArg.ID,
					"desiredstate": dstate,
				}).Error("Failed to wait for a single service")
			}
			wg.Done()
		}(svc)
	}
	wg.Wait()
	return nil
}
