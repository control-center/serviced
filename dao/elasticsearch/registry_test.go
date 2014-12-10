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
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"

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
	t.Logf("Creating a new node: %+v", epn1)

	// verify that only one node gets set per host-container combo for the
	// tenant-endpoint key (regardless of whether the node is ephemeral)
	func(expected registry.EndpointNode) {
		// case 1: add a single non-ephemeral node
		epr.SetEphemeral(false)
		p, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(p, Not(Equals), "")
		defer dt.zkConn.Delete(path.Dir(p))

		actual, err := epr.GetItem(dt.zkConn, p)
		t.Assert(err, IsNil)
		t.Assert(actual, NotNil)

		expected.SetVersion(nil)
		actual.SetVersion(nil)
		t.Assert(expected, Equals, *actual)

		// case 2: update the node
		expected.ContainerIP = "192.168.1.1"
		path2, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(path2, Equals, p)

		actual, err = epr.GetItem(dt.zkConn, path2)
		t.Assert(err, IsNil)
		t.Assert(actual, NotNil)

		expected.SetVersion(nil)
		actual.SetVersion(nil)
		t.Assert(expected, Equals, *actual)

		// case 3: add the same node ephemerally
		epr.SetEphemeral(true)
		expected.ContainerIP = "192.168.2.2"
		path3, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(path3, Equals, p)

		actual, err = epr.GetItem(dt.zkConn, path3)
		t.Assert(err, IsNil)
		t.Assert(actual, NotNil)

		expected.SetVersion(nil)
		actual.SetVersion(nil)
		t.Assert(expected, Equals, *actual)

		// case 4: add a new ephemeral node
		expected.ContainerID = "epn_container_E"
		path4, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(path4, Not(Equals), p)

		actual, err = epr.GetItem(dt.zkConn, path4)
		t.Assert(err, IsNil)
		t.Assert(actual, NotNil)

		expected.SetVersion(nil)
		actual.SetVersion(nil)
		t.Assert(expected, Equals, *actual)

		// case 5: add the same node non-ephemerally
		/*
			epr.SetEphemeral(false)
			expected.ContainerIP = "192.168.3.3"
			path5, err := epr.SetItem(dt.zkConn, expected)
			t.Assert(err, IsNil)
			t.Assert(path5, Equals, path4)

			actual, err = epr.GetItem(dt.zkConn, path5)
			t.Assert(err, IsNil)
			t.Assert(actual, NotNil)

			expected.SetVersion(nil)
			actual.SetVersion(nil)
			t.Assert(expected, Equals, *actual)
		*/
	}(epn1)

	t.Logf("Testing removal of endpoint node %+v", epn1)
	func(expected registry.EndpointNode) {
		// case 1: remove non-ephemeral node
		epr.SetEphemeral(false)
		p, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(p, Not(Equals), "")
		defer dt.zkConn.Delete(path.Dir(p))

		err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
		t.Assert(err, IsNil)
		exists, _ := dt.zkConn.Exists(p)
		t.Assert(exists, Equals, false)

		// case 2: remove non-ephemeral node as ephemeral
		/*
			p, err = epr.SetItem(dt.zkConn, expected)
			t.Assert(err, IsNil)
			t.Assert(p, Not(Equals), "")

			epr.SetEphemeral(true)
			err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
			if err != nil {
				defer dt.zkConn.Delete(p)
			}
			t.Assert(err, IsNil)
			exists, _ = dt.zkConn.Exists(p)
			t.Assert(exists, Equals, false)
		*/

		// case 3: remove ephemeral node
		p, err = epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(p, Not(Equals), "")

		err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
		t.Assert(err, IsNil)
		exists, _ = dt.zkConn.Exists(p)
		t.Assert(exists, Equals, false)

		// case 4: remove ephemeral node as non-ephemeral
		p, err = epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)
		t.Assert(p, Not(Equals), "")

		epr.SetEphemeral(false)
		err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
		if err != nil {
			defer dt.zkConn.Delete(p)
		}
		t.Assert(err, IsNil)
		exists, _ = dt.zkConn.Exists(p)
		t.Assert(exists, Equals, false)
	}(epn1)

	t.Logf("Testing endpoint watcher")
	func(expected registry.EndpointNode) {
		errC := make(chan error)
		eventC := make(chan int)

		// case 0: no parent
		go func() {
			parentKey := registry.TenantEndpointKey("bad_tenant", "bad_endpoint")
			errC <- epr.WatchTenantEndpoint(dt.zkConn, parentKey, nil, nil)
		}()
		select {
		case err := <-errC:
			t.Assert(err, Equals, client.ErrNoNode)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint")
			// TODO: cancel listener here
		}

		t.Logf("Starting endpoint listener")
		go func() {
			changeEvt := func(conn client.Connection, parent string, nodeIDs ...string) {
				eventC <- len(nodeIDs)
			}

			errEvt := func(path string, err error) {
				errC <- err
			}

			parentKey := registry.TenantEndpointKey(expected.TenantID, expected.EndpointID)
			_, err := epr.EnsureKey(dt.zkConn, parentKey)
			t.Assert(err, IsNil)

			epr.WatchTenantEndpoint(dt.zkConn, parentKey, changeEvt, errEvt)
			close(errC)
		}()

		select {
		case count := <-eventC:
			t.Assert(count, Equals, 0)
		case err := <-errC:
			t.Fatalf("unexpected error running endpoint listener: %s", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint: %+v", expected)
			// TODO: cancel listener here
		}

		// case 1: add item
		p, err := epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)

		select {
		case count := <-eventC:
			t.Assert(count, Equals, 1)
		case err := <-errC:
			t.Fatalf("unexpected error running endpoint listener: %s", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint: %+v", expected)
			// TODO: cancel listener here
		}

		// case 2: update an item (should not receieve an event)
		expected.ContainerIP = "192.168.23.12"
		_, err = epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)

		// case 3: add another item
		expected.ContainerID = "test_container_2"
		_, err = epr.SetItem(dt.zkConn, expected)
		t.Assert(err, IsNil)

		select {
		case count := <-eventC:
			t.Assert(count, Equals, 2)
		case err := <-errC:
			t.Fatalf("unexpected error running endpoint listener: %s", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint: %+v", expected)
			// TODO: cancel listener here
		}

		// case 4: remove item
		err = epr.RemoveItem(dt.zkConn, expected.TenantID, expected.EndpointID, expected.HostID, expected.ContainerID)
		t.Assert(err, IsNil)

		select {
		case count := <-eventC:
			t.Assert(count, Equals, 1)
		case err := <-errC:
			t.Fatalf("unexpected error running endpoint listener: %s", err)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint: %+v", expected)
			// TODO: cancel listener here
		}

		// case 5: remove parent
		err = dt.zkConn.Delete(path.Dir(p))
		t.Assert(err, IsNil)

		select {
		case count := <-eventC:
			t.Assert(count, Equals, 0)
		case err := <-errC:
			t.Assert(err, Equals, client.ErrNoNode)
		case <-time.After(5 * time.Second):
			t.Fatalf("Timeout from WatchTenantEndpoint: %+v", expected)
			// TODO: cancel listener here
		}
	}(epn1)
}
