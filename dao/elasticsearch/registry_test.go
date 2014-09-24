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

//This test is an integration test and need zookeeper which is why it isn't in the zzk/registry package
package elasticsearch

import (
	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"
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

func (dt *DaoTest) TestDao_VhostRegistrySet(t *C) {

	vr, err := registry.VHostRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	// TODO: add tests for ephemeral nodes and remove vr.SetEphemeral(false)
	vr.SetEphemeral(false)

	vep := registry.VhostEndpoint{}
	vep.EndpointName = "epn_test"
	vep.ServiceID = "svc_id"
	vep.HostIP = "testip"
	path, err := vr.SetItem(dt.zkConn, "testKey", vep)
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
	glog.Infof("%+v", vep)
	path, err = vr.SetItem(dt.zkConn, "testKey", vep)
	t.Assert(err, IsNil)
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

	epr.SetEphemeral(false)

	//test get and set
	verifySetGet := func(expected registry.EndpointNode) {
		glog.V(1).Infof("verifying set/get for expected %+v", expected)
		for ii := 0; ii < 3; ii++ {
			path, err := epr.SetItem(dt.zkConn, expected)
			glog.V(1).Infof("expected item[%s]: %+v", path, expected)
			t.Assert(err, IsNil)
			t.Assert(path, Not(Equals), 0)

			if epr.IsEphemeral() {
				var obtained *registry.EndpointNode
				obtained, err = epr.GetItem(dt.zkConn, path)
				glog.V(1).Infof("obtained item[%s]: %+v", path, expected)
				t.Assert(err, IsNil)
				t.Assert(obtained, NotNil)
				//remove version for equals
				expected.SetVersion(nil)
				obtained.SetVersion(nil)
				t.Assert(expected, Equals, *obtained)
			}
		}
	}

	aep := dao.ApplicationEndpoint{
		ServiceID:     "epn_service",
		ContainerIP:   "192.168.0.1",
		ContainerPort: 54321,
		ProxyPort:     54321,
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

	verifySetGet(epn1)

	epn2 := epn1
	epn2.EndpointID = "epn_.*"
	epn2.ContainerID = "epn_container2"
	verifySetGet(epn2)

	// test remove
	verifyRemove := func(expected registry.EndpointNode) {
		err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
		t.Assert(err, IsNil)
	}
	verifyRemove(epn1)
	verifyRemove(epn2)

	//test watch tenant endpoint
	verifyWatch := func(expected registry.EndpointNode) {
		numEndpoints := 0
		var obtained *registry.EndpointNode
		go func() {
			errorWatcher := func(path string, err error) {}
			countEvents := func(conn client.Connection, parentPath string, nodeIDs ...string) {
				numEndpoints++
				glog.Infof("seeing event %d parentPath:%s nodeIDs:%v", numEndpoints, parentPath, nodeIDs)

				if len(nodeIDs) > 0 {
					obtained, err = epr.GetItem(dt.zkConn, fmt.Sprintf("%s/%s", parentPath, nodeIDs[0]))
					t.Assert(err, IsNil)
					t.Assert(obtained, NotNil)
				}
			}

			tenantEndpointKey := registry.TenantEndpointKey(expected.TenantID, expected.EndpointID)
			glog.Infof("watching tenant endpoint %s", tenantEndpointKey)
			err = epr.WatchTenantEndpoint(dt.zkConn, tenantEndpointKey, countEvents, errorWatcher)
			t.Assert(err, IsNil)
		}()

		time.Sleep(2 * time.Second)

		const numEndpointsExpected int = 3
		for i := 0; i < numEndpointsExpected; i++ {
			glog.Infof("SetItem %+v", expected)
			expected.ContainerID = fmt.Sprintf("epn_container_%d", i)
			_, err = epr.SetItem(dt.zkConn, expected)
			t.Assert(err, IsNil)
			time.Sleep(1 * time.Second)
		}

		time.Sleep(2 * time.Second)
		//remove version for equals
		expected.SetVersion(nil)
		obtained.SetVersion(nil)
		t.Assert(expected, Equals, *obtained)

		// remove items
		for i := 0; i < numEndpointsExpected; i++ {
			glog.Infof("RemoveItem %+v", expected)
			expected.ContainerID = fmt.Sprintf("epn_container_%d", i)
			err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
			t.Assert(err, IsNil)
			time.Sleep(1 * time.Second)
		}

		err = epr.RemoveTenantEndpointKey(dt.zkConn, expected.TenantID, expected.EndpointID)
		t.Assert(err, IsNil)

		glog.Warning("TODO - deficiency of epr.WatchTenantEndpoint - notice how countEvents is not called after RemoveTenantEndpointKey")
		for i := 0; i < numEndpointsExpected; i++ {
			glog.Infof("SetItem %+v", expected)
			expected.ContainerID = fmt.Sprintf("epn_container_%d", i)
			_, err = epr.SetItem(dt.zkConn, expected)
			t.Assert(err, IsNil)
			time.Sleep(1 * time.Second)
		}
	}

	epn3 := epn1
	verifyWatch(epn3)

	// make sure that watching non-existent path does not result in panic
	epn4 := registry.EndpointNode{
		ApplicationEndpoint: aep,
		TenantID:            "epn_tenant4",
		EndpointID:          "epn_endpoint4",
		HostID:              "epn_host4",
		ContainerID:         "epn_container4",
	}
	verifyWatchNonExistentKey := func(expected registry.EndpointNode) {
		var obtained *registry.EndpointNode
		errorWatcher := func(path string, err error) {}
		showEvents := func(conn client.Connection, parentPath string, nodeIDs ...string) {
			glog.Infof("seeing event parentPath:%s nodeIDs:%v", parentPath, nodeIDs)

			if len(nodeIDs) > 0 {
				obtained, err = epr.GetItem(dt.zkConn, fmt.Sprintf("%s/%s", parentPath, nodeIDs[0]))
				t.Assert(err, IsNil)
				t.Assert(obtained, NotNil)
			}
		}

		tenantEndpointKey := registry.TenantEndpointKey(expected.TenantID, expected.EndpointID)
		glog.Infof("watching tenant endpoint %s", tenantEndpointKey)
		err = epr.WatchTenantEndpoint(dt.zkConn, tenantEndpointKey, showEvents, errorWatcher)
		t.Assert(err, NotNil)
		t.Assert(err, Equals, client.ErrNoNode)
	}
	verifyWatchNonExistentKey(epn4)
}
