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

package scheduler

import (
	"errors"
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"sort"
	"sync"
	"time"
)

type ServiceStateChangeBatch struct {
	services     []*service.Service
	desiredState service.DesiredState
	tenantID     string
	emergency    bool
}

type PendingServiceStateChangeBatch struct {
	services     map[string]CancellableService
	desiredState service.DesiredState
	tenantID     string
	emergency    bool
}

type ServiceStateManager struct {
	sync.RWMutex
	drainQueue             sync.Mutex
	batches                []ServiceStateChangeBatch
	currentBatch           PendingServiceStateChangeBatch
	facade                 Facade
	changed                chan bool
	serviceRunLevelTimeout time.Duration
	ctx                    datastore.Context
}

func (b ServiceStateChangeBatch) Mergeable(batch ServiceStateChangeBatch) bool {
	return b.tenantID == batch.tenantID && b.desiredState == batch.desiredState && b.emergency == batch.emergency
}

func (b PendingServiceStateChangeBatch) Mergeable(batch ServiceStateChangeBatch) bool {
	return b.tenantID == batch.tenantID && b.desiredState == batch.desiredState && b.emergency == batch.emergency
}

func reconcileBatches(old *ServiceStateChangeBatch, new *ServiceStateChangeBatch) {
	newnew := []*service.Service{}
	newold := make([]*service.Service, len(old.services))
	copy(newold, old.services)
	for _, snew := range new.services {
		addnew := true
		// make a copy of newold and iterate over it
		newcopy := make([]*service.Service, len(newold))
		copy(newcopy, newold)
		for i, sold := range newcopy {
			if snew.ID == sold.ID {
				if old.emergency {
					// remove it from new
					addnew = false
					break
				} else if new.emergency {
					// remove it from old
					newold = append(newold[:i], newold[i+1:])
					break
				} else if old.desiredState != new.desiredState {
					// remove it from old
					newold = append(newold[:i], newold[i+1:])
					break
				} else {
					//remove it from new
					addnew = false
					break
				}
			}
		}
		if addnew {
			newnew = append(newnew, snew)
		}
	}

	old.services = newold
	new.services = newnew
}

func mergeBatches(batches []ServiceStateChangeBatch) ([]ServiceStateChangeBatch, error) {
	var newBatches []ServiceStateChangeBatch
	if len(batches) < 1 {
		return newBatches, nil
	}

	var fullServiceList []*service.Service
	desiredState := batches[0].desiredState
	emergency := batches[0].emergency
	for _, b := range batches {
		if b.desiredState != desiredState || b.emergency != emergency {
			// TODO make this a real error
			return nil, errors.New("Can't merge batches with different desired states")
		}
		fullServiceList = append(fullServiceList, b.services)
	}

	// Sort the full list based on desired state
	if desiredState == service.SVCRun {
		// Sort the services by start level
		sort.Sort(service.ByStartLevel{fullServiceList})
	} else if desiredState == service.SVCStop && emergency {
		sort.Sort(service.ByEmergencyShutdown{fullServiceList})
	} else if desiredState == service.SVCStop {
		sort.Sort(service.ByReverseStartLevel{fullServiceList})
	} else if desiredState == service.SVCRestart {
		// TODO: We need to handle this properly
		sort.Sort(service.ByStartLevel{fullServiceList})
	} else if desiredState == service.SVCPause {
		sort.Sort(service.ByReverseStartLevel{fullServiceList})
	}

	// Re-group the services by level
	if len(fullServiceList) > 0 {
		previousEmergencyLevel := fullServiceList[0].EmergencyShutdownLevel
		previousStartLevel := fullServiceList[0].StartLevel
		nextBatch := []*service.Service{}
		for _, svc := range fullServiceList {
			currentEmergencyLevel := svc.EmergencyShutdownLevel
			currentStartLevel := svc.StartLevel
			sameBatch := true
			if emergency {
				sameBatch = currentEmergencyLevel == previousEmergencyLevel
				if sameBatch && currentEmergencyLevel == 0 {
					// For emergency shutdown level 0, we group by reverse start level
					sameBatch = currentStartLevel == previousStartLevel
				}
			} else {
				sameBatch = currentStartLevel == previousStartLevel
			}

			if sameBatch {
				nextBatch = append(nextBatch, svc)
			} else {
				// Add this batch
				newBatches = append(newBatches, nextBatch)
				nextBatch = []*service.Service{svc}
			}
			previousEmergencyLevel = currentEmergencyLevel
			previousStartLevel = currentStartLevel
		}

		// Add the last batch
		newBatches = append(newBatches, nextBatch)
	}
	return newBatches, nil
}

func NewServiceStateManager(ctx datastore.Context, facade Facade, runLevelTimeout time.Duration) *ServiceStateManager {
	return &ServiceStateManager{
		sync.RWMutex{},
		batches:                []ServiceStateChangeBatch{},
		changed:                make(chan bool, 1),
		facade:                 facade,
		serviceRunLevelTimeout: runLevelTimeout,
		ctx: ctx,
	}
}

