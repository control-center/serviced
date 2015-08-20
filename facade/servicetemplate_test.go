// Copyright 2014 The Serviced Authors.
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
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

func (ft *FacadeTest) TestDaoValidServiceForStart(t *C) {
	testService := service.Service{
		ID: "TestDaoValidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name:        "TestDaoValidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
				}),
		},
	}
	err := ft.Facade.validateServicesForStarting(datastore.Get(), &testService)
	if err != nil {
		t.Error("Services failed validation for starting: ", err)
	}
}

func (ft *FacadeTest) TestDaoInvalidServiceForStart(t *C) {
	testService := service.Service{
		ID: "TestDaoInvalidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(
				servicedefinition.EndpointDefinition{
					Name:        "TestDaoInvalidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				}),
		},
	}
	err := ft.Facade.validateServicesForStarting(datastore.Get(), &testService)
	if err == nil {
		t.Error("Services should have failed validation for starting...")
	}
}

func (ft *FacadeTest) TestRenameImageID(t *C) {
	imageId, err := renameImageID("localhost:5000", "quay.io/zenossinc/daily-zenoss5-core:5.0.0_123", "X")
	if err != nil {
		t.Errorf("unexpected failure renamingImageID: %s", err)
		t.FailNow()
	}
	expected := "localhost:5000/X/daily-zenoss5-core"
	if imageId != expected {
		t.Errorf("expected image '%s' got '%s'", expected, imageId)
		t.FailNow()
	}
}
