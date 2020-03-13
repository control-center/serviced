// Copyright 2016 The Serviced Authors.
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

package facade_test

import (
	"errors"
	"time"

	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/registry"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/health"
	"github.com/control-center/serviced/metrics"
	"github.com/control-center/serviced/utils"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var (
	ErrTestZK         = errors.New("mock zookeeper error")
	ErrTestHostStore  = errors.New("mock host store error")
	ErrTestImageStore = errors.New("mock image store error")
)

var testStartTime = time.Now().Add(-time.Hour)

func (ft *FacadeUnitTest) TestGetServiceInstances_ServiceNotFound(c *C) {
	ft.serviceStore.On("Get", ft.ctx, "badservice").Return(nil, facade.ErrServiceDoesNotExist)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, testStartTime, "badservice")
	c.Assert(err, Equals, facade.ErrServiceDoesNotExist)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetServiceInstances_StatesError(c *C) {
	svc := &service.Service{
		ID:           "testservice",
		PoolID:       "default",
		Name:         "serviceA",
		ImageID:      "testtenant/image",
		DesiredState: int(service.SVCRun),
	}

	img := &registry.Image{
		Library: "testtenant",
		Repo:    "image",
		Tag:     "latest",
		UUID:    "someimageuuid",
	}
	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(img, nil)
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)
	ft.zzk.On("GetServiceStates", ft.ctx, "default", "testservice").Return(nil, ErrTestZK)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, testStartTime, "testservice")
	c.Assert(err, Equals, ErrTestZK)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetServiceInstances_HostNotFound(c *C) {
	svc := &service.Service{
		ID:           "testservice",
		PoolID:       "default",
		Name:         "serviceA",
		ImageID:      "testtenant/image",
		DesiredState: int(service.SVCRun),
	}
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetServiceStates", ft.ctx, "default", "testservice").Return(states, nil)

	img := &registry.Image{
		Library: "testtenant",
		Repo:    "image",
		Tag:     "latest",
		UUID:    "someimageuuid",
	}
	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(img, nil)
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(ErrTestHostStore)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, testStartTime, "testservice")
	c.Assert(err, Equals, ErrTestHostStore)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetServiceInstances_BadImage(c *C) {
	svc := &service.Service{
		ID:           "testservice",
		PoolID:       "default",
		Name:         "serviceA",
		ImageID:      "testtenant/image",
		DesiredState: int(service.SVCRun),
	}
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetServiceStates", ft.ctx, "default", "testservice").Return(states, nil)

	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(nil, ErrTestImageStore)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, testStartTime, "testservice")
	c.Assert(err, Equals, ErrTestImageStore)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetServiceInstances_Success(c *C) {
	svc := &service.Service{
		ID:           "testservice",
		PoolID:       "default",
		Name:         "serviceA",
		ImageID:      "testtenant/image",
		DesiredState: int(service.SVCRun),
	}
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
			CurrentStateContainer: zkservice.CurrentStateContainer{Status: service.StateRunning},
		},
	}
	ft.zzk.On("GetServiceStates", ft.ctx, "default", "testservice").Return(states, nil)

	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	img := &registry.Image{
		Library: "testtenant",
		Repo:    "image",
		Tag:     "latest",
		UUID:    "someimageuuid",
	}
	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(img, nil)

	expected := []service.Instance{
		{
			InstanceID:   1,
			HostID:       "testhost",
			HostName:     "sometest.host.org",
			ServiceID:    "testservice",
			ServiceName:  "serviceA",
			ContainerID:  "somecontainerid",
			ImageSynced:  true,
			DesiredState: service.SVCRun,
			CurrentState: service.StateRunning,
			HealthStatus: make(map[string]health.Status),
			MemoryUsage:  service.Usage{Cur: 5, Max: 10, Avg: 7},
			Scheduled:    states[0].Scheduled,
			Started:      states[0].Started,
			Terminated:   states[0].Terminated,
		},
	}

	ft.metricsClient.On("GetInstanceMemoryStats", testStartTime, []metrics.ServiceInstance{{ServiceID: "testservice", InstanceID: 1}}).Return(
		[]metrics.MemoryUsageStats{
			{HostID: "testhost", ServiceID: "testservice", InstanceID: "1", Last: 5, Max: 10, Average: 7},
		}, nil)

	actual, err := ft.Facade.GetServiceInstances(ft.ctx, testStartTime, "testservice")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (ft *FacadeUnitTest) TestGetHostInstances_HostNotFound(c *C) {
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(ErrTestHostStore)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, testStartTime, "testhost")
	c.Assert(err, Equals, ErrTestHostStore)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_StatesError(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	ft.zzk.On("GetHostStates", ft.ctx, "default", "testhost").Return(nil, ErrTestZK)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, testStartTime, "testhost")
	c.Assert(err, Equals, ErrTestZK)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_ServiceNotFound(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetHostStates", ft.ctx, "default", "testhost").Return(states, nil)

	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(nil, facade.ErrServiceDoesNotExist)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, testStartTime, "testhost")
	c.Assert(err, Equals, facade.ErrServiceDoesNotExist)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_Success(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.Key("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
			CurrentStateContainer: zkservice.CurrentStateContainer{Status: service.StateRunning},
		},
	}
	ft.zzk.On("GetHostStates", ft.ctx, "default", "testhost").Return(states, nil)

	svc := &service.Service{
		ID:           "testservice",
		PoolID:       "default",
		Name:         "serviceA",
		ImageID:      "testtenant/image",
		DesiredState: int(service.SVCRun),
	}
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)

	img := &registry.Image{
		Library: "testtenant",
		Repo:    "image",
		Tag:     "latest",
		UUID:    "someimageuuid",
	}
	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(img, nil)

	expected := []service.Instance{
		{
			InstanceID:   1,
			HostID:       "testhost",
			HostName:     "sometest.host.org",
			ServiceID:    "testservice",
			ServiceName:  "serviceA",
			ContainerID:  "somecontainerid",
			ImageSynced:  true,
			DesiredState: service.SVCRun,
			CurrentState: service.StateRunning,
			HealthStatus: make(map[string]health.Status),
			MemoryUsage:  service.Usage{Cur: 5, Max: 10, Avg: 7},
			Scheduled:    states[0].Scheduled,
			Started:      states[0].Started,
			Terminated:   states[0].Terminated,
		},
	}
	ft.metricsClient.On("GetInstanceMemoryStats", testStartTime, []metrics.ServiceInstance{{ServiceID: "testservice", InstanceID: 1}}).Return(
		[]metrics.MemoryUsageStats{
			{HostID: "testhost", ServiceID: "testservice", InstanceID: "1", Last: 5, Max: 10, Average: 7},
		}, nil)
	actual, err := ft.Facade.GetHostInstances(ft.ctx, testStartTime, "testhost")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (ft *FacadeUnitTest) TestGetHostStrategyInstances(c *C) {
	hst1 := host.Host{
		ID:     "testhost1",
		PoolID: "default",
	}
	states1 := []zkservice.State{
		{
			HostID:     "testhost1",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetHostStates", ft.ctx, "default", "testhost1").Return(states1, nil)

	hst2 := host.Host{
		ID:     "testhost2",
		PoolID: "default",
	}
	states2 := []zkservice.State{
		{
			HostID:     "testhost2",
			ServiceID:  "testservice",
			InstanceID: 2,
			HostState: zkservice.HostState{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageUUID:   "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetHostStates", ft.ctx, "default", "testhost2").Return(states2, nil)

	svc := &service.Service{
		ID:            "testservice",
		PoolID:        "default",
		Name:          "serviceA",
		CPUCommitment: 10,
		RAMCommitment: utils.EngNotation{
			Value: uint64(1000),
		},
		HostPolicy: servicedefinition.Pack,
	}
	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)

	expectedMap := map[string]service.StrategyInstance{
		hst1.ID: {
			HostID:        hst1.ID,
			ServiceID:     svc.ID,
			CPUCommitment: int(svc.CPUCommitment),
			RAMCommitment: svc.RAMCommitment.Value,
			HostPolicy:    svc.HostPolicy,
		},
		hst2.ID: {
			HostID:        hst2.ID,
			ServiceID:     svc.ID,
			CPUCommitment: int(svc.CPUCommitment),
			RAMCommitment: svc.RAMCommitment.Value,
			HostPolicy:    svc.HostPolicy,
		},
	}
	actual, err := ft.Facade.GetHostStrategyInstances(ft.ctx, []host.Host{hst1, hst2})
	c.Assert(err, IsNil)
	c.Assert(len(actual), Equals, len(expectedMap))
	for _, result := range actual {
		expected, ok := expectedMap[(*result).HostID]
		c.Assert(ok, Equals, true)
		c.Assert(*result, Equals, expected)
	}
}
