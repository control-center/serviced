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
