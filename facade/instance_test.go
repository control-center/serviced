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
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/health"
	zkservice "github.com/control-center/serviced/zzk/service2"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

var (
	ErrTestZK         = errors.New("mock zookeeper error")
	ErrTestHostStore  = errors.New("mock host store error")
	ErrTestImageStore = errors.New("mock image store error")
)

func (ft *FacadeUnitTest) TestGetServiceInstances_ServiceNotFound(c *C) {
	ft.serviceStore.On("Get", ft.ctx, "badservice").Return(nil, facade.ErrServiceDoesNotExist)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, "badservice")
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

	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(svc, nil)
	ft.zzk.On("GetServiceStates2", "default", "testservice").Return(nil, ErrTestZK)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, "testservice")
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
			HostState2: zkservice.HostState2{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageID:     "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetServiceStates2", "default", "testservice").Return(states, nil)

	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(ErrTestHostStore)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, "testservice")
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
			HostState2: zkservice.HostState2{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageID:     "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetServiceStates2", "default", "testservice").Return(states, nil)

	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	ft.registryStore.On("Get", ft.ctx, "testtenant/image:latest").Return(nil, ErrTestImageStore)
	inst, err := ft.Facade.GetServiceInstances(ft.ctx, "testservice")
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
			HostState2: zkservice.HostState2{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageID:     "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetServiceStates2", "default", "testservice").Return(states, nil)

	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
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
			ID:           1,
			HostID:       "testhost",
			HostName:     "sometest.host.org",
			ServiceID:    "testservice",
			ServiceName:  "serviceA",
			ContainerID:  "somecontainerid",
			ImageSynced:  true,
			DesiredState: service.SVCRun,
			CurrentState: service.Running,
			HealthStatus: make(map[string]health.Status),
			Scheduled:    states[0].Scheduled,
			Started:      states[0].Started,
			Terminated:   states[0].Terminated,
		},
	}
	actual, err := ft.Facade.GetServiceInstances(ft.ctx, "testservice")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}

func (ft *FacadeUnitTest) TestGetHostInstances_HostNotFound(c *C) {
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(ErrTestHostStore)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, "testhost")
	c.Assert(err, Equals, ErrTestHostStore)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_StatesError(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	ft.zzk.On("GetHostStates", "default", "testhost").Return(nil, ErrTestZK)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, "testhost")
	c.Assert(err, Equals, ErrTestZK)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_ServiceNotFound(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState2: zkservice.HostState2{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageID:     "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetHostStates", "default", "testhost").Return(states, nil)

	ft.serviceStore.On("Get", ft.ctx, "testservice").Return(nil, facade.ErrServiceDoesNotExist)
	inst, err := ft.Facade.GetHostInstances(ft.ctx, "testhost")
	c.Assert(err, Equals, facade.ErrServiceDoesNotExist)
	c.Assert(inst, IsNil)
}

func (ft *FacadeUnitTest) TestGetHostInstances_Success(c *C) {
	hst := &host.Host{
		ID:     "testhost",
		Name:   "sometest.host.org",
		PoolID: "default",
	}
	ft.hostStore.On("Get", ft.ctx, host.HostKey("testhost"), mock.AnythingOfType("*host.Host")).Return(nil).Run(func(args mock.Arguments) {
		arg := args.Get(2).(*host.Host)
		*arg = *hst
	})

	states := []zkservice.State{
		{
			HostID:     "testhost",
			ServiceID:  "testservice",
			InstanceID: 1,
			HostState2: zkservice.HostState2{
				DesiredState: service.SVCRun,
				Scheduled:    time.Now(),
			},
			ServiceState: zkservice.ServiceState{
				ImageID:     "someimageuuid",
				Started:     time.Now(),
				Paused:      false,
				ContainerID: "somecontainerid",
			},
		},
	}
	ft.zzk.On("GetHostStates", "default", "testhost").Return(states, nil)

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
			ID:           1,
			HostID:       "testhost",
			HostName:     "sometest.host.org",
			ServiceID:    "testservice",
			ServiceName:  "serviceA",
			ContainerID:  "somecontainerid",
			ImageSynced:  true,
			DesiredState: service.SVCRun,
			CurrentState: service.Running,
			HealthStatus: make(map[string]health.Status),
			Scheduled:    states[0].Scheduled,
			Started:      states[0].Started,
			Terminated:   states[0].Terminated,
		},
	}
	actual, err := ft.Facade.GetHostInstances(ft.ctx, "testhost")
	c.Assert(err, IsNil)
	c.Assert(actual, DeepEquals, expected)
}
