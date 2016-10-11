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

// +build integration

package facade

import (
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

func (ft *FacadeIntegrationTest) TestGetServiceAddressAssignmentDetails(c *C) {
	// add a pool
	p := &pool.ResourcePool{
		ID: "poolid",
	}
	c.Assert(ft.Facade.AddResourcePool(ft.CTX, p), IsNil)

	// add a host
	h := &host.Host{
		ID:      "deadb11f",
		PoolID:  "poolid",
		Name:    "h1",
		IPAddr:  "12.27.36.45",
		RPCPort: 65535,
		IPs: []host.HostIPResource{
			{
				HostID:    "deadb11f",
				IPAddress: "12.27.36.45",
			},
		},
	}
	_, err := ft.Facade.AddHost(ft.CTX, h)
	c.Assert(err, IsNil)

	// add a service with an address assignment
	svcA := service.Service{
		ID:           "serviceid1",
		Name:         "svcA",
		DeploymentID: "depid",
		PoolID:       "poolid",
		Launch:       "auto",
		DesiredState: 0,
		Endpoints: []service.ServiceEndpoint{
			{
				Name:        "ep1",
				Application: "ep1",
				Purpose:     "export",
				AddressConfig: servicedefinition.AddressResourceConfig{
					Port:     1234,
					Protocol: "tcp",
				},
			},
		},
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)

	// add two child services, with and without address assignments
	svcB := service.Service{
		ID:              "serviceid2",
		Name:            "svcB",
		ParentServiceID: "serviceid1",
		DeploymentID:    "depid",
		PoolID:          "poolid",
		Launch:          "auto",
		DesiredState:    0,
		Endpoints: []service.ServiceEndpoint{
			{
				Name:        "ep2",
				Application: "ep2",
				Purpose:     "export",
				AddressConfig: servicedefinition.AddressResourceConfig{
					Port:     2123,
					Protocol: "tcp",
				},
			},
		},
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)

	svcC := service.Service{
		ID:              "serviceid3",
		Name:            "svcC",
		ParentServiceID: "serviceid1",
		DeploymentID:    "depid",
		PoolID:          "poolid",
		Launch:          "auto",
		DesiredState:    0,
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcC), IsNil)

	// assign the ips
	req := addressassignment.AssignmentRequest{
		ServiceID:      "serviceid1",
		IPAddress:      "12.27.36.45",
		AutoAssignment: false,
	}
	c.Assert(ft.Facade.AssignIPs(ft.CTX, req), IsNil)

	addrs, err := ft.Facade.GetServiceAddressAssignmentDetails(ft.CTX, "serviceid1", false)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 1)
	expected := []service.IPAssignment{
		{
			ServiceID:   "serviceid1",
			ServiceName: "svcA",
			PoolID:      "poolid",
			Type:        "static",
			HostID:      "deadb11f",
			HostName:    "h1",
			IPAddress:   "12.27.36.45",
			Port:        1234,
		},
	}
	c.Assert(addrs, DeepEquals, expected)

	addrs, err = ft.Facade.GetServiceAddressAssignmentDetails(ft.CTX, "serviceid1", true)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 2)
	expected = append(expected, service.IPAssignment{
		ServiceID:   "serviceid2",
		ServiceName: "svcB",
		PoolID:      "poolid",
		Type:        "static",
		HostID:      "deadb11f",
		HostName:    "h1",
		IPAddress:   "12.27.36.45",
		Port:        2123,
	})
	c.Assert(addrs, DeepEquals, expected)

	addrs, err = ft.Facade.GetServiceAddressAssignmentDetails(ft.CTX, "serviceid3", false)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 0)

	addrs, err = ft.Facade.GetServiceAddressAssignmentDetails(ft.CTX, "serviceid3", true)
	c.Assert(err, IsNil)
	c.Assert(addrs, HasLen, 0)
}
