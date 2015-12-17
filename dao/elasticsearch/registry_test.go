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

// +build integration,!quick

//This test is an integration test and need zookeeper which is why it isn't in the zzk/registry package
package elasticsearch

import (
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"

	"time"
)

func (dt *DaoTest) TestDao_PublicEndpointRegistryCreate(t *C) {

	_, err := registry.PublicEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	//test idempotence
	_, err = registry.PublicEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)
}

func (dt *DaoTest) TestDao_PublicEndpointRegistrySet(t *C) {

	pepr, err := registry.PublicEndpointRegistry(dt.zkConn)
	t.Assert(err, IsNil)

	// TODO: add tests for ephemeral nodes and remove pepr.SetEphemeral(false)
	pepr.SetEphemeral(false)

	pep := registry.PublicEndpoint{}
	pep.EndpointName = "epn_test"
	pep.ServiceID = "svc_id"
	pep.HostIP = "testip"
	path, err := pepr.SetItem(dt.zkConn, "testKey", pep)
	t.Assert(err, IsNil)
	t.Assert(path, Not(Equals), 0)

	var newPep *registry.PublicEndpoint
	newPep, err = pepr.GetItem(dt.zkConn, path)
	t.Assert(err, IsNil)
	t.Assert(pep, NotNil)
	//remove version for equals
	newPep.SetVersion(nil)
	t.Assert(pep, Equals, *newPep)

	//test double add
	glog.Infof("%+v", pep)
	path, err = pepr.SetItem(dt.zkConn, "testKey", pep)
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

	aep := applicationendpoint.ApplicationEndpoint{
		ServiceID:     "epn_service",
		ContainerID:   "epn_container",
		ContainerIP:   "192.168.0.1",
		ContainerPort: 54321,
		ProxyPort:     54321,
		HostID:        "epn_host1",
		HostIP:        "192.168.0.2",
		HostPort:      12345,
		Protocol:      "epn_tcp",
	}

	epn1 := registry.EndpointNode{
		ApplicationEndpoint: aep,
		TenantID:            "epn_tenant",
		EndpointID:          "epn_endpoint",
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

			epr.WatchTenantEndpoint(dt.zkConn, parentKey, changeEvt, errEvt, make(chan interface{}))
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

		// case 2: update an item (should not receive an event)
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
