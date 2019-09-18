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
	// ErrWaitTimeout occurs when we timeout waiting for a batch of services to reach the desired state
	ErrWaitTimeout = errors.New("Timeout waiting for services")
	// ErrNotRunning occurs when you try to perform an operation on a service state manager that is not running
	ErrNotRunning = errors.New("Service state manager is not running")
	// ErrAlreadyStarted occurs when you try to start a service state manager that has already been started
	ErrAlreadyStarted = errors.New("Service state manager has already been started")
)

// ServiceStateManager provides a way to organize and manage services before scheduling them
type ServiceStateManager interface {
	// ScheduleServices schedules a set of services to change their desired state
	ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error
	// AddTenant prepares the service state manager to receive requests for a new tenant
	AddTenant(tenantID string) error
	// RemoveTenant notifies the service state manager that a tenant no longer exists
	RemoveTenant(tenantID string) error
	// WaitScheduled blocks until all requested services have been scheduled or cancelled
	WaitScheduled(tenantID string, serviceIDs ...string)
	// Wait blocks until all processing for the current tenant has completed
	Wait(tenantID string)
	// SyncCurrentStates will try to determine the current state of services and update the service store
	SyncCurrentStates(svcIDs []string)
}

// ServiceStateChangeBatch represents a batch of services with the same
// desired state that will be operated on by a ServiceStateManager
type ServiceStateChangeBatch struct {
	Services     map[string]*CancellableService
	DesiredState service.DesiredState
	Emergency    bool
}

// CancellableService is a service whose scheduling may be canceled by a channel
type CancellableService struct {
	*service.Service
	cancel     chan interface{}
	C          <-chan interface{}
	cancelLock *sync.Mutex
}

// Types for sorting
type CancellableServices []*CancellableService

func (s CancellableServices) Len() int      { return len(s) }
func (s CancellableServices) Swap(i, j int) { s[i], s[j] = s[j], s[i] }

type ByEmergencyShutdown struct{ CancellableServices }

// Sort by EmergencyShutdownLevel - 1, to ensure level of 0 is last
func (s ByEmergencyShutdown) Less(i, j int) bool {
	if s.CancellableServices[i].EmergencyShutdownLevel == s.CancellableServices[j].EmergencyShutdownLevel {
		// If emergency shutdown level is the same, order by reverse start level
		return s.CancellableServices[i].StartLevel-1 > s.CancellableServices[j].StartLevel-1
	} else {
		return s.CancellableServices[i].EmergencyShutdownLevel-1 < s.CancellableServices[j].EmergencyShutdownLevel-1
	}
}

type ByStartLevel struct{ CancellableServices }

func (s ByStartLevel) Less(i, j int) bool {
	return s.CancellableServices[i].StartLevel-1 < s.CancellableServices[j].StartLevel-1
}

type ByReverseStartLevel struct{ CancellableServices }

func (s ByReverseStartLevel) Less(i, j int) bool {
	return s.CancellableServices[i].StartLevel-1 > s.CancellableServices[j].StartLevel-1
}

// ServiceStateQueue is a queue with a queueLoop processing it's batches
type ServiceStateQueue struct {
	lock         *sync.RWMutex
	BatchQueue   []ServiceStateChangeBatch
	CurrentBatch ServiceStateChangeBatch
	Changed      chan bool
	Facade       Facade
}

func NewServiceStateQueue(facade Facade) *ServiceStateQueue {
	return &ServiceStateQueue{
		lock:         &sync.RWMutex{},
		BatchQueue:   []ServiceStateChangeBatch{},
		CurrentBatch: ServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       facade,
	}
}

// BatchServiceStateManager schedules batches of services to start/stop/restart according to
//  StartLevel
type BatchServiceStateManager struct {
	lock                   *sync.RWMutex
	Facade                 Facade
	ctx                    datastore.Context
	ServiceRunLevelTimeout time.Duration
	TenantQueues           map[string]map[service.DesiredState]*ServiceStateQueue
	TenantShutDowns        map[string]chan<- int
	currentStateLock       *sync.Mutex
	currentStateWaits      map[string]*CurrentStateWait
	shutdown               chan struct{}
}

type CurrentStateWait struct {
	cancelLock   *sync.Mutex
	cancel       chan interface{}
	WaitingState service.DesiredState
	Done         <-chan struct{}
}

func (w *CurrentStateWait) Cancel() {
	w.cancelLock.Lock()
	defer w.cancelLock.Unlock()
	// Make sure it isn't already closed
	select {
	case <-w.cancel:
	default:
		close(w.cancel)
	}
	<-w.Done
}

