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
	"time"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/host"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetHost(c *C) {
	hostID := "someHostID"
	expectedHost := host.Host{ID: hostID}
	key := host.HostKey(hostID)
	ft.hostStore.On("Get", ft.ctx, key, mock.AnythingOfType("*host.Host")).
		Return(nil).
		Run(func(args mock.Arguments) {
			host := args.Get(2).(*host.Host)
			*host = expectedHost
		})

	result, err := ft.Facade.GetHost(ft.ctx, hostID)

	c.Assert(err, IsNil)
	c.Assert(result.ID, Equals, hostID)
}

func (ft *FacadeUnitTest) Test_GetHostFailsForNoSuchEntity(c *C) {
	hostID := "someHostID"
	key := host.HostKey(hostID)
	ft.hostStore.On("Get", ft.ctx, key, mock.AnythingOfType("*host.Host")).Return(datastore.ErrNoSuchEntity{})

	result, err := ft.Facade.GetHost(ft.ctx, hostID)

	c.Assert(result, IsNil)
	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_GetHostFailsForOtherDBError(c *C) {
	hostID := "someHostID"
	key := host.HostKey(hostID)
	expectedError := datastore.ErrEmptyKind
	ft.hostStore.On("Get", ft.ctx, key, mock.AnythingOfType("*host.Host")).Return(expectedError)

	result, err := ft.Facade.GetHost(ft.ctx, hostID)

	c.Assert(result, IsNil)
	c.Assert(err, Equals, expectedError)
}

func (ft *FacadeUnitTest) TestGetReadHostsShouldReturnCorrectValuesForHost(c *C) {
	ft.setupMockDFSLocking()

	expectedHost := getTestHost()

	ft.hostStore.On("GetN", ft.ctx, uint64(20000)).
		Return([]host.Host{expectedHost}, nil)

	hosts, err := ft.Facade.GetReadHosts(ft.ctx)
	c.Assert(err, IsNil)
	c.Assert(hosts, Not(IsNil))
	c.Assert(len(hosts), Equals, 1)

	h := hosts[0]
	c.Assert(h.ID, Equals, expectedHost.ID)
	c.Assert(h.Name, Equals, expectedHost.Name)
	c.Assert(h.PoolID, Equals, expectedHost.PoolID)
	c.Assert(h.Cores, Equals, expectedHost.Cores)
	c.Assert(h.Memory, Equals, expectedHost.Memory)
	c.Assert(h.RAMLimit, Equals, expectedHost.RAMLimit)
	c.Assert(h.KernelVersion, Equals, expectedHost.KernelVersion)
	c.Assert(h.KernelRelease, Equals, expectedHost.KernelRelease)
	c.Assert(h.ServiceD.Date, Equals, expectedHost.ServiceD.Date)
	c.Assert(h.ServiceD.Release, Equals, expectedHost.ServiceD.Release)
	c.Assert(h.ServiceD.Version, Equals, expectedHost.ServiceD.Version)
	c.Assert(h.CreatedAt, TimeEqual, expectedHost.CreatedAt)
	c.Assert(h.UpdatedAt, TimeEqual, expectedHost.UpdatedAt)
}

func (ft *FacadeUnitTest) Test_FindReadHostsInPoolShouldReturnCorrectValues(c *C) {
	ft.setupMockDFSLocking()

	expectedHost := getTestHost()

	ft.hostStore.On("FindHostsWithPoolID", ft.ctx, "name").
		Return([]host.Host{expectedHost}, nil)

	result, err := ft.Facade.FindReadHostsInPool(ft.ctx, "name")

	c.Assert(err, IsNil)
	c.Assert(result, Not(IsNil))

	h := result[0]
	c.Assert(h.ID, Equals, expectedHost.ID)
	c.Assert(h.Name, Equals, expectedHost.Name)
	c.Assert(h.PoolID, Equals, expectedHost.PoolID)
	c.Assert(h.Cores, Equals, expectedHost.Cores)
	c.Assert(h.Memory, Equals, expectedHost.Memory)
	c.Assert(h.RAMLimit, Equals, expectedHost.RAMLimit)
	c.Assert(h.KernelVersion, Equals, expectedHost.KernelVersion)
	c.Assert(h.KernelRelease, Equals, expectedHost.KernelRelease)
	c.Assert(h.ServiceD.Date, Equals, expectedHost.ServiceD.Date)
	c.Assert(h.ServiceD.Release, Equals, expectedHost.ServiceD.Release)
	c.Assert(h.ServiceD.Version, Equals, expectedHost.ServiceD.Version)
	c.Assert(h.CreatedAt, TimeEqual, expectedHost.CreatedAt)
	c.Assert(h.UpdatedAt, TimeEqual, expectedHost.UpdatedAt)
}

func getTestHost() host.Host {
	return host.Host{
		ID:            "expectedHost",
		Name:          "ExpectedHost",
		PoolID:        "Pool",
		Cores:         12,
		Memory:        15000,
		RAMLimit:      "50%",
		KernelVersion: "1.1.1",
		KernelRelease: "1.2.3",
		ServiceD: struct {
			Version   string
			Date      string
			Gitcommit string
			Gitbranch string
			Buildtag  string
			Release   string
		}{
			Version:   "1.2.3.4.5",
			Date:      "1/1/1999",
			Gitcommit: "commit",
			Gitbranch: "branch",
			Buildtag:  "tag",
			Release:   "Release",
		},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}
}
