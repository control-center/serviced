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

type ServiceStateManager interface {
	// ScheduleServices schedules a set of services to change their desired state
	ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error
	// AddTenant prepares the service state manager to receive requests for a new tenant
	AddTenant(tenantID string) error
	// RemoveTenant notifies the service state manager that a tenant no longer exists
	RemoveTenant(tenantID string) error
	// Wait blocks until all current processing for the given tenant is complete
	Wait(tenantID string)
	// WaitScheduled blocks until all requested services have been scheduled or cancelled
	WaitScheduled(tenantID string, serviceIDs ...string)
}

// ServiceStateChangeBatch represents a batch of services with the same
// desired state that will be operated on by a ServiceStateManager
type ServiceStateChangeBatch struct {
	Services     map[string]CancellableService
	DesiredState service.DesiredState
	Emergency    bool
}

// CancellableService is a service whose scheduling may be canceled by a channel
type CancellableService struct {
	*service.Service
	cancel chan interface{}
	C      <-chan interface{}
}

// Types for sorting
type CancellableServices []CancellableService

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

// ServiceStateQueue is a queue with a tenantLoop processing it's batches
type ServiceStateQueue struct {
	sync.RWMutex
	BatchQueue   []ServiceStateChangeBatch
	CurrentBatch ServiceStateChangeBatch
	Changed      chan bool
	Facade       Facade
}

// BatchServiceStateManager intelligently schedules batches of services to start/stop/restart with the facade
type BatchServiceStateManager struct {
	sync.RWMutex
	Facade                 Facade
	ctx                    datastore.Context
	ServiceRunLevelTimeout time.Duration
	TenantQueues           map[string]map[service.DesiredState]*ServiceStateQueue
	TenantShutDowns        map[string]chan<- int
}

func NewCancellableService(svc *service.Service) CancellableService {
	cancel := make(chan interface{})
	return CancellableService{
		svc,
		cancel,
		cancel,
	}
}

