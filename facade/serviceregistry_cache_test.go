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

// +build unit

package facade

import (
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	zkr "github.com/control-center/serviced/zzk/registry"

	. "gopkg.in/check.v1"
)

var _ = Suite(&ServiceRegistryCacheTest{})

type ServiceRegistryCacheTest struct {
	cache        *serviceRegistryCache
}

func (t *ServiceRegistryCacheTest) SetUpTest(c *C) {
	t.cache = NewServiceRegistryCache()
}

func (t *ServiceRegistryCacheTest) TearDownTest(c *C) {
	t.cache = nil
}

func (t *ServiceRegistryCacheTest) Test_GetRegistryForService_CreatesCache(c *C) {
	service1 := "service1"
	svcRegistry := t.cache.GetRegistryForService(service1)

	c.Assert(len(t.cache.registry), Equals, 1)

	_, ok := t.cache.registry[service1]
	c.Assert(ok, Equals, true)

	c.Assert(svcRegistry.ServiceID, Equals, service1)
	t.assertPortMapsEqual(c, svcRegistry.PublicPorts, map[zkr.PublicPortKey]zkr.PublicPort{})
	t.assertVHostMapsEqual(c, svcRegistry.VHosts, map[zkr.VHostKey]zkr.VHost{})

	service2 := "service2"
	svcRegistry = t.cache.GetRegistryForService(service2)

	c.Assert(len(t.cache.registry), Equals, 2)

	_, ok = t.cache.registry[service2]
	c.Assert(ok, Equals, true)

	c.Assert(svcRegistry.ServiceID, Equals, service2)
	t.assertPortMapsEqual(c, svcRegistry.PublicPorts, map[zkr.PublicPortKey]zkr.PublicPort{})
	t.assertVHostMapsEqual(c, svcRegistry.VHosts, map[zkr.VHostKey]zkr.VHost{})
}

func (t *ServiceRegistryCacheTest) Test_BuildCache_ForEmptyValues(c *C) {
	publicPorts := make(map[zkr.PublicPortKey]zkr.PublicPort)
	vhosts := make(map[zkr.VHostKey]zkr.VHost)

	t.cache.BuildCache(publicPorts, vhosts)

	c.Assert(len(t.cache.registry), Equals, 0)
}

func (t *ServiceRegistryCacheTest) Test_BuildCache_ForSomeValues(c *C) {
	// "service1" - only has public ports
	// "service2" - has public ports and vhosts
	// "service3" - has only vhosts
	publicPorts := t.getExpectedPortsForService1()
	for key, value := range  t.getExpectedPortsForService2() {
		publicPorts[key] = value
	}
	vhosts :=  t.getExpectedVHostsForService2()
	for key, value := range  t.getExpectedVHostsForService3() {
		vhosts[key] = value
	}
	t.cache.BuildCache(publicPorts, vhosts)

	c.Assert(len(t.cache.registry), Equals, 3)

	svc1, ok := t.cache.registry["service1"]
	c.Assert(ok, Equals, true)
	c.Assert(svc1.ServiceID, Equals, "service1")
	t.assertPortMapsEqual(c,  svc1.PublicPorts, t.getExpectedPortsForService1())
	t.assertVHostMapsEqual(c, svc1.VHosts,      map[zkr.VHostKey]zkr.VHost{})

	svc2, ok := t.cache.registry["service2"]
	c.Assert(ok, Equals, true)
	c.Assert(svc2.ServiceID, Equals, "service2")
	t.assertPortMapsEqual(c,  svc2.PublicPorts, t.getExpectedPortsForService2())
	t.assertVHostMapsEqual(c, svc2.VHosts,      t.getExpectedVHostsForService2())

	svc3, ok := t.cache.registry["service3"]
	c.Assert(ok, Equals, true)
	c.Assert(svc3.ServiceID, Equals, "service3")
	t.assertPortMapsEqual(c,  svc3.PublicPorts, map[zkr.PublicPortKey]zkr.PublicPort{})
	t.assertVHostMapsEqual(c, svc3.VHosts,      t.getExpectedVHostsForService3())
}

func (t *ServiceRegistryCacheTest) Test_UpdateRegistry_WithEmptyValues(c *C) {
	// Load the cache with values for both ports and vhosts
	serviceID := "service2"
	t.cache.BuildCache(t.getExpectedPortsForService2(), t.getExpectedVHostsForService2())

	emptyPublicPorts := make(map[zkr.PublicPortKey]zkr.PublicPort)
	emptyVHosts := make(map[zkr.VHostKey]zkr.VHost)

	t.cache.UpdateRegistry(serviceID, emptyPublicPorts, emptyVHosts)

	svcRegistry := t.cache.GetRegistryForService(serviceID)
	c.Assert(len(svcRegistry.PublicPorts), Equals, 0)
	c.Assert(len(svcRegistry.VHosts), Equals, 0)
}