func NewCancellableService(svc *service.Service) *CancellableService {
	cancel := make(chan interface{})
	return &CancellableService{
		Service:    svc,
		cancel:     cancel,
		C:          cancel,
		cancelLock: &sync.Mutex{},
	}
}

func (s *CancellableService) Cancel() {
	s.cancelLock.Lock()
	defer s.cancelLock.Unlock()
	// Make sure it isn't already closed
	select {
	case <-s.cancel:
		return
	default:
		close(s.cancel)
	}
}

func (b ServiceStateChangeBatch) Cancel() {
	for _, svc := range b.Services {
		svc.Cancel()
	}
}

func (q *ServiceStateQueue) Cancel() {
	q.lock.RLock()
	defer q.lock.RUnlock()
	for _, b := range q.BatchQueue {
		b.Cancel()
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

// NewBatchServiceStateManager creates a new, initialized ServiceStateManager
func NewBatchServiceStateManager(facade Facade, ctx datastore.Context, runLevelTimeout time.Duration) *BatchServiceStateManager {
	return &BatchServiceStateManager{
		lock:             &sync.RWMutex{},
		currentStateLock: &sync.Mutex{},
		Facade:           facade,
		ctx:              ctx,
		ServiceRunLevelTimeout: runLevelTimeout,
		TenantQueues:           make(map[string]map[service.DesiredState]*ServiceStateQueue),
		TenantShutDowns:        make(map[string]chan<- int),
		currentStateWaits:      make(map[string]*CurrentStateWait),
	}
}

// Shutdown properly cancels pending services in queueLoop
func (s *BatchServiceStateManager) Shutdown() {
	s.lock.Lock()
	defer s.lock.Unlock()
	close(s.shutdown)
	for tenantID, _ := range s.TenantShutDowns {
		s.removeTenant(tenantID)
	}

	s.currentStateLock.Lock()
	defer s.currentStateLock.Unlock()
	for _, thread := range s.currentStateWaits {
		thread.Cancel()
	}

	// Clear the current state wait list
	s.currentStateWaits = make(map[string]*CurrentStateWait)
}

// Start gets tenants from the facade and adds them to the service state manager
func (s *BatchServiceStateManager) Start() error {

	s.lock.Lock()
	if s.shutdown != nil {
		select {
		case <-s.shutdown:
		default:
			s.lock.Unlock()
			return ErrAlreadyStarted
		}
	}

	s.shutdown = make(chan struct{})
	s.lock.Unlock()

	tenantIDs, err := s.Facade.GetTenantIDs(s.ctx)
	if err != nil {
		return err
	}

	for _, tenantID := range tenantIDs {
		s.AddTenant(tenantID)
	}

	plog.WithFields(logrus.Fields{
		"tenantids": tenantIDs,
	}).Info("Started ServiceStateManager")
	return nil
}

// AddTenant adds a queue for a tenant and starts the processing loop for it
func (s *BatchServiceStateManager) AddTenant(tenantID string) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	// We can't start the queue loop if we have been shutdown, so don't allow adding a tenant
	if s.shutdown == nil {
		return ErrNotRunning
	}

	select {
	case <-s.shutdown:
		return ErrNotRunning
	default:
	}

	if _, ok := s.TenantQueues[tenantID]; ok {
		return ErrDuplicateTenantID
	}

	shutdown := make(chan int)
	s.TenantQueues[tenantID] = make(map[service.DesiredState]*ServiceStateQueue)
	s.TenantQueues[tenantID][service.SVCRun] = NewServiceStateQueue(s.Facade)
	s.TenantQueues[tenantID][service.SVCStop] = NewServiceStateQueue(s.Facade)
	s.TenantShutDowns[tenantID] = shutdown
	for t, q := range s.TenantQueues[tenantID] {
		go s.queueLoop(tenantID, t.String(), q, shutdown)
	}

	plog.WithField("tenantid", tenantID).Debug("Added tenant to service state manager")

	return nil
}

// RemoveTenant cancels the pending batches in queue for the tenant and deletes it from the service state manager
func (s *BatchServiceStateManager) RemoveTenant(tenantID string) error {
	defer plog.WithField("tenantid", tenantID).Debug("Tenant removed from service state manager")
	s.lock.Lock()
	defer s.lock.Unlock()
	return s.removeTenant(tenantID)
}

func (s *BatchServiceStateManager) removeTenant(tenantID string) error {
	cancel, ok := s.TenantShutDowns[tenantID]
	if !ok {
		return ErrBadTenantID
	}

	// blocks until the cancel is received
	for _, q := range s.TenantQueues[tenantID] {
		cancel <- 0
		q.Cancel()
	}

	delete(s.TenantShutDowns, tenantID)
	delete(s.TenantQueues, tenantID)

	return nil
}

func (s *BatchServiceStateManager) SyncCurrentStates(svcIDs []string) {
	states := make(map[string]service.ServiceCurrentState)
	s.lock.RLock()
	defer s.lock.RUnlock()
	// Get states for everything in our queues
	for _, tenantQueues := range s.TenantQueues {
		for _, queue := range tenantQueues {
			func() {
				queue.lock.RLock()
				defer queue.lock.RUnlock()

				// Pending states for services in the queue
				for _, batch := range queue.BatchQueue {
					pendingState := service.DesiredToCurrentPendingState(batch.DesiredState, batch.Emergency)
					for _, svc := range batch.Services {
						states[svc.ID] = pendingState
					}
				}

				// Transition states for services in the current batch
				transitionState := service.DesiredToCurrentTransitionState(queue.CurrentBatch.DesiredState, queue.CurrentBatch.Emergency)
				for _, svc := range queue.CurrentBatch.Services {
					states[svc.ID] = transitionState
				}
			}()
		}
	}

	// Update the states we know and build a list of missing states
	s.currentStateLock.Lock()
	defer s.currentStateLock.Unlock()

	var missingStates []string
	for _, sid := range svcIDs {
		if state, ok := states[sid]; ok {
			s.updateServiceCurrentState(state, sid)
		} else {
			missingStates = append(missingStates, sid)
		}
	}

	// Now figure out the ones that aren't in the queues
	// Get updated service info
	svcs := s.Facade.GetServicesForScheduling(s.ctx, missingStates)
	for _, svc := range svcs {
		s.startCurrentStateWait(svc, service.DesiredState(svc.DesiredState), svc.EmergencyShutdown)
	}
}

// Lock s.currentStateLock before calling this
func (s *BatchServiceStateManager) startCurrentStateWait(svc *service.Service, desiredState service.DesiredState, emergency bool) {
	if thread, ok := s.currentStateWaits[svc.ID]; ok {
		if thread.WaitingState == desiredState {
			// We are already waiting on it, do nothing
			return
		} else {
			// Cancel this wait and start a new one
			thread.Cancel()
			delete(s.currentStateWaits, svc.ID)
		}
	}

	finalState := service.DesiredToCurrentFinalState(desiredState, emergency)
	currentState := service.DesiredToCurrentTransitionState(desiredState, emergency)

	// Set current state to the transition state
	s.updateServiceCurrentState(currentState, svc.ID)

	// Spawn a thread that will update the state to the final state when it reaches it
	cancel := make(chan interface{})
	done := make(chan struct{})
	thread := &CurrentStateWait{
		cancelLock:   &sync.Mutex{},
		cancel:       cancel,
		Done:         done,
		WaitingState: desiredState,
	}
	s.currentStateWaits[svc.ID] = thread
	go func() {
		defer close(done)
		s.Facade.WaitSingleService(svc, desiredState, cancel)

		// If we were cancelled, bail
		select {
		case <-cancel:
			return
		default:
		}

		s.Facade.SetServicesCurrentState(s.ctx, finalState, svc.ID)
	}()
}

// Lock s.currentStateLock before calling this
func (s *BatchServiceStateManager) updateServiceCurrentState(state service.ServiceCurrentState, serviceIDs ...string) {
	// Cancel any existing waits
	for _, sid := range serviceIDs {
		if thread, ok := s.currentStateWaits[sid]; ok {
			thread.Cancel()
			delete(s.currentStateWaits, sid)
		}
	}
	s.Facade.SetServicesCurrentState(s.ctx, state, serviceIDs...)

}

// ScheduleServices merges and reconciles a slice of services with the
// ServiceStateChangeBatches in the ServiceStateManager's queue
func (s *BatchServiceStateManager) ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error {
	logger := plog.WithFields(logrus.Fields{
		"servicecount": len(svcs),
		"tenantid":     tenantID,
		"desiredstate": desiredState.String(),
		"emergency":    emergency,
	})

	var err error
	logger.Debug("Scheduling services")
	s.lock.Lock()
	var queues map[service.DesiredState]*ServiceStateQueue
	queues, ok := s.TenantQueues[tenantID]

	if !ok {
		s.lock.Unlock()
		return ErrBadTenantID
	}

	// Get a lock on all queues
	for _, q := range queues {
		q.lock.Lock()
		defer q.lock.Unlock()
	}
	s.lock.Unlock()
	// Build the cancellable services from the list
	cancellableServices := make(map[string]*CancellableService)
	for _, svc := range svcs {
		cancellableServices[svc.ID] = NewCancellableService(svc)
	}

	// Merge with oldBatch batchQueue
	// 1. If this is emergency, merge with other emergencies and move to front of the queue
	// 2. If any service in this batch is currently in the "pending" batch:
	//    A. If the desired states are the same, leave it pending and remove it from this batch
	//    B. If the desired states are different, cancel waits on the pending request and leave it in this batch
	// 3. If this and the last N batches at the end of the queue all have the same desired state, merge and re-group them
	// 4. If any service in this batch also appears in an earlier batch:
	//    A. If the desired state is the same, leave it in the earlier batch and remove it here
	//    B. If the desired state is different, delete it from the earlier batch and leave it in the new one
	newBatch := ServiceStateChangeBatch{
		Services:     cancellableServices,
		DesiredState: desiredState,
		Emergency:    emergency,
	}
	var expeditedBatch ServiceStateChangeBatch
	expeditedServices := make(map[string]*CancellableService)
	var cancelledIDs []string
	for _, queue := range queues {
		// reconcile the new batch against all batches in q
		var cancelled []string
		newBatch, expeditedBatch, cancelled = queue.reconcileWithBatchQueue(newBatch)
		cancelledIDs = append(cancelledIDs, cancelled...)
		for id, svc := range expeditedBatch.Services {
			expeditedServices[id] = svc
		}

		if len(newBatch.Services) == 0 {
			// Nothing left to reconcile
			break
		}

		// reconcile with the pending batch
		newBatch = queue.reconcileWithPendingBatch(newBatch)
	}

	// Now check for services that are already in the desired state
	var cancelled []string
	newBatch, cancelled = s.reconcileWithCurrentState(newBatch)
	cancelledIDs = append(cancelledIDs, cancelled...)

	if len(cancelledIDs) > 0 {
		// Spawn a goroutine to update the current state of all cancelled services
		defer func() {
			logger.WithField("services", cancelledIDs).Debug("Cancelling and re-syncing services")
			go s.SyncCurrentStates(cancelledIDs)
		}()
	}

	expeditedBatch.Services = expeditedServices

	if len(expeditedBatch.Services) > 0 {
		// process the expedited batch now
		go func() {
			s.updateBatch(&expeditedBatch)
			failed := s.processBatch(tenantID, expeditedBatch)
			// Cancel these services to notify waiters that they have been scheduled
			for _, svc := range expeditedBatch.Services {
				svc.Cancel()
			}
			if len(failed) > 0 {
				s.SyncCurrentStates(failed)
			}
		}()
	}

	if len(newBatch.Services) == 0 {
		return nil
	}

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

	if newBatch.Emergency {
		err = queue.mergeEmergencyBatch(newBatch)
	} else {
		err = queue.mergeBatch(newBatch)
	}

	// Set pending desired state
	currentState := service.DesiredToCurrentPendingState(newBatch.DesiredState, newBatch.Emergency)
	var sids []string
	for sid := range newBatch.Services {
		sids = append(sids, sid)
	}
	s.currentStateLock.Lock()
	s.updateServiceCurrentState(currentState, sids...)
	s.currentStateLock.Unlock()

	// Signal update or exit
	select {
	case queue.Changed <- true:
	default:
	}

	return err
}

// reconcileWithCurrentState will compare the desired state to the services' current states and return a new batch
//  with unnecessary services removed.  It also returns a list of IDs of the services it omitted
func (s *BatchServiceStateManager) reconcileWithCurrentState(new ServiceStateChangeBatch) (ServiceStateChangeBatch, []string) {
	newSvcs := make(map[string]*CancellableService)
	var cancelledIDs []string
	for id, newSvc := range new.Services {
		if service.DesiredStateIsRedundant(new.DesiredState, new.Emergency, service.ServiceCurrentState(newSvc.CurrentState)) {
			newSvc.Cancel()
			cancelledIDs = append(cancelledIDs, id)
		} else {
			newSvcs[id] = newSvc
		}
	}
	return ServiceStateChangeBatch{
		Services:     newSvcs,
		DesiredState: new.DesiredState,
		Emergency:    new.Emergency,
	}, cancelledIDs
}

func (s *ServiceStateQueue) reconcileWithBatchQueue(new ServiceStateChangeBatch) (ServiceStateChangeBatch, ServiceStateChangeBatch, []string) {
	var newBatchQueue []ServiceStateChangeBatch
	var cancelledIDs []string
	expeditedBatch := ServiceStateChangeBatch{
		Services:     make(map[string]*CancellableService),
		DesiredState: new.DesiredState,
		Emergency:    new.Emergency,
	}
	updated := new
	for _, batch := range s.BatchQueue {
		var expedited map[string]*CancellableService
		var cancelled []string
		updated, expedited, cancelled = batch.reconcile(updated)
		cancelledIDs = append(cancelledIDs, cancelled...)
		if len(batch.Services) > 0 {
			newBatchQueue = append(newBatchQueue, batch)
		}
		for key, value := range expedited {
			expeditedBatch.Services[key] = value
		}
	}

	s.BatchQueue = newBatchQueue

	plog.WithFields(logrus.Fields{
		"newdesiredState": new.DesiredState,
		"newemergency":    new.Emergency,
	}).Debug("finished reconcile with batch queue")
	return updated, expeditedBatch, cancelledIDs
}

func (b ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) (batch ServiceStateChangeBatch, expedited map[string]*CancellableService, cancelled []string) {
	expedited = make(map[string]*CancellableService)
	newSvcs := make(map[string]*CancellableService)
	for id, newSvc := range newBatch.Services {
		logger := plog.WithFields(logrus.Fields{
			"id":   id,
			"name": newSvc.Name,
		})

		if oldsvc, ok := b.Services[id]; ok {
			// There is already an entry in the queue for this service, so reconcile
			if b.Emergency && !newBatch.Emergency {
				// this service is going to be stopped, so don't bother queuing it
				// nobody should be watching it yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			} else if b.Emergency && b.DesiredState == newBatch.DesiredState {
				// Duplicate, and it is an emergency so it should already be at the front of the queue
				// so discard the new one.
				// nobody should be watching it yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			} else if b.Emergency && b.DesiredState == service.SVCStop {
				// Two emergencies with different desired states, stop takes priority
				newSvc.Cancel()
			} else if newBatch.Emergency {
				// newBatch is going to be brought to the front of the queue on merge,
				// so we can take this service out of the old batch
				delete(b.Services, id)
				oldsvc.Cancel()
				newSvcs[id] = newSvc
			} else if b.DesiredState != newBatch.DesiredState {
				// this service has a newer desired state than it does in b,
				// so we can take this service out of old batch
				logger.Debugf("State mismatch, existing: %s, newBatch: %s", b.DesiredState, newBatch.DesiredState)
				delete(b.Services, id)
				oldsvc.Cancel()

				// Is this a "cancel"?
				if service.DesiredCancelsPending(service.DesiredToCurrentPendingState(b.DesiredState, b.Emergency), newBatch.DesiredState) {
					plog.WithFields(logrus.Fields{
						"service":  newSvc.ID,
						"oldstate": b.DesiredState,
						"newstate": newBatch.DesiredState,
					}).Debug("Cancelling pending service")
					cancelled = append(cancelled, newSvc.ID)
				} else {
					newSvcs[id] = newSvc
				}
			} else {
				logger.Debug("State matches, expediting")
				// this service exists in b with the same desired state,
				// so expedite it
				delete(b.Services, id)
				expedited[id] = oldsvc
				// nobody should be watching the new service yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			}
		} else {
			logger.Debug("New service, adding to batch")
			// no overlap, keep the new service
			newSvcs[id] = newSvc
		}
	}

	plog.WithFields(logrus.Fields{
		"existingdesiredState": b.DesiredState,
		"existingemergency":    b.Emergency,
		"newdesiredstate":      newBatch.DesiredState,
		"newemergency":         newBatch.Emergency,
		"newservices":          len(newSvcs),
	}).Debug("finished reconcile")

	batch = ServiceStateChangeBatch{
		Services:     newSvcs,
		DesiredState: newBatch.DesiredState,
		Emergency:    newBatch.Emergency,
	}

	return
}

func (s *ServiceStateQueue) reconcileWithPendingBatch(newBatch ServiceStateChangeBatch) ServiceStateChangeBatch {
	reconciledBatch := ServiceStateChangeBatch{
		Services:     make(map[string]*CancellableService),
		DesiredState: newBatch.DesiredState,
		Emergency:    newBatch.Emergency,
	}
	b := s.CurrentBatch
	for id, newSvc := range newBatch.Services {
		if oldsvc, ok := s.CurrentBatch.Services[id]; ok {
			// There is already an entry in the queue for this service, so reconcile
			if b.Emergency && !newBatch.Emergency {
				// this service is going to be stopped, so don't bother queuing it
				// nobody should be watching it yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			} else if b.Emergency && b.DesiredState == newBatch.DesiredState {
				// Duplicate, and it is an emergency so it should already be at the front of the queue
				// so discard the new one.
				// nobody should be watching it yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			} else if b.Emergency && b.DesiredState == service.SVCStop {
				// Two emergencies with different desired states, stop takes priority
				newSvc.Cancel()
			} else if newBatch.Emergency {
				// newBatch is going to be brought to the front of the queue on merge,
				// so we can take this service out of the old batch
				oldsvc.Cancel()
				reconciledBatch.Services[id] = newSvc
			} else if b.DesiredState != newBatch.DesiredState {
				// this service has a newer desired state than it does in b,
				// so we can take this service out of old batch
				oldsvc.Cancel()
				reconciledBatch.Services[id] = newSvc
			} else {
				// this service exists in current batch with the same desired state,
				// so ignore it
				// nobody should be watching the new service yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			}
		} else {
			// no overlap, keep the new service
			reconciledBatch.Services[id] = newSvc
		}
	}

	plog.WithFields(logrus.Fields{
		"existingdesiredstate": s.CurrentBatch.DesiredState,
		"existingemergency":    s.CurrentBatch.Emergency,
		"newdesiredstate":      newBatch.DesiredState,
		"newemergency":         newBatch.Emergency,
		"newservices":          len(reconciledBatch.Services),
	}).Debug("finished reconcile with pending batch")

	return reconciledBatch
}

func (s *ServiceStateQueue) mergeEmergencyBatch(newBatch ServiceStateChangeBatch) error {
	if !newBatch.Emergency {
		return s.mergeBatch(newBatch)
	}
	// find the last emergency batch in the queue
	lastEmergencyBatch := -1
	for i, batch := range s.BatchQueue {
		if batch.Emergency && batch.DesiredState == newBatch.DesiredState {
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
		if s.BatchQueue[i].DesiredState == newBatch.DesiredState && s.BatchQueue[i].Emergency == newBatch.Emergency {
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
	if len(batches) < 1 {
		return batches, nil
	}

	var fullServiceList []*CancellableService

	// Make sure all of the batches we're merging have the same desiredState
	// and emergency status
	desiredState := batches[0].DesiredState
	emergency := batches[0].Emergency
	for _, b := range batches {
		if b.DesiredState != desiredState || b.Emergency != emergency {
			return nil, ErrMismatchedDesiredStates
		}
		for _, svc := range b.Services {
			fullServiceList = append(fullServiceList, svc)
		}
	}

	// Sort the full list based on desired state
	if desiredState == service.SVCRun {
		// Sort the services by start level
		sort.Sort(ByStartLevel{
			CancellableServices: fullServiceList,
		})
	} else if desiredState == service.SVCStop && emergency {
		sort.Sort(ByEmergencyShutdown{
			CancellableServices: fullServiceList,
		})
	} else if desiredState == service.SVCStop {
		sort.Sort(ByReverseStartLevel{
			CancellableServices: fullServiceList,
		})
	} else if desiredState == service.SVCRestart {
		sort.Sort(ByStartLevel{
			CancellableServices: fullServiceList,
		})
	} else if desiredState == service.SVCPause {
		sort.Sort(ByReverseStartLevel{
			CancellableServices: fullServiceList,
		})
	}

	// regroup the services by level
	previousEmergencyLevel := fullServiceList[0].EmergencyShutdownLevel
	previousStartLevel := fullServiceList[0].StartLevel

	newBatches := []ServiceStateChangeBatch{}
	newSvcs := make(map[string]*CancellableService)

	for _, svc := range fullServiceList {
		currentEmergencyLevel := svc.EmergencyShutdownLevel
		currentStartLevel := svc.StartLevel
		var sameBatch bool

		if emergency && desiredState == service.SVCStop {
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
			newSvcs[svc.ID] = svc
		} else {
			// append our batch from the previous level, start a new batch with svc
			newBatches = append(newBatches, ServiceStateChangeBatch{
				Services:     newSvcs,
				DesiredState: desiredState,
				Emergency:    emergency,
			})
			newSvcs = make(map[string]*CancellableService)
			newSvcs[svc.ID] = svc
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

// Wait blocks until the queues are empty for tenantID
func (s *BatchServiceStateManager) Wait(tenantID string) {
	s.lock.RLock()
	var wg sync.WaitGroup
	for _, queue := range s.TenantQueues[tenantID] {
		wg.Add(1)
		go func(q *ServiceStateQueue) {
			s.drainQueue(q)
			wg.Done()
		}(queue)
	}
	s.lock.RUnlock()
	wg.Wait()
}

// WaitScheduled blocks until every service has been scheduled or moved/removed from the queue
func (s *BatchServiceStateManager) WaitScheduled(tenantID string, serviceIDs ...string) {
	s.lock.RLock()
	var wg sync.WaitGroup
	for _, sid := range serviceIDs {
		// find the service in the queues
		if svc, ok := s.findService(tenantID, sid); ok {
			wg.Add(1)
			go func(cs *CancellableService) {
				<-cs.C
				wg.Done()
			}(svc)
		} else {
			plog.WithFields(logrus.Fields{
				"tenantid":  tenantID,
				"serviceid": sid,
			}).Debug("Not waiting for service, could not find it")
		}
	}
	s.lock.RUnlock()
	wg.Wait()
}

func (s *BatchServiceStateManager) findService(tenantID, serviceID string) (*CancellableService, bool) {
	for _, queue := range s.TenantQueues[tenantID] {
		svc, ok := func() (*CancellableService, bool) {
			queue.lock.RLock()
			defer queue.lock.RUnlock()
			for _, batch := range queue.BatchQueue {
				if svc, ok := batch.Services[serviceID]; ok {
					return svc, true
				}
			}
			if svc, ok := queue.CurrentBatch.Services[serviceID]; ok {
				return svc, true
			}
			return nil, false
		}()

		if ok {
			return svc, ok
		}
	}
	return nil, false
}

func (s *BatchServiceStateManager) drainQueue(queue *ServiceStateQueue) {
	for {
		empty := func() bool {
			queue.lock.RLock()
			defer queue.lock.RUnlock()
			return len(queue.BatchQueue) == 0 && len(queue.CurrentBatch.Services) == 0
		}()

		if empty {
			return
		}

		time.Sleep(100 * time.Millisecond)
	}
}

func (s *BatchServiceStateManager) queueLoop(tenantID, queueName string, queue *ServiceStateQueue, cancel <-chan int) {
	logger := plog.WithFields(logrus.Fields{
		"tenantid": tenantID,
		"queue":    queueName,
	})

	logger.Info("Started loop for queue")
	defer logger.Info("queue loop exited")
	var badsids []string
	for {
		select {
		case <-cancel:
			return
		default:
		}

		batch, err := queue.getNextBatch()

		// We re-sync the failed services from the previous run after getNextBatch has updated the queue
		if len(badsids) > 0 {
			s.SyncCurrentStates(badsids)
		}

		if err == nil {
			batchlogger := logger.WithFields(logrus.Fields{
				"emergency":    batch.Emergency,
				"desiredstate": batch.DesiredState,
				"timeout":      s.ServiceRunLevelTimeout,
				"batchsize":    len(batch.Services),
			})
			batchlogger.Debug("Got Batch")

			s.updateBatch(&batch)
			badsids = s.processBatch(tenantID, batch)

			// Cancel the services that didn't get scheduled, so we don't wait on them
			for _, sid := range badsids {
				batch.Services[sid].Cancel()
			}

			// Wait on this batch, with cancel option
			desiredState := batch.DesiredState
			if desiredState == service.SVCRestart {
				desiredState = service.SVCRun
			}
			err := queue.waitServicesWithTimeout(desiredState, batch.Services, s.ServiceRunLevelTimeout)
			if err == ErrWaitTimeout {
				batchlogger.Warn("Timeout waiting for service batch to reach desired state")
			} else if err != nil {
				batchlogger.WithError(err).Error("Error waiting for service batch to reach desired state")
			}
		} else {
			if err != ErrBatchQueueEmpty {
				logger.WithError(err).Error("Error getting next batch")
			}
			logger.Debug("Waiting for change")
			select {
			case <-cancel:
				return
			case <-queue.Changed:
			}
		}
	}
}

func (s *BatchServiceStateManager) updateBatch(batch *ServiceStateChangeBatch) {
	serviceIDs := make([]string, len(batch.Services))
	i := 0
	for _, svc := range batch.Services {
		serviceIDs[i] = svc.ID
		i++
	}

	services := s.Facade.GetServicesForScheduling(s.ctx, serviceIDs)

	newServices := make(map[string]*CancellableService, len(services))

	for id, batchService := range batch.Services {
		found := false
		for _, svc := range services {
			if id == svc.ID {
				found = true
				batchService.Service = svc
				newServices[svc.ID] = batchService
				break
			}
		}
		if !found {
			batchService.Cancel()
		}
	}

	batch.Services = newServices
}

func (s *BatchServiceStateManager) processBatch(tenantID string, batch ServiceStateChangeBatch) []string {
	batchLogger := plog.WithFields(
		logrus.Fields{
			"tenantid":     tenantID,
			"emergency":    batch.Emergency,
			"desiredstate": batch.DesiredState,
		})
	// Schedule services for this batch
	var services []*CancellableService
	var serviceIDs []string
	for _, svc := range batch.Services {
		select {
		case <-svc.C:

		default:
			services = append(services, svc)
			serviceIDs = append(serviceIDs, svc.ID)
			if batch.Emergency && batch.DesiredState == service.SVCStop {
				// Set EmergencyShutdown to true for this service and update the database
				svc.EmergencyShutdown = true
				uerr := s.Facade.UpdateService(s.ctx, *svc.Service)
				if uerr != nil {
					batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
				}
			}
		}
	}

	// Update pending state to transition state (-ing state)
	s.currentStateLock.Lock()
	s.updateServiceCurrentState(service.DesiredToCurrentTransitionState(batch.DesiredState, batch.Emergency), serviceIDs...)
	s.currentStateLock.Unlock()

	failedServiceIDs, serr := s.Facade.ScheduleServiceBatch(s.ctx, services, tenantID, batch.DesiredState)
	if serr != nil {
		batchLogger.WithError(serr).Error("Error scheduling services")
		return serviceIDs
	}

	// Start a watch to update to final state
	s.currentStateLock.Lock()
	for _, svc := range batch.Services {
		// Check if it was failed
		failed := false
		for _, fid := range failedServiceIDs {
			if fid == svc.ID {
				failed = true
				break
			}
		}

		if !failed {
			s.startCurrentStateWait(svc.Service, batch.DesiredState, batch.Emergency)
		}
	}
	s.currentStateLock.Unlock()
	return failedServiceIDs
}

func (s *ServiceStateQueue) getNextBatch() (b ServiceStateChangeBatch, err error) {
	s.lock.Lock()
	defer s.lock.Unlock()
	if len(s.BatchQueue) > 0 {
		b = s.BatchQueue[0]
		s.BatchQueue = s.BatchQueue[1:]
	} else {
		err = ErrBatchQueueEmpty
	}

	// Make a copy of this batch to store in current batch so we don't have to hold the lock while processing it
	serviceCopy := make(map[string]*CancellableService, len(b.Services))
	for k, v := range b.Services {
		serviceCopy[k] = v
	}

	s.CurrentBatch = ServiceStateChangeBatch{
		DesiredState: b.DesiredState,
		Emergency:    b.Emergency,
		Services:     serviceCopy,
	}

	return
}

func (s *ServiceStateQueue) waitServicesWithTimeout(dstate service.DesiredState, services map[string]*CancellableService, timeout time.Duration) error {
	done := make(chan error)
	cancel := make(chan interface{})
	logger := plog.WithFields(logrus.Fields{
		"numservices": len(services),
	})

	logger.Debug("Starting waitServicesWithTimeout")

	defer func() {
		// close all channels
		close(cancel)
		for _, svc := range services {
			svc.Cancel()
		}
	}()

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
		return ErrWaitTimeout
		// defer will cancel all waits
	}
}

func (s *ServiceStateQueue) waitServicesWithCancel(dstate service.DesiredState, services map[string]*CancellableService) error {
	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(svcArg *CancellableService) {
			if err := s.Facade.WaitSingleService(svcArg.Service, dstate, svcArg.C); err != nil {
				plog.WithError(err).WithFields(logrus.Fields{
					"serviceid":    svcArg.ID,
					"desiredstate": dstate,
				}).Error("Failed to wait for service")
			}
			wg.Done()
		}(svc)
	}
	wg.Wait()
	return nil
}