// Blocks until the queue is empty.  Call LockQueue first
func (s *ServiceStateManager) DrainQueue() {
	for {
		if len(s.batches) == 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func (s *ServiceStateManager) LockQueue() {
	s.drainQueue.Lock()
}

func (s *ServiceStateManager) UnLockQueue() {
	s.drainQueue.Unlock()
}

func (s *ServiceStateManager) ScheduleServices(svcs []*service.Service, desiredState service.DesiredState, emergency bool, tenantID string) error {
	s.Lock()
	defer s.Unlock()

	s.drainQueue.Lock()
	defer s.drainQueue.Unlock()

	// Merge with existing batches
	// 1. If this is emergency, merge with other emergencies and move to front of the queue
	// 2. If any service in this batch is currently in the "pending" batch:
	//    A. If the desired states are the same, leave it pending and remove it from this batch
	//    B. If the desired states are different, cancel the pending request and leave it in this batch
	// 3. If this and the last N batches at the end of the queue all have the same desired state, merge and re-group them
	// 4. If any service in this batch also appears in an earlier batch:
	//    A. If the desired state is the same, leave it in the earlier batch and remove it here
	//    B. If the desired state is different, delete it from the earlier batch and leave it in the new one

	newBatch := ServiceStateChangeBatch{svcs, tenantID, desiredState, emergency}
	for _, batch := range s.batches {
		if batch.tenantID == tenantID {
			reconcileBatches(&batch, &newBatch)
		}
	}

	// reconcile with the pending batch
	if s.currentBatch.tenantID == newBatch.tenantID {
		var newnew []*service.Service
		for _, newsvc := range newBatch.services {
			keepnew := true
			for _, pendingSvc := range s.currentBatch.services {
				if newsvc.ID == pendingSvc.ID {
					if newsvc.DesiredState != pendingSvc.DesiredState {
						s.cancelPending(newsvc.ID)
					} else if newBatch.emergency {
						s.cancelPending(newsvc.ID)
					} else {
						// Remove it from newBatch
						keepnew = false
					}
					break
				}
				if keepnew {
					newnew = append(newnew, newsvc)
				}
			}
		}
		newBatch.services = newnew
	}

	// Merge this into the queue
	if emergency {
		// merge this with any other emergency batches at the front of the queue
		lastBatchToMerge := -1
		for i, queuebatch := range s.batches {
			if queuebatch.emergency && queuebatch.tenantID == tenantID {
				lastBatchToMerge = i
			} else {
				break
			}
		}

		newBatches, err := mergeBatches(append(s.batches[:lastBatchToMerge+1], newBatch))
		if err != nil {
			return err
		}
		s.batches = append(newBatches, s.batches[lastBatchToMerge+1:])
	} else {
		// merge this with any other batches matching desired state at the end of the queue
		lastBatchToMerge := len(s.batches)
		for i := len(s.batches) - 1; i >= 0; i-- {
			if s.batches[i].desiredState == desiredState && s.batches[i].tenantID == tenantID {
				lastBatchToMerge = i
			} else {
				break
			}
		}

		newBatches, err := mergeBatches(append(s.batches[lastBatchToMerge:], newBatch))
		if err != nil {
			return err
		}
		s.batches = append(s.batches[:lastBatchToMerge], newBatches)
	}

	// Signal update or exit
	select {
	case s.changed <- true:
	default:
	}

	return nil
}

func (s *ServiceStateManager) getNextBatch() PendingServiceStateChangeBatch {
	s.Lock()
	defer s.Unlock()
	if len(s.batches) > 0 {
		b := PendingServiceStateChangeBatch{
			services:     make(map[string]CancellableService),
			desiredState: s.batches[0].desiredState,
			emergency:    s.batches[0].emergency,
			tenantID:     s.batches[0].tenantID,
		}

		for _, svc := range s.batches[0].services {
			b.services[svc.ID] = CancellableService{svc, make(chan interface{})}
		}

		s.batches = s.batches[1:]
		s.currentBatch = b
		return b
	}

	return nil
}

func (s *ServiceStateManager) mainloop(cancel <-chan interface{}) {
	for {
		batch := s.getNextBatch()
		if batch != nil {
			batchLogger := plog.WithFields(
				logrus.Fields{
					"emergency":    batch.emergency,
					"desiredstate": batch.desiredState,
					"tenantid":     batch.tenantID,
				})
			// Schedule services for this batch
			var services []*service.Service
			for _, svc := range batch.services {
				services = append(services, svc)
				if batch.emergency {
					// Set EmergencyShutdown to true for this service and update the database
					svc.EmergencyShutdown = true
					uerr := s.facade.UpdateService(s.ctx, *svc, false, false)
					if uerr != nil {
						batchLogger.WithField("service", svc.ID).WithError(uerr).Error("Failed to update database with EmergencyShutdown")
					}
				}
			}

			_, serr := s.facade.ScheduleServiceBatch(s.ctx, services)
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

func (s *ServiceStateManager) waitServicesWithTimeout(dstate service.DesiredState, services []CancellableService, timeout time.Duration) error {
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
	case <-timer:
		// close all channels
		s.Lock()
		for _, svc := range services {
			s.cancelPending(svc.ID)
		}
		s.Unlock()
		return errors.New("Timeout waiting for services")
	}

	return nil
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

type CancellableService struct {
	*service.Service
	Cancel chan interface{}
}

func (s *ServiceStateManager) waitServicesWithCancel(dstate service.DesiredState, services []CancellableService) error {
	var wg sync.WaitGroup
	for _, svc := range services {
		wg.Add(1)
		go func(CancellableService) {
			err := s.facade.WaitSingleService(svc.Service, dstate, svc.Cancel)
			plog.WithError(err).WithFields(logrus.Fields{
				"serviceid":    svc.ID,
				"desiredstate": dstate,
			}).Error("Failed to wait for a single service")
			wg.Done()
		}(svc)
	}
	wg.Wait()
	return nil
}
