// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

func (ft *FacadeTest) TestDaoValidServiceForStart(t *C) {
	testService := service.Service{
		Id: "TestDaoValidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "TestDaoValidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
				},
			},
		},
	}
	err := ft.Facade.validateServicesForStarting(datastore.Get(), &testService)
	if err != nil {
		t.Error("Services failed validation for starting: ", err)
	}
}

func (ft *FacadeTest) TestDaoInvalidServiceForStart(t *C) {
	testService := service.Service{
		Id: "TestDaoInvalidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "TestDaoInvalidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				},
			},
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