func (t *ServiceRegistryCacheTest) Test_UpdateRegistry_WithSomeValues(c *C) {
	// Load the cache with values for both ports and vhosts
	t.cache.BuildCache(t.getExpectedPortsForService2(), t.getExpectedVHostsForService2())

	emptyPublicPorts := make(map[zkr.PublicPortKey]zkr.PublicPort)
	emptyVHosts := make(map[zkr.VHostKey]zkr.VHost)
	t.cache.UpdateRegistry("service1", t.getExpectedPortsForService1(), emptyVHosts)

	c.Assert(len(t.cache.registry), Equals, 2)

	// "service1" should be match the values specified by the UpdateRegistry call
	svc1 := t.cache.GetRegistryForService( "service1")
	c.Assert(svc1.ServiceID, Equals, "service1")
	t.assertPortMapsEqual(c,  svc1.PublicPorts, t.getExpectedPortsForService1())
	t.assertVHostMapsEqual(c, svc1.VHosts,      map[zkr.VHostKey]zkr.VHost{})

	// "service2" should be unchanged from what was added by BuildCache
	svc2 := t.cache.GetRegistryForService( "service2")
	c.Assert(svc2.ServiceID, Equals, "service2")
	t.assertPortMapsEqual(c,  svc2.PublicPorts, t.getExpectedPortsForService2())
	t.assertVHostMapsEqual(c, svc2.VHosts,      t.getExpectedVHostsForService2())

	t.cache.UpdateRegistry("service3", emptyPublicPorts, t.getExpectedVHostsForService3())

	c.Assert(len(t.cache.registry), Equals, 3)

	svc3 := t.cache.GetRegistryForService( "service3")
	c.Assert(svc3.ServiceID, Equals, "service3")
	t.assertPortMapsEqual(c,  svc3.PublicPorts, map[zkr.PublicPortKey]zkr.PublicPort{})
	t.assertVHostMapsEqual(c, svc3.VHosts,      t.getExpectedVHostsForService3())
}

func (t *ServiceRegistryCacheTest) Test_BuildSyncRequest_NoEndpoints(c *C) {
	svc := service.Service{
		ID: "service1",
	}
	result := t.cache.BuildSyncRequest("tenantID", &svc)

	c.Assert(len(result.PortsToDelete), Equals, 0)
	c.Assert(len(result.PortsToPublish), Equals, 0)
	c.Assert(len(result.VHostsToDelete), Equals, 0)
	c.Assert(len(result.VHostsToPublish), Equals, 0)

	// Verify that the cache is unchanged by the BuildSyncRequest
	c.Assert(len(t.cache.registry), Equals, 0)
}

// Verify that the new, enabled endpoints are added to the request object
func (t *ServiceRegistryCacheTest) Test_BuildSyncRequest_AddsEndpoints(c *C) {
	svc := t.getTestService()

	tenantID := "expectedTenantID"
	result := t.cache.BuildSyncRequest(tenantID, &svc)

	c.Assert(len(result.PortsToDelete), Equals, 0)
	c.Assert(len(result.PortsToPublish), Equals, 1)
	c.Assert(len(result.VHostsToDelete), Equals, 0)
	c.Assert(len(result.VHostsToPublish), Equals, 1)

	portKey := zkr.PublicPortKey{
		HostID:      "master",
		PortAddress: svc.Endpoints[0].PortList[0].PortAddr,
	}
	port := zkr.PublicPort{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[0].Application,
		Protocol:    svc.Endpoints[0].PortList[0].Protocol,
		UseTLS:      svc.Endpoints[0].PortList[0].UseTLS,
	}
	expectedPorts := map[zkr.PublicPortKey]zkr.PublicPort{}
	expectedPorts[portKey] = port
	t.assertPortMapsEqual(c, result.PortsToPublish, expectedPorts)

	vhostKey := zkr.VHostKey{
		HostID:    "master",
		Subdomain: svc.Endpoints[1].VHostList[0].Name,
	}
	vhost := zkr.VHost{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[1].Application,
	}
	expectedVHosts := map[zkr.VHostKey]zkr.VHost{}
	expectedVHosts[vhostKey] = vhost

	t.assertVHostMapsEqual(c, result.VHostsToPublish, expectedVHosts)
}

