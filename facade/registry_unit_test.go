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
	"github.com/control-center/serviced/domain/registry"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetRegistryImage(c *C) {
	imageID := "someImageID"
	expectedImage := registry.Image{UUID: imageID}
	ft.registryStore.On("Get", ft.ctx, imageID).
		Return(&expectedImage, nil)

	result, err := ft.Facade.GetRegistryImage(ft.ctx, imageID)

	c.Assert(err, IsNil)
	c.Assert(result.UUID, Equals, imageID)
}

func (ft *FacadeUnitTest) Test_GetRegistryImageFailsForNoSuchEntity(c *C) {
	imageID := "someImageID"
	ft.registryStore.On("Get", ft.ctx, imageID).
		Return(nil, datastore.ErrNoSuchEntity{})

	result, err := ft.Facade.GetRegistryImage(ft.ctx, imageID)

	c.Assert(result, IsNil)
	c.Assert(err, Equals, datastore.ErrNoSuchEntity{})
}

func (ft *FacadeUnitTest) Test_GetRegistryImageFailsForOtherDBError(c *C) {
	imageID := "someImageID"
	expectedError := datastore.ErrEmptyKind
	ft.registryStore.On("Get", ft.ctx, imageID).
		Return(nil, expectedError)

	result, err := ft.Facade.GetRegistryImage(ft.ctx, imageID)

	c.Assert(result, IsNil)
	c.Assert(err, Equals, expectedError)
}
