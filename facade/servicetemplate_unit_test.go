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
	"github.com/control-center/serviced/domain/servicetemplate"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_GetServiceTemplates(c *C) {
	var templateList []*servicetemplate.ServiceTemplate
	templateList = append(templateList, &servicetemplate.ServiceTemplate{ID: "template1"})
	templateList = append(templateList, &servicetemplate.ServiceTemplate{ID: "template2"})
	ft.templateStore.On("GetServiceTemplates", ft.ctx).Return(templateList, nil)

	result, err := ft.Facade.GetServiceTemplates(ft.ctx)

	c.Assert(err, IsNil)
	c.Assert(result, Not(IsNil))
	c.Assert(len(result), Equals, len(templateList))

	for _, expectedTemplate := range templateList {
		actualTemplate, ok := result[expectedTemplate.ID]
		c.Assert(ok, Equals, true)
		c.Assert(actualTemplate, Not(IsNil))
		c.Assert(actualTemplate, DeepEquals, *expectedTemplate)
	}
}

func (ft *FacadeUnitTest) Test_GetServiceTemplatesEmpty(c *C) {
	var emptyList []*servicetemplate.ServiceTemplate
	ft.templateStore.On("GetServiceTemplates", ft.ctx).Return(emptyList, nil)

	result, err := ft.Facade.GetServiceTemplates(ft.ctx)

	c.Assert(err, IsNil)
	c.Assert(result, Not(IsNil))
	c.Assert(len(result), Equals, 0)
}

func (ft *FacadeUnitTest) Test_GetServiceTemplatesFails(c *C) {
	expectedError := datastore.ErrEmptyKind
	ft.templateStore.On("GetServiceTemplates", ft.ctx).Return(nil, expectedError)

	result, err := ft.Facade.GetServiceTemplates(ft.ctx)

	c.Assert(err, Not(IsNil))
	c.Assert(err, Equals, expectedError)
	c.Assert(result, Not(IsNil))
	c.Assert(len(result), Equals, 0)
}