// Verify that the cached endpoints flagged for removal if all endpoints are disabled
func (t *ServiceRegistryCacheTest) Test_BuildSyncRequest_EndpointsDisabled(c *C) {
	// Based on the test service, seed the cache with some initial values
	tenantID := "expectedTenantID"
	svc := t.getTestService()
	portKey := zkr.PublicPortKey{
		HostID:      "master",
		PortAddress: svc.Endpoints[0].PortList[0].PortAddr,
	}
	port := zkr.PublicPort{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[0].Application,
		Protocol:    svc.Endpoints[0].PortList[0].Protocol,
		UseTLS:      svc.Endpoints[0].PortList[0].UseTLS,
	}
	initialPorts := map[zkr.PublicPortKey]zkr.PublicPort{}
	initialPorts[portKey] = port

	vhostKey := zkr.VHostKey{
		HostID:    "master",
		Subdomain: svc.Endpoints[1].VHostList[0].Name,
	}
	vhost := zkr.VHost{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[1].Application,
	}
	initialVHosts := map[zkr.VHostKey]zkr.VHost{}
	initialVHosts[vhostKey] = vhost

	// Load the initial values into the cache
	t.cache.BuildCache(initialPorts, initialVHosts)

	// Now simulate disabling all of the endpoints
	svc = t.getTestService()
	svc.Endpoints[0].PortList[0].Enabled = false
	svc.Endpoints[1].VHostList[0].Enabled = false

	result := t.cache.BuildSyncRequest(tenantID, &svc)

	// Verify that the previously cached values were flagged for removal
	c.Assert(len(result.PortsToDelete), Equals, 1)
	c.Assert(len(result.PortsToPublish), Equals, 0)
	c.Assert(len(result.VHostsToDelete), Equals, 1)
	c.Assert(len(result.VHostsToPublish), Equals, 0)

	for _, key := range result.PortsToDelete {
		_, ok := initialPorts[key]
		c.Assert(ok, Equals, true)
	}
	for _, key := range result.VHostsToDelete {
		_, ok := initialVHosts[key]
		c.Assert(ok, Equals, true)
	}
}

func (t *ServiceRegistryCacheTest) Test_BuildSyncRequest_ReplaceAllEndpoints(c *C) {
	// Based on the test service, seed the cache with some initial values
	tenantID := "expectedTenantID"
	svc := t.getTestService()
	portKey := zkr.PublicPortKey{
		HostID:      "master",
		PortAddress: svc.Endpoints[0].PortList[0].PortAddr,
	}
	port := zkr.PublicPort{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[0].Application,
		Protocol:    svc.Endpoints[0].PortList[0].Protocol,
		UseTLS:      svc.Endpoints[0].PortList[0].UseTLS,
	}
	initialPorts := map[zkr.PublicPortKey]zkr.PublicPort{}
	initialPorts[portKey] = port

	vhostKey := zkr.VHostKey{
		HostID:    "master",
		Subdomain: svc.Endpoints[1].VHostList[0].Name,
	}
	vhost := zkr.VHost{
		TenantID:    tenantID,
		ServiceID:   svc.ID,
		Application: svc.Endpoints[1].Application,
	}
	initialVHosts := map[zkr.VHostKey]zkr.VHost{}
	initialVHosts[vhostKey] = vhost

	// Load the initial values into the cache
	t.cache.BuildCache(initialPorts, initialVHosts)

	// Now simulate modifying the key values for all of the endpoints (e.g. a service edit or migration might do this)
	svc = t.getTestService()
	svc.Endpoints[0].PortList[0].PortAddr = ":9999"
	svc.Endpoints[1].VHostList[0].Name = "somethingCompletelyDifferent"

	result := t.cache.BuildSyncRequest(tenantID, &svc)

	// Verify  the old keys are flagged for removal
	c.Assert(len(result.PortsToDelete), Equals, 1)
	for _, key := range result.PortsToDelete {
		_, ok := initialPorts[key]
		c.Assert(ok, Equals, true)
	}

	c.Assert(len(result.VHostsToDelete), Equals, 1)
	for _, key := range result.VHostsToDelete {
		_, ok := initialVHosts[key]
		c.Assert(ok, Equals, true)
	}

	// Verify the modified values are in the to-be-published lists
	c.Assert(len(result.PortsToPublish), Equals, 1)
	portKey.PortAddress = svc.Endpoints[0].PortList[0].PortAddr
	expectedPorts := map[zkr.PublicPortKey]zkr.PublicPort{}
	expectedPorts[portKey] = port
	t.assertPortMapsEqual(c, result.PortsToPublish, expectedPorts)

	c.Assert(len(result.VHostsToPublish), Equals, 1)
	vhostKey.Subdomain = svc.Endpoints[1].VHostList[0].Name
	expectedVHosts := map[zkr.VHostKey]zkr.VHost{}
	expectedVHosts[vhostKey] = vhost
	t.assertVHostMapsEqual(c, result.VHostsToPublish, expectedVHosts)
}

