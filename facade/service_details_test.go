// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, sotware
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package facade

import (
	"fmt"
	"time"

	"github.com/control-center/serviced/domain/service"
	. "github.com/control-center/serviced/utils/checkers"

	. "gopkg.in/check.v1"
)

func (t *IntegrationTest) AddServices(c *C) {
	zenossCore := service.Service{
		ID:           "1111",
		Name:         "Zenoss.Core",
		DeploymentID: "deployment",
		PoolID:       "pool",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Tags:         []string{"daemon"},
	}
	c.Assert(t.Facade.AddService(t.CTX, zenossCore), IsNil)

	zenjobs := service.Service{
		ID:              "2222",
		Name:            "zenjobs",
		DeploymentID:    "deployment",
		PoolID:          "pool",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Tags:            []string{"daemon"},
		ParentServiceID: "1111",
	}
	c.Assert(t.Facade.AddService(t.CTX, zenjobs), IsNil)

	zenoss := service.Service{
		ID:           "3333",
		Name:         "Zenoss",
		DeploymentID: "deployment",
		PoolID:       "pool",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Tags:         []string{"zenoss-application"},
	}
	c.Assert(t.Facade.AddService(t.CTX, zenoss), IsNil)

	zenperfsnmp := service.Service{
		ID:              "4444",
		Name:            "zenperfsnmp",
		DeploymentID:    "deployment",
		PoolID:          "pool",
		Launch:          "auto",
		DesiredState:    int(service.SVCStop),
		Tags:            []string{"collector", "daemon"},
		ParentServiceID: "1111",
	}
	c.Assert(t.Facade.AddService(t.CTX, zenperfsnmp), IsNil)
}

func (t *IntegrationTest) TestQueryHasChildren(c *C) {
	t.AddServices(c)
	details, _ := t.Facade.QueryServiceDetails(t.CTX, service.Query{})
	c.Assert(details, HasLen, 4)

	for _, d := range details {
		if d.ID == "1111" {
			c.Assert(d.HasChildren, IsTrue)
		} else {
			c.Assert(d.HasChildren, IsFalse)
		}
	}
}

func (t *IntegrationTest) TestQueryHasChildrenWorksIfChildrenNotReturned(c *C) {
	t.AddServices(c)
	query := service.Query{Name: "Zenoss.Core"}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)
	c.Assert(details, HasLen, 1)
	core := details[0]
	c.Assert(core.HasChildren, IsTrue)
}

func (t *IntegrationTest) TestQuery(c *C) {
	t.AddServices(c)
	details, _ := t.Facade.QueryServiceDetails(t.CTX, service.Query{})
	c.Assert(details, HasLen, 4)
}

func (t *IntegrationTest) TestQueryName(c *C) {
	t.AddServices(c)
	query := service.Query{Name: "Zen"}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)

	c.Assert(details, HasLen, 2)
	c.Assert(details[0].Name == "Zenoss" || details[1].Name == "Zenoss", IsTrue)
	c.Assert(details[0].Name == "Zenoss.Core" || details[1].Name == "Zenoss.Core", IsTrue)
}

func (t *IntegrationTest) TestQuerySingleTag(c *C) {
	t.AddServices(c)
	query := service.Query{Tags: []string{"zenoss-application"}}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)

	c.Assert(details, HasLen, 1)
	c.Assert(details[0].Name == "Zenoss", IsTrue)
}

func (t *IntegrationTest) TestQueryMultipleTag(c *C) {
	t.AddServices(c)
	query := service.Query{Tags: []string{"daemon", "collector"}}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)

	fmt.Println(details[0])
	c.Assert(details, HasLen, 1)
	c.Assert(details[0].Name == "zenperfsnmp", IsTrue)
}

func (t *IntegrationTest) TestQuerySince(c *C) {
	t.AddServices(c)
	firstMaxUpdateTime := t.getLatestUpdatedAt()
	time.Sleep(time.Second)

	zenperfsnmp := service.Service{
		ID:           "5555",
		Name:         "newzenperfsnmp",
		DeploymentID: "deployment",
		PoolID:       "pool",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Tags:         []string{"collector", "daemon"},
	}
	c.Assert(t.Facade.AddService(t.CTX, zenperfsnmp), IsNil)
	newMaxUpdateTime := t.getLatestUpdatedAt()

	difference := newMaxUpdateTime.Sub(firstMaxUpdateTime)
	since := difference / 2

	query := service.Query{Since: since}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)

	c.Assert(details, HasLen, 1)
	c.Assert(details[0].Name == "newzenperfsnmp", IsTrue)
}

func (t *IntegrationTest) TestQueryTenants(c *C) {
	t.AddServices(c)
	query := service.Query{Tenants: true}
	details, _ := t.Facade.QueryServiceDetails(t.CTX, query)

	c.Assert(details, HasLen, 2)
	c.Assert(details[0].Name == "Zenoss" || details[1].Name == "Zenoss", IsTrue)
	c.Assert(details[0].Name == "Zenoss.Core" || details[1].Name == "Zenoss.Core", IsTrue)
}

func (t *IntegrationTest) getLatestUpdatedAt() time.Time {
	details, _ := t.Facade.QueryServiceDetails(t.CTX, service.Query{})
	var max time.Time
	for _, detail := range details {
		fmt.Println(detail.UpdatedAt, detail.ID)
		if detail.UpdatedAt.After(max) {
			max = detail.UpdatedAt
		}
	}
	return max
}
