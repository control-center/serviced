// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

//This test is an integration test and need zookeeper which is why it isn't in the zzk/registry package
package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/coordinator/client"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/zzk/registry"
	. "gopkg.in/check.v1"

	"fmt"
	"time"
)

func (dt *DaoTest) TestDao_VhostRegistryCreate(t *C) {

	_, err := registry.VHostRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	//test idempotence
	_, err = registry.VHostRegistry(dt.zkConn)
	t.Assert(err, IsNil)
}

func (dt *DaoTest) TestDao_VhostRegistryAdd(t *C) {

	vr, err := registry.VHostRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	//	func (vr *VhostRegistry) AddItem(conn client.Connection, key string, node *VhostEndpoint) (string, error) {

	vep := registry.VhostEndpoint{}
	vep.EndpointName = "epn_test"
	vep.ServiceID = "svc_id"
	vep.HostIP = "testip"
	path, err := vr.AddItem(dt.zkConn, "testKey", vep)
	t.Assert(err, IsNil)
	t.Assert(path, Not(Equals), 0)

	var newVep *registry.VhostEndpoint
	newVep, err = vr.GetItem(dt.zkConn, path)
	t.Assert(err, IsNil)
	t.Assert(vep, NotNil)
	//remove version for equals
	newVep.SetVersion(nil)
	t.Assert(vep, Equals, *newVep)

	//test double add
	path, err = vr.AddItem(dt.zkConn, "testKey", vep)
	t.Assert(err, NotNil)
}

func (dt *DaoTest) TestDao_EndpointRegistryCreate(t *C) {

	_, err := registry.CreateEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	//test idempotence
	_, err = registry.CreateEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)
}

func (dt *DaoTest) TestDao_EndpointRegistrySet(t *C) {

	epr, err := registry.CreateEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	verifyGet := func(expected registry.EndpointNode) {
		glog.Infof("verifying set/get for expected %+v", expected)
		for ii := 0; ii < 3; ii++ {
			path, err := epr.SetItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID, expected)
			t.Assert(err, IsNil)
			t.Assert(path, Not(Equals), 0)

			var obtained *registry.EndpointNode
			obtained, err = epr.GetItem(dt.zkConn, path)
			t.Assert(err, IsNil)
			t.Assert(obtained, NotNil)
			//remove version for equals
			expected.SetVersion(nil)
			obtained.SetVersion(nil)
			t.Assert(expected, Equals, *obtained)
		}
	}

	aep := dao.ApplicationEndpoint{
		ServiceID:     "epn_service",
		ContainerIP:   "192.168.0.1",
		ContainerPort: 54321,
		HostIP:        "192.168.0.2",
		HostPort:      12345,
		Protocol:      "epn_tcp",
	}
	epn1 := registry.EndpointNode{
		ApplicationEndpoint: aep,
		TenantID:            "epn_tenant",
		EndpointID:          "epn_endpoint",
		HostID:              "epn_host1",
		ContainerID:         "epn_container",
	}

	verifyGet(epn1)
	epn2 := epn1
	epn2.EndpointID = "epn_.*"
	verifyGet(epn2)

	//test watch tenant endpoint
	numEndpoints := 0
	var epn4 *registry.EndpointNode
	go func() {
		errorWatcher := func(path string, err error) {}
		countEvents := func(conn client.Connection, parentPath string, nodeIDs ...string) {
			numEndpoints++
			glog.Infof("seeing event %d", numEndpoints)
			glog.Infof("  nodeIDs %v", nodeIDs)

			epn4, err = epr.GetItem(dt.zkConn, fmt.Sprintf("%s/%s", parentPath, nodeIDs[0]))
			t.Assert(err, IsNil)
			t.Assert(epn4, NotNil)
		}

		glog.Infof("watching tenant endpoint")
		err = epr.WatchTenantEndpoint(dt.zkConn, epn1.TenantID, epn1.EndpointID, countEvents, errorWatcher)
		t.Assert(err, IsNil)
	}()

	time.Sleep(2 * time.Second)

	const numEndpointsExpected int = 3
	epn3 := epn1
	for i := 0; i < numEndpointsExpected; i++ {
		glog.Infof("SetItem %+v", epn3)
		epn3.ContainerID = fmt.Sprintf("epn_container_%d", i)
		_, err = epr.SetItem(dt.zkConn, epn3.TenantID, epn3.EndpointID, epn3.HostID, epn3.ContainerID, epn3)
		t.Assert(err, IsNil)
		time.Sleep(1 * time.Second)
	}

	time.Sleep(2 * time.Second)
	//remove version for equals
	epn3.SetVersion(nil)
	epn4.SetVersion(nil)
	t.Assert(epn3, Equals, *epn4)
}