func (t *ServiceRegistryCacheTest) getExpectedPortsForService1() (expected map[zkr.PublicPortKey]zkr.PublicPort) {
	portKey1 := zkr.PublicPortKey{
		HostID:      "host1",
		PortAddress: ":1281",
	}
	port1 := zkr.PublicPort{
		ServiceID: "service1",
		Protocol:  "tcp",
	}
	portKey2 := zkr.PublicPortKey{
		HostID:      "host1",
		PortAddress: ":1282",
	}
	port2 := zkr.PublicPort{
		ServiceID: "service1",
		Protocol:  "http",
	}
	expected = map[zkr.PublicPortKey]zkr.PublicPort{}
	expected[portKey1] = port1
	expected[portKey2] = port2
	return expected
}
func (t *ServiceRegistryCacheTest) getExpectedPortsForService2() (expected map[zkr.PublicPortKey]zkr.PublicPort) {

	portKey3 := zkr.PublicPortKey{
		HostID:      "host2",
		PortAddress: ":1282",
	}
	port3 := zkr.PublicPort{
		ServiceID: "service2",
		Protocol:  "http",
	}
	expected = map[zkr.PublicPortKey]zkr.PublicPort{}
	expected[portKey3] = port3
	return expected
}

func (t *ServiceRegistryCacheTest) getExpectedVHostsForService2() (expected map[zkr.VHostKey]zkr.VHost) {
	vhostKey1 := zkr.VHostKey{
		HostID:      "host1",
		Subdomain:   "domain1",
	}
	vhost1 := zkr.VHost{
		ServiceID:   "service2",
		Application: "app1",
	}
	vhostKey2 := zkr.VHostKey{
		HostID:      "host1",
		Subdomain:   "domain2",
	}
	vhost2 := zkr.VHost{
		ServiceID:   "service2",
		Application: "app2",
	}

	expected = map[zkr.VHostKey]zkr.VHost{}
	expected[vhostKey1] = vhost1
	expected[vhostKey2] = vhost2
	return expected
}

func (t *ServiceRegistryCacheTest) getExpectedVHostsForService3() (expected map[zkr.VHostKey]zkr.VHost) {
	vhostKey3 := zkr.VHostKey{
		HostID:      "host2",
		Subdomain:   "domain3",
	}
	vhost3 := zkr.VHost{
		ServiceID:   "service3",
		Application: "app3",
	}

	expected = map[zkr.VHostKey]zkr.VHost{}
	expected[vhostKey3] = vhost3
	return expected
}

func (t *ServiceRegistryCacheTest) getTestService() service.Service {
	return service.Service{
		ID: "service1",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				Name:        "ep1",
				Application: "app1",
				PortList:    []servicedefinition.Port{
					servicedefinition.Port{
						Enabled:  true,
						PortAddr: ":1281",
						Protocol: "http",
						UseTLS:   true,
					},
				},
			},
			service.ServiceEndpoint{
				Name:        "ep2",
				Application: "app2",
				VHostList:    []servicedefinition.VHost{
					servicedefinition.VHost{
						Enabled:  true,
						Name:     "vhost1",
					},
				},
			},
		},
	}
}
func (t *ServiceRegistryCacheTest) assertPortMapsEqual(c *C, actual, expected map[zkr.PublicPortKey]zkr.PublicPort) {
	c.Assert(len(actual), Equals, len(expected))
	for key, value := range expected {
		actualValue, ok := actual[key]
		c.Assert(ok, Equals, true)
		c.Assert(actualValue, Equals, value)
	}
}

func (t *ServiceRegistryCacheTest) assertVHostMapsEqual(c *C, actual, expected map[zkr.VHostKey]zkr.VHost) {
	c.Assert(len(actual), Equals, len(expected))
	for key, value := range expected {
		actualValue, ok := actual[key]
		c.Assert(ok, Equals, true)
		c.Assert(actualValue, Equals, value)
	}
}

