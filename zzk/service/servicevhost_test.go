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

package service

import (
	"github.com/control-center/serviced/coordinator/client/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"testing"
)

// Current test is very basic. For future work, I'd like to have the following tests:
// Initial conditions:
// - no servicevhosts node
// - empty servicevhosts node
// - servicevhosts node with other data
// - servicevhosts node already populated
// Inputs:
// - service with no vhost - no change to initial conditions
// - service with vhost - end state should have servicevhosts node populated with service entry

type ServiceProfile struct {
	Name       string
	VHostNames []string
}

func makeService(sp ServiceProfile) *service.Service {
	svc := new(service.Service)
	svc.Name = sp.Name
	for _, vhn := range sp.VHostNames {
		svc.Endpoints = append(
			svc.Endpoints, service.ServiceEndpoint{
				Name: vhn,
				VHostList: []servicedefinition.VHost{
					{
						Name:    vhn,
						Enabled: true,
					},
				},
			})
	}
	svc.Endpoints = []service.ServiceEndpoint{
		{
			VHostList: []servicedefinition.VHost{
				{
					Name:    "test1",
					Enabled: true,
				},
			},
		},
	}
	return svc
}

func TestUpdateServiceVHosts(t *testing.T) {
	testSvcProfile := ServiceProfile{
		Name:       "Test Service",
		VHostNames: []string{"vhost1", "vhost2"},
	}

	mockConn := new(mocks.Connection)
	mockSvc := makeService(testSvcProfile)

	mockConn.On("Children", zkServiceVhosts).Return([]string{"foo", "bar"}, nil)
	mockConn.On("Get", zkServiceVhosts+"/:vhOn:__test1", &ServiceVhostNode{"", "", false, nil}).Return(nil)
	mockConn.On("Set", zkServiceVhosts+"/:vhOn:__test1", &ServiceVhostNode{"", "test1", true, nil}).Return(nil)

	UpdateServiceVhosts(mockConn, mockSvc)

	mockConn.AssertNumberOfCalls(t, "Children", 1)
	mockConn.AssertNumberOfCalls(t, "Set", 1)
}
