// Copyright 2015 The Serviced Authors.
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
//
// +build: unit

package health_test

import (
	"fmt"
	"testing"

	"github.com/control-center/serviced/dao"
	dao_mocks "github.com/control-center/serviced/dao/mocks"
	"github.com/control-center/serviced/domain"
	facade_mocks "github.com/control-center/serviced/facade/mocks"
	"github.com/control-center/serviced/health"
	"github.com/stretchr/testify/mock"

	. "gopkg.in/check.v1"
)

func TestHealth(t *testing.T) { TestingT(t) }

type HealthSuite struct{}

var (
	_ = Suite(&HealthSuite{})
)

const (
	passed = "passed"
)

func (s *HealthSuite) TestRegisterAndGetHealthCheck(c *C) {
	f := &facade_mocks.FacadeInterface{}
	cdao := &dao_mocks.ControlPlane{}
	hmap := health.NewHealthStatuses(cdao, f)

	serviceID := "abc123def456"
	instanceID := "0"
	name := "health_test"

	healthChecks := map[string]domain.HealthCheck{name: domain.HealthCheck{}}
	f.On("GetHealthChecksForService", nil, serviceID).Return(healthChecks, nil)

	hmap.SetHealthStatus(serviceID, instanceID, name, passed)
	result := hmap.GetHealthStatusesForService(serviceID)

	_, ok := result[instanceID]
	c.Assert(ok, Equals, true)
	status, ok := result[instanceID][name]
	c.Assert(ok, Equals, true)
	c.Assert(status.Status, Equals, passed)

	invalidhealthcheck := fmt.Sprintf("%s_x", name)
	hmap.SetHealthStatus(serviceID, instanceID, invalidhealthcheck, passed)
	result = hmap.GetHealthStatusesForService(serviceID)
	_, ok = result[instanceID]
	c.Assert(ok, Equals, true)
	_, ok = result[instanceID][invalidhealthcheck]
	c.Assert(ok, Equals, false)
}

func (s *HealthSuite) TestCleanup(c *C) {
	f := &facade_mocks.FacadeInterface{}
	cdao := &dao_mocks.ControlPlane{}
	hmap := health.NewHealthStatuses(cdao, f)

	serviceID1 := "abc123def456"
	serviceID2 := "ghi789jkl012"
	instanceID := "0"
	name := "health_test"

	healthChecks := map[string]domain.HealthCheck{
		name: domain.HealthCheck{},
	}

	f.On("GetHealthChecksForService", nil, serviceID1).Return(healthChecks, nil)
	hmap.SetHealthStatus(serviceID1, instanceID, name, passed)

	f.On("GetHealthChecksForService", nil, serviceID2).Return(healthChecks, nil)
	hmap.SetHealthStatus(serviceID2, instanceID, name, passed)

	result1 := hmap.GetHealthStatusesForService(serviceID1)
	_, ok := result1[instanceID]
	c.Assert(ok, Equals, true)
	_, ok = result1[instanceID][name]
	c.Assert(ok, Equals, true)

	result2 := hmap.GetHealthStatusesForService(serviceID2)
	_, ok = result2[instanceID]
	c.Assert(ok, Equals, true)
	_, ok = result2[instanceID][name]
	c.Assert(ok, Equals, true)

	svcs := []dao.RunningService{
		dao.RunningService{
			ServiceID:  serviceID2,
			InstanceID: 0,
		},
	}
	cdao.On("GetRunningServices", mock.Anything, mock.Anything).Return(nil).Run(func(args mock.Arguments) {
		s := args.Get(1).(*[]dao.RunningService)
		*s = svcs
	})

	hmap.CleanupOnce()

	result1 = hmap.GetHealthStatusesForService(serviceID1)
	_, ok = result1[instanceID]
	c.Assert(ok, Equals, false)

	result2 = hmap.GetHealthStatusesForService(serviceID2)
	_, ok = result2[instanceID]
	c.Assert(ok, Equals, true)

}
