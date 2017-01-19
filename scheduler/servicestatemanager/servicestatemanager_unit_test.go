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

// +build unit

package servicestatemanager_test

import (
	"fmt"
	"testing"
	"time"

	datastoremocks "github.com/control-center/serviced/datastore/mocks"
	"github.com/control-center/serviced/domain/service"
	ssm "github.com/control-center/serviced/scheduler/servicestatemanager"
	"github.com/control-center/serviced/scheduler/servicestatemanager/mocks"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

func TestServiceStateManager(t *testing.T) { TestingT(t) }

type ServiceStateManagerSuite struct {
	serviceStateManager *ssm.BatchServiceStateManager
	facade              *mocks.Facade
	ctx                 *datastoremocks.Context
}

var _ = Suite(&ServiceStateManagerSuite{})

func (s *ServiceStateManagerSuite) SetUpTest(c *C) {
	s.facade = &mocks.Facade{}
	s.ctx = &datastoremocks.Context{}
	s.serviceStateManager = ssm.NewBatchServiceStateManager(s.facade, s.ctx, 10*time.Second)
}

func getTestServicesABC() []*service.Service {
	return []*service.Service{
		&service.Service{
			ID:                     "A",
			DesiredState:           1,
			EmergencyShutdownLevel: 0,
			StartLevel:             2,
		},
		&service.Service{
			ID:                     "B",
			DesiredState:           1,
			EmergencyShutdownLevel: 1,
			StartLevel:             3,
		},
		&service.Service{
			ID:                     "C",
			DesiredState:           1,
			EmergencyShutdownLevel: 2,
			StartLevel:             2,
		},
	}
}

func getTestServicesDEF() []*service.Service {
	return []*service.Service{
		&service.Service{
			ID:                     "D",
			DesiredState:           1,
			EmergencyShutdownLevel: 0,
			StartLevel:             2,
		},
		&service.Service{
			ID:                     "E",
			DesiredState:           1,
			EmergencyShutdownLevel: 1,
			StartLevel:             3,
		},
		&service.Service{
			ID:                     "F",
			DesiredState:           1,
			EmergencyShutdownLevel: 2,
			StartLevel:             2,
		},
	}
}

func getTestServicesADGH() []*service.Service {
	return []*service.Service{
		&service.Service{
			ID:                     "A",
			DesiredState:           1,
			EmergencyShutdownLevel: 0,
			StartLevel:             2,
		},
		&service.Service{
			ID:                     "D",
			DesiredState:           1,
			EmergencyShutdownLevel: 0,
			StartLevel:             2,
		},
		&service.Service{
			ID:                     "G",
			DesiredState:           1,
			EmergencyShutdownLevel: 1,
			StartLevel:             3,
		},
		&service.Service{
			ID:                     "H",
			DesiredState:           1,
			EmergencyShutdownLevel: 2,
			StartLevel:             2,
		},
	}
}
func getTestServicesI() []*service.Service {
	return []*service.Service{
		&service.Service{
			ID:                     "I",
			DesiredState:           1,
			EmergencyShutdownLevel: 0,
			StartLevel:             0,
		},
	}
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_ScheduleServices_NoErr(c *C) {

	// Test that the batch has been added to the BatchQueue
	// and split by nomral start level
	tenantID := "tenant"
	s.serviceStateManager.TenantQueues[tenantID] = make(map[service.DesiredState]*ssm.ServiceStateQueue)
	s.serviceStateManager.TenantQueues[tenantID][service.SVCRun] = &ssm.ServiceStateQueue{
		BatchQueue:   make([]ssm.ServiceStateChangeBatch, 0),
		CurrentBatch: ssm.ServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.facade,
	}
	s.serviceStateManager.TenantQueues[tenantID][service.SVCStop] = &ssm.ServiceStateQueue{
		BatchQueue:   make([]ssm.ServiceStateChangeBatch, 0),
		CurrentBatch: ssm.ServiceStateChangeBatch{},
		Changed:      make(chan bool),
		Facade:       s.facade,
	}

	startQueue := s.serviceStateManager.TenantQueues[tenantID][service.SVCRun]
	stopQueue := s.serviceStateManager.TenantQueues[tenantID][service.SVCStop]

	// Test that:
	// 1. The batch has been added to the startQueue
	// 2. The batch has been split into batches by startlevel
	// 3. Nothing was falsely added to the stopQueue
	err := s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}
	pass := s.CompareBatchSlices(c, startQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)

	c.Assert(len(stopQueue.BatchQueue), Equals, 0)

	// Test that:
	// 1. The batch has been added to the stopQueue
	// 2. The batch was split by Emergency shutdown level
	// 3. The Emergency batches were moved to the front of the queue,
	// 4. The existing batches have been purged of startQueue
	err = s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCStop, true)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	c.Assert(len(startQueue.BatchQueue), Equals, 0)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
	})

	c.Assert(pass, Equals, true)

	// Test that trying to start a batch that has been scheduled for Emergency shutdown has no effect on the queue
	err = s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	c.Assert(len(startQueue.BatchQueue), Equals, 0)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
	})
	c.Assert(pass, Equals, true)

	// Test that adding a non-conflicting non-Emergency batch gets split by start level and appended to the queue
	err = s.serviceStateManager.ScheduleServices(getTestServicesDEF(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}
	pass = s.CompareBatchSlices(c, startQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"D": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "D",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"F": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "F",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"E": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "E",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
	})
	c.Assert(pass, Equals, true)

	// Add a non-Emergency batch with some expedited and some non-conflicting services, and make sure that:
	//  1. The expedited services are removed from the incoming batch and processed on their own (mocked)
	//  2. The non-conflicting services are merged with the end of the queue based on start level
	s.facade.On("ScheduleServiceBatch", s.ctx, mock.AnythingOfType("[]*service.Service"), tenantID, service.SVCRun).Return([]string{}, nil).Once()
	err = s.serviceStateManager.ScheduleServices(getTestServicesADGH(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	pass = s.CompareBatchSlices(c, startQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"F": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "F",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
				"H": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "H",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"E": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "E",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
				"G": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "G",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
	})
	c.Assert(pass, Equals, true)

	// Stop services DEF and make sure it cancels the start requests
	err = s.serviceStateManager.ScheduleServices(getTestServicesDEF(), tenantID, service.SVCStop, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	pass = s.CompareBatchSlices(c, startQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"H": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "H",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"G": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "G",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"E": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "E",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"D": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "D",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"F": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "F",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
	// Add an Emergency shutdown request with EmergencyShutDownLevel 0 and StartLevel 0 and make sure it gets placed
	//  before other EmergencyShutDownLevel 0
	err = s.serviceStateManager.ScheduleServices(getTestServicesI(), tenantID, service.SVCStop, true)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	pass = s.CompareBatchSlices(c, startQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"H": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "H",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"G": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "G",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
	pass = s.CompareBatchSlices(c, stopQueue.BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"I": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "I",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             0,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"E": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "E",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"D": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "D",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"F": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "F",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_ScheduleServices_ReconcileWithPending(c *C) {
	// Set up a pending batch
	tenantID := "tenant"
	s.serviceStateManager.TenantQueues[tenantID] = make(map[service.DesiredState]*ssm.ServiceStateQueue)
	pendingServices := make(map[string]ssm.CancellableService)
	for _, s := range getTestServicesABC() {
		s.StartLevel = 0
		pendingServices[s.ID] = ssm.NewCancellableService(s)
	}

	queue := &ssm.ServiceStateQueue{
		CurrentBatch: ssm.ServiceStateChangeBatch{
			Services:     pendingServices,
			DesiredState: 0,
			Emergency:    false,
		},
	}
	s.serviceStateManager.TenantQueues[tenantID][service.SVCStop] = queue

	// Add a batch that gets cancelled by the pending batch
	err := s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCStop, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	// Make sure the pending services do NOT get cancelled
	for _, pending := range pendingServices {
		select {
		case <-pending.C:
			c.Fatal("Pending Service cancelled unexpectedly")
		default:
		}
	}

	// Our queue should still be empty
	c.Assert(len(s.serviceStateManager.TenantQueues[tenantID][service.SVCStop].BatchQueue), Equals, 0)

	// Add an Emergency batch that cancels a pending batch
	err = s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCStop, true)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	// Make sure the pending services do NOT get cancelled
	for _, pending := range pendingServices {
		select {
		case <-pending.C:
		default:
			c.Fatal("Pending Service NOT cancelled")
		}
	}

	// Our queue should be populated
	pass := s.CompareBatchSlices(c, s.serviceStateManager.TenantQueues[tenantID][service.SVCStop].BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 0,
			Emergency:    true,
		},
	})
	c.Assert(pass, Equals, true)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_ScheduleServices_NonEmergencyCancelPending(c *C) {
	// Set up a pending batch
	tenantID := "tenant"
	s.serviceStateManager.TenantQueues[tenantID] = make(map[service.DesiredState]*ssm.ServiceStateQueue)
	pendingServices := make(map[string]ssm.CancellableService)
	for _, s := range getTestServicesABC() {
		s.StartLevel = 0
		pendingServices[s.ID] = ssm.NewCancellableService(s)
	}

	queue := &ssm.ServiceStateQueue{
		CurrentBatch: ssm.ServiceStateChangeBatch{
			Services:     pendingServices,
			DesiredState: 0,
			Emergency:    false,
		},
	}
	s.serviceStateManager.TenantQueues[tenantID][service.SVCRun] = &ssm.ServiceStateQueue{}
	s.serviceStateManager.TenantQueues[tenantID][service.SVCStop] = queue

	// Add a non-Emergency batch that cancels a pending batch
	err := s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	// Make sure the pending services do NOT get cancelled
	for _, pending := range pendingServices {
		select {
		case <-pending.C:
		default:
			c.Fatal("Pending Service NOT cancelled")
		}
	}

	for _, batch := range s.serviceStateManager.TenantQueues[tenantID][service.SVCStop].BatchQueue {
		s.LogBatch(c, batch)
	}

	// Our queue should be populated
	pass := s.CompareBatchSlices(c, s.serviceStateManager.TenantQueues[tenantID][service.SVCStop].BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_ScheduleServices_TwoTenants(c *C) {
	// set up 2 tenants
	tenantID := "tenant"
	s.serviceStateManager.TenantQueues[tenantID] = make(map[service.DesiredState]*ssm.ServiceStateQueue)
	tenantID2 := "tenant2"
	s.serviceStateManager.TenantQueues[tenantID2] = make(map[service.DesiredState]*ssm.ServiceStateQueue)

	queue := &ssm.ServiceStateQueue{
		CurrentBatch: ssm.ServiceStateChangeBatch{},
	}

	queue2 := &ssm.ServiceStateQueue{
		CurrentBatch: ssm.ServiceStateChangeBatch{},
	}

	s.serviceStateManager.TenantQueues[tenantID][service.SVCRun] = queue
	s.serviceStateManager.TenantQueues[tenantID2][service.SVCRun] = queue2

	err := s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	err = s.serviceStateManager.ScheduleServices(getTestServicesDEF(), tenantID2, service.SVCRun, false)
	if err != nil {
		c.Fatalf("ssm.Error in TestScheduleServices: %v\n", err)
	}

	// Check that the queues are correct:
	pass := s.CompareBatchSlices(c, s.serviceStateManager.TenantQueues[tenantID][service.SVCRun].BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"A": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "A",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"C": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "C",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"B": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "B",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})

	c.Assert(pass, Equals, true)

	pass = s.CompareBatchSlices(c, s.serviceStateManager.TenantQueues[tenantID2][service.SVCRun].BatchQueue, []ssm.ServiceStateChangeBatch{
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"D": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "D",
						DesiredState:           1,
						EmergencyShutdownLevel: 0,
						StartLevel:             2,
					},
				},
				"F": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "F",
						DesiredState:           1,
						EmergencyShutdownLevel: 2,
						StartLevel:             2,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
		ssm.ServiceStateChangeBatch{
			Services: map[string]ssm.CancellableService{
				"E": ssm.CancellableService{
					Service: &service.Service{
						ID:                     "E",
						DesiredState:           1,
						EmergencyShutdownLevel: 1,
						StartLevel:             3,
					},
				},
			},
			DesiredState: 1,
			Emergency:    false,
		},
	})
	c.Assert(pass, Equals, true)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_ScheduleServices_BadTenant(c *C) {
	tenantID := "tenant"
	// No tenants exist
	err := s.serviceStateManager.ScheduleServices(getTestServicesABC(), tenantID, service.SVCRun, false)
	c.Assert(err, Equals, ssm.ErrBadTenantID)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_MergeBatches_UnmatchedStates(c *C) {
	batches := []ssm.ServiceStateChangeBatch{ssm.ServiceStateChangeBatch{DesiredState: 0}, ssm.ServiceStateChangeBatch{DesiredState: 1}}

	// No tenants exist
	_, err := ssm.MergeBatches(batches)
	c.Assert(err, Equals, ssm.ErrMismatchedDesiredStates)
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_AddAndRemoveTenants(c *C) {
	// Add a tenant
	err := s.serviceStateManager.AddTenant("tenant")
	c.Assert(err, IsNil)

	// Confirm that the loop was started by sending to the Changed channel
	Changed := s.serviceStateManager.TenantQueues["tenant"][service.SVCRun].Changed
	timer := time.NewTimer(1 * time.Second)
	select {
	case Changed <- true:
	case <-timer.C:
		c.Fatalf("Tenant loop not running")
	}

	// Try to re-add the same tenant
	err = s.serviceStateManager.AddTenant("tenant")
	c.Assert(err, Equals, ssm.ErrDuplicateTenantID)

	// Remove the tenant
	err = s.serviceStateManager.RemoveTenant("tenant")
	c.Assert(err, IsNil)

	// Confirm that the loop was stopped by trying to send to the Changed channel
	timer.Reset(100 * time.Millisecond)
	select {
	case Changed <- true:
		c.Fatalf("Tenant loop not terminated")
	case <-timer.C:
	}

	// Confirm the tenant was removed from the queues and cancel channels
	_, ok := s.serviceStateManager.TenantQueues["tenant"]
	c.Assert(ok, Equals, false)
	_, ok = s.serviceStateManager.TenantShutDowns["tenant"]
	c.Assert(ok, Equals, false)

	// Try to remove the tenant again, make sure it fails
	err = s.serviceStateManager.RemoveTenant("tenant")
	c.Assert(err, Equals, ssm.ErrBadTenantID)

}

func (s *ServiceStateManagerSuite) TestServiceStateManager_StartShutdown(c *C) {
	// set up some tenants
	s.facade.On("GetTenantIDs", s.ctx).Return([]string{"tenant1", "tenant2"}, nil)

	// Start the manager
	s.serviceStateManager.Start()

	// Make sure both tenants were added
	queue1, ok := s.serviceStateManager.TenantQueues["tenant1"][service.SVCRun]
	c.Assert(ok, Equals, true)

	_, ok = s.serviceStateManager.TenantShutDowns["tenant1"]
	c.Assert(ok, Equals, true)

	queue2, ok := s.serviceStateManager.TenantQueues["tenant2"][service.SVCRun]
	c.Assert(ok, Equals, true)

	_, ok = s.serviceStateManager.TenantShutDowns["tenant2"]
	c.Assert(ok, Equals, true)

	// Make sure the loops were started
	changed1 := queue1.Changed
	timer := time.NewTimer(1 * time.Second)
	select {
	case changed1 <- true:
	case <-timer.C:
		c.Fatalf("Tenant 1 loop not running")
	}

	changed2 := queue2.Changed
	timer.Reset(1 * time.Second)
	select {
	case changed2 <- true:
	case <-timer.C:
		c.Fatalf("Tenant 2 loop not running")
	}

	// Stop the manager
	s.serviceStateManager.Shutdown()

	// Make sure both loops were stopped
	timer.Reset(100 * time.Millisecond)
	select {
	case changed1 <- true:
		c.Fatalf("Tenant loop 1 not terminated")
	case <-timer.C:
	}

	timer.Reset(100 * time.Millisecond)
	select {
	case changed2 <- true:
		c.Fatalf("Tenant loop 1 not terminated")
	case <-timer.C:
	}
}

func (s *ServiceStateManagerSuite) TestServiceStateManager_tenantLoop(c *C) {
	// Setup a tenant
	s.facade.On("GetTenantIDs", s.ctx).Return([]string{"tenant1"}, nil).Once()

	svcs := getTestServicesADGH()

	svcA := svcs[0]
	svcD := svcs[1]
	svcG := svcs[2]
	svcH := svcs[3]

	// Start the manager
	s.serviceStateManager.Start()

	// The first batch should contain A, D, H because of startlevel
	// Those should get waited on by a call to the facade from runLoop
	s.facade.On("ScheduleServiceBatch", s.ctx, mock.AnythingOfType("[]*service.Service"), "tenant1", service.SVCRun).Return([]string{}, nil).Once()
	s.facade.On("WaitSingleService", svcA, service.SVCRun, mock.AnythingOfType("<-chan interface {}")).
		Return(nil).Run(func(mock.Arguments) { c.Logf("Waited on A") }).Once()
	s.facade.On("WaitSingleService", svcD, service.SVCRun, mock.AnythingOfType("<-chan interface {}")).
		Return(nil).Run(func(mock.Arguments) { c.Logf("Waited on D") }).Once()
	s.facade.On("WaitSingleService", svcH, service.SVCRun, mock.AnythingOfType("<-chan interface {}")).
		Return(nil).Run(func(mock.Arguments) { c.Logf("Waited on H") }).Once()

	// We'll sleep a bit to make sure those services reach desired state in zk (mocked),
	// then it should grab another batch off of the queue (which will just contain G at this point) and it should get processed
	s.facade.On("ScheduleServiceBatch", s.ctx, []*service.Service{svcG}, "tenant1", service.SVCRun).Return([]string{}, nil).Once()
	s.facade.On("WaitSingleService", svcG, service.SVCRun, mock.AnythingOfType("<-chan interface {}")).
		Return(nil).Run(func(mock.Arguments) { c.Logf("Waited on G") }).Once()

	err := s.serviceStateManager.ScheduleServices(svcs, "tenant1", service.SVCRun, false)
	c.Assert(err, IsNil)

	// Sleep so our stuff goes through the loop process and we can guarantee our calls
	time.Sleep(time.Millisecond * 300)
	s.facade.AssertExpectations(c)

	// Stop the manager
	s.serviceStateManager.Shutdown()
}

func (s *ServiceStateManagerSuite) LogBatch(c *C, b ssm.ServiceStateChangeBatch) {
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

	c.Logf(`ssm.ServiceStateChangeBatch{
	Services: map[string]ssm.CancellableService{
		%v
	},
	DesiredState: %v,
	Emergency: %v,
}`, svcStr, b.DesiredState, b.Emergency)
}

func (s *ServiceStateManagerSuite) CompareBatchSlices(c *C, a, b []ssm.ServiceStateChangeBatch) bool {
	for n, batchA := range a {
		batchB := b[n]
		s.LogBatch(c, batchA)
		s.LogBatch(c, batchB)
		if !s.CompareBatches(c, batchA, batchB) {
			return false
		}
	}
	return true
}

// CompareBatches compares two batches and ignores the channels
func (s *ServiceStateManagerSuite) CompareBatches(c *C, a, b ssm.ServiceStateChangeBatch) bool {
	if !(a.DesiredState == b.DesiredState && a.Emergency == b.Emergency) {
		return false
	}
	for id, svc := range a.Services {
		c.Logf("svc1: id: %v svc:%+v", id, svc)
		c.Logf("svc2: id: %v svc:%+v", id, b.Services[id])
		if !s.CompareCancellableServices(svc, b.Services[id]) {
			return false
		}
	}
	return true
}

func (s *ServiceStateManagerSuite) CompareCancellableServices(a, b ssm.CancellableService) bool {
	return a.ID == b.ID && a.DesiredState == b.DesiredState &&
		a.EmergencyShutdownLevel == b.EmergencyShutdownLevel && a.StartLevel == b.StartLevel
}