func (s CancellableService) Cancel() {
	// Make sure it isn't already closed
	select {
	case <-s.cancel:
		return
	default:
		close(s.cancel)
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
func NewBatchServiceStateManager(facade Facade, ctx datastore.Context, runLevelTimeout time.Duration) *BatchServiceStateManager {
	return &BatchServiceStateManager{
		RWMutex: sync.RWMutex{},
		Facade:  facade,
		ctx:     ctx,
		ServiceRunLevelTimeout: runLevelTimeout,
		TenantQueues:           make(map[string]map[service.DesiredState]*ServiceStateQueue),
		TenantShutDowns:        make(map[string]chan<- int),
	}
}

// Shutdown properly cancels pending services in tenantLoop
func (s *BatchServiceStateManager) Shutdown() {
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
func (s *BatchServiceStateManager) Start() error {
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
func (s *BatchServiceStateManager) AddTenant(tenantID string) error {
	s.Lock()
	defer s.Unlock()

	if _, ok := s.TenantQueues[tenantID]; ok {
		return ErrDuplicateTenantID
	}

	shutdown := make(chan int)
	s.TenantQueues[tenantID] = make(map[service.DesiredState]*ServiceStateQueue)
	s.TenantQueues[tenantID][service.SVCRun] = &ServiceStateQueue{
		CurrentBatch: ServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.Facade,
	}
	s.TenantQueues[tenantID][service.SVCStop] = &ServiceStateQueue{
		CurrentBatch: ServiceStateChangeBatch{},
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
func (s *BatchServiceStateManager) RemoveTenant(tenantID string) error {
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
func (s *BatchServiceStateManager) ScheduleServices(svcs []*service.Service, tenantID string, desiredState service.DesiredState, emergency bool) error {
	var err error

	s.RLock()
	var queues map[service.DesiredState]*ServiceStateQueue
	queues, ok := s.TenantQueues[tenantID]
	s.RUnlock()
	if !ok {
		return ErrBadTenantID
	}

	// Build the cancellable services from the list
	cancellableServices := make(map[string]CancellableService)
	for _, svc := range svcs {
		cancellableServices[svc.ID] = NewCancellableService(svc)
	}

	// Merge with oldBatch batchQueue
	// 1. If this is emergency, merge with other emergencies and move to front of the queue
	// 2. If any service in this batch is currently in the "pending" batch:
	//    A. If the desired states are the same, leave it pending and remove it from this batch
	//    B. If the desired states are different, cancel the pending request and leave it in this batch
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
	expeditedServices := make(map[string]CancellableService)

	for _, queue := range queues {
		func(q *ServiceStateQueue) {
			q.Lock()
			defer q.Unlock()
			// reconcile the new batch against all batches in q
			newBatch, expeditedBatch = q.reconcileWithBatchQueue(newBatch)
			for id, svc := range expeditedBatch.Services {
				expeditedServices[id] = svc
			}

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
		// process the expedited batch now
		go func() {
			s.processBatch(tenantID, expeditedBatch)
			// Cancel these services to notify waiters that they have been scheduled
			for _, svc := range expeditedBatch.Services {
				svc.Cancel()
			}
		}()
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

	return err
}

func (s *ServiceStateQueue) reconcileWithBatchQueue(new ServiceStateChangeBatch) (ServiceStateChangeBatch, ServiceStateChangeBatch) {
	var newBatchQueue []ServiceStateChangeBatch
	expeditedBatch := ServiceStateChangeBatch{
		Services:     make(map[string]CancellableService),
		DesiredState: new.DesiredState,
		Emergency:    new.Emergency,
	}
	updated := new
	for _, batch := range s.BatchQueue {
		var expedited map[string]CancellableService
		updated, expedited = batch.reconcile(updated)
		if len(batch.Services) > 0 {
			newBatchQueue = append(newBatchQueue, batch)
		}
		for key, value := range expedited {
			expeditedBatch.Services[key] = value
		}
	}

	s.BatchQueue = newBatchQueue

	plog.WithFields(logrus.Fields{
		"newDesiredState": new.DesiredState,
		"newEmergency":    new.Emergency,
	}).Debug("finished reconcile with batch queue")
	return updated, expeditedBatch
}

func (b ServiceStateChangeBatch) reconcile(newBatch ServiceStateChangeBatch) (ServiceStateChangeBatch, map[string]CancellableService) {
	expedited := make(map[string]CancellableService)
	newSvcs := make(map[string]CancellableService)
	for id, newSvc := range newBatch.Services {
		if oldsvc, ok := b.Services[id]; ok {
			// There is already an entry in the queue for this service, so reconcile
			if b.Emergency {
				// this service is going to be stopped, so don't bother queuing it
				// nobody should be watching it yet, but go ahead and cancel it anyway
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
				delete(b.Services, id)
				oldsvc.Cancel()
				newSvcs[id] = newSvc
			} else {
				// this service exists in b with the same desired state,
				// so expedite it
				delete(b.Services, id)
				expedited[id] = oldsvc
				// nobody should be watching the new service yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			}
		} else {
			// no overlap, keep the new service
			newSvcs[id] = newSvc
		}
	}

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
	reconciledBatch := ServiceStateChangeBatch{
		Services:     make(map[string]CancellableService),
		DesiredState: newBatch.DesiredState,
		Emergency:    newBatch.Emergency,
	}

	for id, newSvc := range newBatch.Services {
		if oldsvc, ok := s.CurrentBatch.Services[id]; ok {
			// There is already an entry in the queue for this service, so reconcile
			if s.CurrentBatch.Emergency {
				// this service is going to be stopped, so don't bother queuing it
				// nobody should be watching it yet, but go ahead and cancel it anyway
				newSvc.Cancel()
			} else if newBatch.Emergency {
				// newBatch is going to be brought to the front of the queue on merge,
				// so we can take this service out of the old batch
				oldsvc.Cancel()
				reconciledBatch.Services[id] = newSvc
			} else if s.CurrentBatch.DesiredState != newBatch.DesiredState {
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
		"existingDesiredState": s.CurrentBatch.DesiredState,
		"existingEmergency":    s.CurrentBatch.Emergency,
		"newDesiredState":      newBatch.DesiredState,
		"newEmergency":         newBatch.Emergency,
	}).Debug("finished reconcile")

	return reconciledBatch
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

	var fullServiceList []CancellableService

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
	newSvcs := make(map[string]CancellableService)

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
			newSvcs[svc.ID] = svc
		} else {
			// append our batch from the previous level, start a new batch with svc
			newBatches = append(newBatches, ServiceStateChangeBatch{
				Services:     newSvcs,
				DesiredState: desiredState,
				Emergency:    emergency,
			})
			newSvcs = make(map[string]CancellableService)
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

// WaitScheduled blocks until every service has been scheduled or moved/removed from the queue
func (s *BatchServiceStateManager) WaitScheduled(tenantID string, serviceIDs ...string) {
	s.RLock()
	var wg sync.WaitGroup
	for _, sid := range serviceIDs {
		// find the service in the queues
		if svc, ok := s.findService(tenantID, sid); ok {
			wg.Add(1)
			go func(s CancellableService) {
				<-s.C
				wg.Done()
			}(svc)
		}
	}
	s.RUnlock()
	wg.Wait()
}

func (s *BatchServiceStateManager) findService(tenantID, serviceID string) (CancellableService, bool) {
	s.RLock()
	defer s.RUnlock()
	for _, queue := range s.TenantQueues[tenantID] {
		svc, ok := func() (CancellableService, bool) {
			queue.RLock()
			defer queue.RUnlock()
			for _, batch := range queue.BatchQueue {
				if svc, ok := batch.Services[serviceID]; ok {
					return svc, true
				}
			}
			return CancellableService{}, false
		}()

		if ok {
			return svc, ok
		}
	}
	return CancellableService{}, false
}

func (s *BatchServiceStateManager) drainQueue(queue *ServiceStateQueue) {
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

func (s *BatchServiceStateManager) tenantLoop(tenantID string, queue *ServiceStateQueue, cancel <-chan int) {
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
			sids := s.processBatch(tenantID, batch)

			// Cancel the services that didn't get scheduled, so we don't wait on them
			for _, sid := range sids {
				batch.Services[sid].Cancel()
			}

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
			}
		}
	}
}

func (s *BatchServiceStateManager) processBatch(tenantID string, batch ServiceStateChangeBatch) []string {
	batchLogger := plog.WithFields(
		logrus.Fields{
			"tenantid":     tenantID,
			"emergency":    batch.Emergency,
			"desiredstate": batch.DesiredState,
		})
	// Schedule services for this batch
	var services []*service.Service
	var serviceIDs []string
	for _, svc := range batch.Services {
		services = append(services, svc.Service)
		serviceIDs = append(serviceIDs, svc.ID)
		if batch.Emergency {
			// Set EmergencyShutdown to true for this service and update the database
			svc.EmergencyShutdown = true
			uerr := s.Facade.UpdateService(s.ctx, *svc.Service)
			if uerr != nil {
				batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
			}
		}
	}

	failedServiceIDs, serr := s.Facade.ScheduleServiceBatch(s.ctx, services, tenantID, batch.DesiredState)
	if serr != nil {
		batchLogger.WithError(serr).Error("Error scheduling services")
		return serviceIDs
	}

	return failedServiceIDs
}

func (s *ServiceStateQueue) getNextBatch() (b ServiceStateChangeBatch, err error) {
	s.Lock()
	defer s.Unlock()
	if len(s.BatchQueue) > 0 {
		b = s.BatchQueue[0]
		s.BatchQueue = s.BatchQueue[1:]
	} else {
		err = ErrBatchQueueEmpty
	}

	s.CurrentBatch = b
	return
}

func (s *ServiceStateQueue) waitServicesWithTimeout(dstate service.DesiredState, services map[string]CancellableService, timeout time.Duration) error {
	done := make(chan error)
	cancel := make(chan interface{})
	defer func() {
		// close all channels
		close(cancel)
		s.RLock()
		for _, svc := range services {
			svc.Cancel()
		}
		s.RUnlock()
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
		return errors.New("Timeout waiting for services")
		// defer will cancel all waits
	}
}

func (s *ServiceStateQueue) waitServicesWithCancel(dstate service.DesiredState, services map[string]CancellableService) error {
	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(svcArg CancellableService) {
			if err := s.Facade.WaitSingleService(svcArg.Service, dstate, svcArg.C); err != nil {
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
