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
	"net"

	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/zzk/registry"
	. "gopkg.in/check.v1"
)

func (ft *FacadeIntegrationTest) Test_PublicEndpoint_PortAdd(c *C) {
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
			service.ServiceEndpoint{
				Application: "zproxy",
				Name:        "zproxy",
				PortNumber:  8080,
				Protocol:    "tcp",
				Purpose:     "export",
			},
		},
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
			service.ServiceEndpoint{
				Application: "service2",
				Name:        "service2",
				PortNumber:  9090,
				Protocol:    "tcp",
				Purpose:     "export",
			},
		},
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
	ft.zzk.On("CheckRunningPublicEndpoint", registry.PublicEndpointKey(":22222-1"), svcA.ID).Return(nil)
	port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	c.Assert(err, IsNil)
	if port == nil {
		c.Errorf("Adding a valid public endpoint port returned a nil port")
	}

	// Add a duplicate port.
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding a duplicate port")
	}

	// Add a port with an invalid port range.
	portAddr = ":70000"
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding an invalid port address %s", portAddr)
	}

	portAddr = ":0"
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding an invalid port address %s", portAddr)
	}

	portAddr = ":-1"
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding an invalid port address %s", portAddr)
	}

	// Add a port for an invalid service.
	portAddr = ":22223"
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, "invalid", endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding a port to an invalid service")
	}

	// Add a port to a service that's defined in another service.
	// Add a port for an invalid service.
	portAddr = ":22222"
	port, err = ft.Facade.AddPublicEndpointPort(ft.CTX, svcB.ID, endpointName, portAddr,
		usetls, protocol, isEnabled, restart)
	if err == nil {
		c.Errorf("Expected failure adding a port that already exists in another service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd: PASSED")
}

func (ft *FacadeIntegrationTest) Test_PublicEndpoint_PortRemove(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortRemove: STARTED")

	// Add a service so we can test our public endpoint.
	svcA := service.Service{
		ID:           "validate-service-tenant-A",
		Name:         "TestFacade_validateServiceTenantA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				Application: "zproxy",
				Name:        "zproxy",
				PortNumber:  8080,
				Protocol:    "tcp",
				Purpose:     "export",
				PortList: []servicedefinition.Port{
					servicedefinition.Port{
						PortAddr: ":22222",
						Enabled:  true,
						UseTLS:   true,
						Protocol: "https",
					},
				},
			},
		},
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)

	// Remove port.
	err := ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222")
	c.Assert(err, IsNil)

	// Remove port with an invalid service
	err = ft.Facade.RemovePublicEndpointPort(ft.CTX, "invalid", "zproxy", ":22222")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid service")
	}

	// Remove port with an invalid endpoint
	err = ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "invalid", ":22222")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid endpoint")
	}

	// Remove port with an invalid port address
	err = ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":55555")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid port address")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortRemove: PASSED")
}

func (ft *FacadeIntegrationTest) Test_PublicEndpoint_PortEnable(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable: STARTED")

	// Add a service so we can test our public endpoint.
	svcA := service.Service{
		ID:           "validate-service-tenant-A",
		Name:         "TestFacade_validateServiceTenantA",
		DeploymentID: "deployment-id",
		PoolID:       "pool-id",
		Launch:       "auto",
		DesiredState: int(service.SVCStop),
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				Application: "zproxy",
				Name:        "zproxy",
				PortNumber:  8080,
				Protocol:    "tcp",
				Purpose:     "export",
				PortList: []servicedefinition.Port{
					servicedefinition.Port{
						PortAddr: ":22222",
						Enabled:  true,
						UseTLS:   true,
						Protocol: "https",
					},
				},
			},
		},
	}
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)

	// The public endpoint should be enabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == false {
		c.Errorf("Expected service port public endpoint to be enabled")
	}

	// Disable the port.
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222", false)
	c.Assert(err, IsNil)
	svc, err = ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == true {
		c.Errorf("Expected service port public endpoint to be disabled")
	}

	// Enable the port.
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222", true)
	c.Assert(err, IsNil)
	svc, err = ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == false {
		c.Errorf("Expected service port public endpoint to be enabled")
	}

	// Enable a port with an invalid serviceid
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, "invalid", "zproxy", ":22222", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid service")
	}

	// Enable a port with an invalid application/endpoint id
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "invalid", ":22222", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid endpoint")
	}

	// Enable a port with invalid port addresses.
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", "invalid", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid port address")
	}
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":-5000", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid port address")
	}
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":0", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid port address")
	}
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":77777", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid port address")
	}

	// Try to enable a port with a port number that's already in use.
	func() {
		// Open an unused port first.
		listener, err := net.Listen("tcp", ":0")
		c.Assert(err, IsNil)
		defer listener.Close()
		// Now try to enable a port with the same address we just opened.
		err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", listener.Addr().String(), true)
		if err == nil {
			c.Errorf("Expected failure enabling a port that's already in use")
		}
	}()

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable: PASSED")
}
