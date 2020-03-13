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

	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	. "gopkg.in/check.v1"
)

var sa = servicedefinition.AddressResourceConfig{
	Port:     8080,
	Protocol: "tcp",
}

// Perform pre-test setup for each of the PublicEndpoint tests
func (ft *IntegrationTest) setupServiceWithPublicEndpoints(c *C) (service.Service, service.Service) {
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
				VHostList: []servicedefinition.VHost{
					servicedefinition.VHost{
						Name:    "zproxy",
						Enabled: true,
					},
				},
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
	ft.zzk.On("GetVHost", "zproxy").Return("", "", nil).Once()
	ft.zzk.On("GetPublicPort", ":22222").Return("", "", nil).Once()
	c.Assert(ft.Facade.AddService(ft.CTX, svcA), IsNil)
	c.Assert(ft.Facade.AddService(ft.CTX, svcB), IsNil)
	// add the resource pool (no permissions required)
	rp := pool.ResourcePool{ID: "pool-id"}
	if err := ft.Facade.AddResourcePool(ft.CTX, &rp); err != nil {
		c.Fatalf("Failed to add the default resource pool: %+v, %s", rp, err)
	}
	return svcA, svcB
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Add mock calls.
	ft.zzk.On("GetPublicPort", ":22222").Return("", "", nil)
	ft.zzk.On("GetVHost", "zproxy").Return("", "", nil)
	ft.zzk.On("GetPublicPort", ":33333").Return("", "", nil)

	// Add a valid port.
	port, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":33333",
		true, "http", true, false)
	c.Assert(err, IsNil)
	if port == nil {
		c.Errorf("Adding a valid public endpoint port returned a nil port")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_VerifyEnabledFlag(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_VerifyEnabledFlag: STARTED")

	// Add a service so we can test our public endpoint.
	_, svcB := ft.setupServiceWithPublicEndpoints(c)

	// Add mock calls.
	ft.zzk.On("GetPublicPort", ":12345").Return("", "", nil)

	// Add a new vhost with enabled=false.
	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcB.ID, "service2", ":12345", true, "http", false, false)
	c.Assert(err, IsNil)

	// Check to make sure the new vhost is *not* enabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcB.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == true {
		c.Errorf("Expected port public endpoint to be disabled")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_VerifyEnabledFlag: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_DuplicatePort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_DuplicatePort: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Add a duplicate port.
	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding a duplicate port")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_DuplicatePort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_OutOfRangePort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_OutOfRangePort: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Add a port with an invalid port range.
	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":70000",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding an out of range port address :70000")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_OutOfRangePort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_PortZero(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_PortZero: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":0",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding an invalid port address :0")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_PortZero: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_NegativePort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_NegativePort: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":-1",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding a negative port address :-1")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_NegativePort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_InvalidService: starting")

	// Add a service so we can test our public endpoint.
	ft.setupServiceWithPublicEndpoints(c)

	// Add a port for an invalid service.
	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, "invalid", "zproxy", ":22223",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding a port to an invalid service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortAdd_PortInAnotherService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_PortInAnotherService: starting")

	// Add a service so we can test our public endpoint.
	_, svcB := ft.setupServiceWithPublicEndpoints(c)

	// Add a port to a service that's defined in another service.
	_, err := ft.Facade.AddPublicEndpointPort(ft.CTX, svcB.ID, "service2", ":22222",
		true, "http", true, false)
	if err == nil {
		c.Errorf("Expected failure adding a port that already exists in another service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortAdd_PortInAnotherService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortRemove(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortRemove: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove port.
	ft.zzk.On("GetVHost", "zproxy").Return(svcA.ID, "zproxy", nil).Once()
	ft.zzk.On("GetPublicPort", ":22222").Return(svcA.ID, "zproxy", nil).Once()
	err := ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222")
	c.Assert(err, IsNil)

	fmt.Println(" ##### Test_PublicEndpoint_PortRemove: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortRemove_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidService: STARTED")

	// Add a service so we can test our public endpoint.
	ft.setupServiceWithPublicEndpoints(c)

	// Remove port with an invalid service
	err := ft.Facade.RemovePublicEndpointPort(ft.CTX, "invalid", "zproxy", ":22222")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortRemove_InvalidEndpoint(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidEndpoint: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove port with an invalid endpoint
	err := ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "invalid", ":22222")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid endpoint")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidEndpoint: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortRemove_InvalidPort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidPort: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove port with an invalid port address
	err := ft.Facade.RemovePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":55555")
	if err == nil {
		c.Errorf("Expected failure removing a port with an invalid port address")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortRemove_InvalidPort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortDisable(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortDisable: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// The public endpoint should be enabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == false {
		c.Errorf("Expected service port public endpoint to be enabled")
	}

	// Disable the port.
	ft.zzk.On("GetVHost", "zproxy").Return(svcA.ID, "zproxy", nil).Once()
	ft.zzk.On("GetPublicPort", ":22222").Return(svcA.ID, "zproxy", nil).Once()
	err = ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222", false)
	c.Assert(err, IsNil)
	svc, err = ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].PortList[0].Enabled == true {
		c.Errorf("Expected service port public endpoint to be disabled")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortDisable: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_EnabledPort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_EnabledPort: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable the port.
	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":22222", true)
	if err == nil {
		c.Errorf("Expected error enabling a port that's already enabled")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_EnabledPort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidService: STARTED")

	// Enable a port with an invalid serviceid
	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, "invalid", "zproxy", ":22222", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidEndpoint(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidEndpoint: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable a port with an invalid application/endpoint id
	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "invalid", ":22222", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid endpoint")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidEndpoint: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidPortAddr(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortAddr: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable a port with invalid port addresses.
	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", "invalid", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with an invalid port address")
	}
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortAddr: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_NegativePort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_NegativePort: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":-5000", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with a negative port")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_NegativePort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidPortZero(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortZero: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":0", true)
	if err == nil {
		c.Errorf("Expected failure enabling a port with port 0")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortZero: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidPortTooHigh(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortTooHigh: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	err := ft.Facade.EnablePublicEndpointPort(ft.CTX, svcA.ID, "zproxy", ":77777", true)
	if err == nil {
		c.Errorf("Expected failure enabling an out of range port address")
	}

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortTooHigh: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_PortEnable_InvalidPortInUse(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortInUse: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

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

	fmt.Println(" ##### Test_PublicEndpoint_PortEnable_InvalidPortInUse: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd_VerifyEnabledFlag(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_VerifyEnabledFlag: STARTED")

	// Add a service so we can test our public endpoint.
	_, svcB := ft.setupServiceWithPublicEndpoints(c)

	// Mock call expectations:
	ft.zzk.On("GetVHost", "service2").Return("", "", nil)

	// Add a new vhost with enabled=false.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, svcB.ID, "service2", "service2", false, false)
	c.Assert(err, IsNil)

	// Check to make sure the new vhost is *not* enabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcB.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].VHostList[0].Enabled == true {
		c.Errorf("Expected service vhost public endpoint to be disabled")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_VerifyEnabledFlag: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidService: STARTED")

	// Add a vhost to an invalid service.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, "invalid", "zproxy", "zproxy", true, true)
	if err == nil {
		c.Errorf("Expected failure adding a vhost with an invalid service id")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd_InvalidEndpoint(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidEndpoint: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Add a vhost to a service with an invalid endpoint.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, svcA.ID, "invalid", "zproxy", true, true)
	if err == nil {
		c.Errorf("Expected failure adding a vhost with an invalid endpoint")
	}
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidEndpoint: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd_DuplicateVHost(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_DuplicateVHost: STARTED")

	_, svcB := ft.setupServiceWithPublicEndpoints(c)

	// Add a vhost to a service, but another service already has this vhost.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, svcB.ID, "service2", "zproxy", true, true)
	if err == nil {
		c.Errorf("Expected failure adding a duplicate vhost name")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_DuplicateVHost: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd_InvalidVHostName(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidVHostName: STARTED")

	_, svcB := ft.setupServiceWithPublicEndpoints(c)

	// Mock call expectations:
	ft.zzk.On("GetVHost", "test#$%").Return("", "", nil)

	// Add a vhost to a service with a vhost name that contains invalid characters.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, svcB.ID, "service2", "test#$%", true, true)
	if err == nil {
		c.Errorf("Expected failure adding a vhost with invalid characters")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd_InvalidVHostName: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostAdd(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Mock call expectations:
	ft.zzk.On("GetPublicPort", ":22222").Return("", "", nil)
	ft.zzk.On("GetVHost", "zproxy").Return("", "", nil)
	ft.zzk.On("GetVHost", "zproxy2").Return("", "", nil)

	// Add a valid vhost entry.
	_, err := ft.Facade.AddPublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "zproxy2", true, true)
	if err != nil {
		c.Errorf("Unexpected failure adding a valid vhost")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostAdd: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_Remove(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_Remove: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove port.
	ft.zzk.On("GetVHost", "zproxy").Return(svcA.ID, "zproxy", nil).Once()
	ft.zzk.On("GetPublicPort", ":22222").Return(svcA.ID, "zproxy", nil).Once()
	err := ft.Facade.RemovePublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "zproxy")
	c.Assert(err, IsNil)

	fmt.Println(" ##### Test_PublicEndpointVHost_Remove: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostRemove_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidService: STARTED")

	// Add a service so we can test our public endpoint.
	ft.setupServiceWithPublicEndpoints(c)

	// Remove vhost with an invalid service
	err := ft.Facade.RemovePublicEndpointVHost(ft.CTX, "invalid", "zproxy", "zproxy")
	if err == nil {
		c.Errorf("Expected failure removing a vhost with an invalid service")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostRemove_InvalidEndpoint(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidEndpoint: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove vhost with an invalid endpoint
	err := ft.Facade.RemovePublicEndpointVHost(ft.CTX, svcA.ID, "invalid", "zproxy")
	if err == nil {
		c.Errorf("Expected failure removing a vhost with an invalid endpoint")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidEndpoint: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_VHostRemove_InvalidPort(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidPort: STARTED")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Remove vhost with an invalid port address
	err := ft.Facade.RemovePublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "invalid")
	if err == nil {
		c.Errorf("Expected failure removing a vhost that doesn't exist")
	}

	fmt.Println(" ##### Test_PublicEndpoint_VHostRemove_InvalidPort: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_Disable(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_Disable: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable a vhost.
	ft.zzk.On("GetVHost", "zproxy").Return(svcA.ID, "zproxy", nil).Once()
	ft.zzk.On("GetPublicPort", ":22222").Return(svcA.ID, "zproxy", nil).Once()
	err := ft.Facade.EnablePublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "zproxy", false)
	if err != nil {
		c.Errorf("Unexpected failure disabling a vhost")
	}

	// Make sure the vhost is disabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].VHostList[0].Enabled == true {
		c.Errorf("Expected service vhost public endpoint to be disabled")
	}

	fmt.Println(" ##### Test_PublicEndpointVHost_Disable: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_EnableEnabledVHost(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_EnableEnabledVHost: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Make sure the vhost is enabled.
	svc, err := ft.Facade.GetService(ft.CTX, svcA.ID)
	c.Assert(err, IsNil)
	if svc.Endpoints[0].VHostList[0].Enabled == false {
		c.Errorf("Expected service vhost public endpoint to be enabled")
	}

	// Enable a vhost.
	err = ft.Facade.EnablePublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "zproxy", true)
	if err == nil {
		c.Errorf("Expected failure enabling a vhost that is already enabled")
	}

	fmt.Println(" ##### Test_PublicEndpointVHost_EnableEnabledVHost: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_InvalidService(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidService: STARTED")

	// Enable a vhost on a service with an invalid service.
	err := ft.Facade.EnablePublicEndpointVHost(ft.CTX, "invalid", "zproxy", "zproxy", true)
	if err == nil {
		c.Errorf("Expected failure enabling a vhost with an invalid service")
	}

	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidService: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_InvalidEndpoint(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidEndpoint: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable a vhost on a service with an invalid endpoint.
	err := ft.Facade.EnablePublicEndpointVHost(ft.CTX, svcA.ID, "invalid", "zproxy", true)
	if err == nil {
		c.Errorf("Expected failure enabling a vhost with an invalid endpoint")
	}

	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidEndpoint: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpointVHost_InvalidVHost(c *C) {
	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidVHost: STARTED")

	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Enable a vhost on a service with an invalid endpoint.
	err := ft.Facade.EnablePublicEndpointVHost(ft.CTX, svcA.ID, "zproxy", "invalid", true)
	if err == nil {
		c.Errorf("Expected failure enabling an invalid vhost")
	}

	fmt.Println(" ##### Test_PublicEndpointVHost_InvalidVHost: PASSED")
}

func (ft *IntegrationTest) Test_PublicEndpoint_SetAddressConfig(c *C) {
	fmt.Println(" ##### Test_PublicEndpoint_SetAddressConfig: starting")

	// Add a service so we can test our public endpoint.
	svcA, _ := ft.setupServiceWithPublicEndpoints(c)

	// Add mock calls.
	ft.zzk.On("GetPublicPort", ":22222").Return("", "", nil)
	ft.zzk.On("GetVHost", "zproxy").Return("", "", nil)
	ft.zzk.On("GetPublicPort", ":33333").Return("", "", nil)

	err := ft.Facade.SetAddressConfig(ft.CTX, svcA.ID, "zproxy", sa)
	c.Assert(err, IsNil)

	fmt.Println(" ##### Test_PublicEndpoint_SetAddressConfig: PASSED")
}
