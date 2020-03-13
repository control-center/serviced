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

package facade_test

import (
	"errors"

	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/logfilter"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/stretchr/testify/mock"
	. "gopkg.in/check.v1"
)

func (ft *FacadeUnitTest) Test_UpdateLogFilters_AddsNewFilters(c *C) {
	template := buildTestTemplate()
	var addResult error
	ft.setupAddFilter(c, template, addResult)

	err := ft.Facade.UpdateLogFilters(ft.ctx, template)

	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_UpdateLogFilters_AddFails(c *C) {
	template := buildTestTemplate()
	addResult := errors.New("mock add failed")
	ft.setupAddFilter(c, template, addResult)

	err := ft.Facade.UpdateLogFilters(ft.ctx, template)

	c.Assert(err, NotNil)
	c.Assert(err, Equals, addResult)
}

func (ft *FacadeUnitTest) Test_UpdateLogFilters_UpdatesExistingFilters(c *C) {
	template := buildTestTemplate()
	var updateResult error
	ft.setupUpdateFilter(c, template, updateResult)

	err := ft.Facade.UpdateLogFilters(ft.ctx, template)

	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_UpdateLogFilters_UpdateFails(c *C) {
	template := buildTestTemplate()
	updateResult := errors.New("mock update failed")
	ft.setupUpdateFilter(c, template, updateResult)

	err := ft.Facade.UpdateLogFilters(ft.ctx, template)

	c.Assert(err, NotNil)
	c.Assert(err, Equals, updateResult)
}

func (ft *FacadeUnitTest) Test_UpdateLogFilters_GenericGetFailure(c *C) {
	template := buildTestTemplate()
	getError := errors.New("mock get failed")
	ft.logfilterStore.On("Get", ft.ctx, "filter1", template.Version).Return(nil, getError)

	err := ft.Facade.UpdateLogFilters(ft.ctx, template)

	c.Assert(err, NotNil)
	c.Assert(err, Equals, getError)
}

func (ft *FacadeUnitTest) Test_RemoveLogFilters_RemovesFilters(c *C) {
	template := buildTestTemplate()
	ft.logfilterStore.On("Delete", ft.ctx, "filter1", template.Version).Return(nil)

	err := ft.Facade.RemoveLogFilters(ft.ctx, template)

	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_RemoveLogFilters_IgnoresNoSuchEntity(c *C) {
	template := buildTestTemplate()
	removeError := datastore.ErrNoSuchEntity{}
	ft.logfilterStore.On("Delete", ft.ctx, "filter1", template.Version).Return(removeError)

	err := ft.Facade.RemoveLogFilters(ft.ctx, template)

	c.Assert(err, IsNil)
}

func (ft *FacadeUnitTest) Test_RemoveLogFilters_ReportsUnexpectedErrors(c *C) {
	template := buildTestTemplate()
	removeError := errors.New("mock delete failed")
	ft.logfilterStore.On("Delete", ft.ctx, "filter1", template.Version).Return(removeError)

	err := ft.Facade.RemoveLogFilters(ft.ctx, template)

	c.Assert(err, NotNil)
	c.Assert(err, Equals, removeError)
}

func buildTestTemplate() *servicetemplate.ServiceTemplate {
	return &servicetemplate.ServiceTemplate{
		ID:      "tid1",
		Name:    "template1",
		Version: "1.0",
		Services: []servicedefinition.ServiceDefinition{
			servicedefinition.ServiceDefinition{
				Name: "service1",
				LogFilters: map[string]string{
					"filter1": "new filter",
				},
			},
		},
	}
}

func (ft *FacadeUnitTest) setupAddFilter(c *C, template *servicetemplate.ServiceTemplate, result error) {
	ft.logfilterStore.On("Get", ft.ctx, "filter1", template.Version).Return(nil, datastore.ErrNoSuchEntity{})
	ft.logfilterStore.On("Put", ft.ctx, mock.AnythingOfType("*logfilter.LogFilter")).
		Run(func(args mock.Arguments) {
			filter := args.Get(1).(*logfilter.LogFilter)
			c.Assert(filter.Version, Equals, template.Version)
			c.Assert(filter.Filter, Equals, template.Services[0].LogFilters["filter1"])
		}).
		Return(result)
}

func (ft *FacadeUnitTest) setupUpdateFilter(c *C, template *servicetemplate.ServiceTemplate, result error) {
	existingFilter := &logfilter.LogFilter{
		Name:    "filter1",
		Version: template.Version,
		Filter:  "old filter",
	}
	ft.logfilterStore.On("Get", ft.ctx, "filter1", template.Version).Return(existingFilter, nil)
	ft.logfilterStore.On("Put", ft.ctx, existingFilter).
		Run(func(args mock.Arguments) {
			filter := args.Get(1).(*logfilter.LogFilter)
			c.Assert(filter.Version, Equals, template.Version)
			c.Assert(filter.Filter, Equals, template.Services[0].LogFilters["filter1"])
		}).
		Return(result)
}
