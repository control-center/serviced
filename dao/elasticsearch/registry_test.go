// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

//This test is an integration test and need zookeeper which is why it isn't in the zzk/registry package
package elasticsearch

import (
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/zzk/registry"
	. "gopkg.in/check.v1"
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

func (dt *DaoTest) TestDao_EndpointRegistryAdd(t *C) {

	epr, err := registry.CreateEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)

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
		ContainerID:         "epn_container",
	}
	path, err := epr.AddItem(dt.zkConn, epn1.TenantID, epn1.EndpointID, epn1.ContainerID, &epn1)
	t.Assert(err, IsNil)
	t.Assert(path, Not(Equals), 0)

	var epn2 *registry.EndpointNode
	epn2, err = epr.GetItem(dt.zkConn, path)
	t.Assert(err, IsNil)
	t.Assert(epn2, NotNil)
	//remove version for equals
	epn1.SetVersion(nil)
	epn2.SetVersion(nil)
	t.Assert(epn1, Equals, *epn2)

	//test double add
	path, err = epr.AddItem(dt.zkConn, epn1.TenantID, epn1.EndpointID, epn1.ContainerID, &epn1)
	t.Assert(err, NotNil)
}
