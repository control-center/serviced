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

// +build integration

package facade

import (
    "fmt"

	"github.com/control-center/serviced/domain/service"
)

func (ft *FacadeIntegrationTest) Test_PublicEndpoint_PortAdd(t *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd: starting")

    // Add a service so we can test our public endpoint.
	svcA := service.Service{
		ID:           "validate-service-tenant-A",
		Name:         "TestFacade_validateServiceTenantA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
        Endpoints: []service.ServiceEndpoint{
            service.ServiceEndpoint {
                Application: "zproxy",
                Name: "zproxy",
                PortList: DefaultTestPublicEndpointPorts,
                PortNumber: 8080,
                Protocol: "tcp",
                Purpose: "export",
            },
        }
	}
    // Add a service so we can test our public endpoint.
	svcB := service.Service{
		ID:           "validate-service-tenant-B",
		Name:         "TestFacade_validateServiceTenantB",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
        Endpoints: []service.ServiceEndpoint{
            service.ServiceEndpoint {
                Application: "service2",
                Name: "service2",
                PortList: DefaultTestPublicEndpointPorts,
                PortNumber: 9090,
                Protocol: "tcp",
                Purpose: "export",
            },
        }
	}
    c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
    c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)

    endpointName := "zproxy"
    portAddr := ":22222"
    usetls := true
    protocol := "http"
    isEnabled := true
    restart := false

    // Add a valid port.
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    t.Assert(err, IsNil)
    if port == nil {
        t.Errorf("Adding a valid public endpoint port returned a nil port")
    }

    // Add a duplicate port.
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding a duplicate port")
    }

    // Add a port with an invalid port range.
    portAddr = ":70000"
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding an invalid port address %s", portAddr)
    }

    portAddr = ":0"
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding an invalid port address %s", portAddr)
    }

    portAddr = ":-1"
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding an invalid port address %s", portAddr)
    }
    
    // Add a port for an invalid service.
    portAddr = ":22223"
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, "invalid", endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding a port to an invalid service", portAddr)
    }
    
    // Add a port to a service that's defined in another service.
    // Add a port for an invalid service.
    portAddr = ":22222"
    port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcB.ID, endpointName, portAddr,
        usetls, protocol, isEnabled, restart)
    if err == nil {
        t.Errorf("Expected failure adding a port that already exists in another service", portAddr)
    }
    
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd: PASSED")
}
