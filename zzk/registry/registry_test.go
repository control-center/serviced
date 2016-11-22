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

// +build integration,!quick

package registry_test

import (
	//"path"

	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/registry"
	. "gopkg.in/check.v1"
)

// If the request is empty, then the ZK namespace should remain unchanged
func (t *ZZKTest) TestSyncRegistry_NOOP(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	request := ServiceRegistrySyncRequest{}

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	ok, err := conn.Exists("/net/pub")
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)

	ok, err = conn.Exists("/net/vhosts")
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}

func (t *ZZKTest) TestSyncRegistry_ForPublicPorts(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// Verify adding a public port
	portKey := PublicPortKey{HostID: "master", PortAddress: ":2181"}
	port := PublicPort{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Protocol:    "http",
		UseTLS:      false,
	}
	request := ServiceRegistrySyncRequest{
		ServiceID:      "serviceid1",
		PortsToPublish: map[PublicPortKey]PublicPort{portKey: port},
	}

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	actualPublicPort := &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualPublicPort)
	c.Assert(err, IsNil)
	actualPublicPort.SetVersion(nil)
	c.Check(*actualPublicPort, DeepEquals, port)

	// Verify updating a public port
	port.UseTLS = false
	port.Protocol = "tcp"
	request.PortsToPublish[portKey] = port

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	actualPublicPort = &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualPublicPort)
	c.Assert(err, IsNil)
	actualPublicPort.SetVersion(nil)
	c.Check(*actualPublicPort, DeepEquals, port)

	// Verify deleting a public port
	delete(request.PortsToPublish, portKey)
	request.PortsToDelete = append(request.PortsToDelete, portKey)

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	ok, err := conn.Exists("/net/pub/master/:2181")
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}

func (t *ZZKTest) TestSyncRegistry_ForVHosts(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// Verify adding a vhost
	vhostKey := VHostKey{HostID: "master", Subdomain: "subdomain"}
	vhost := VHost{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
	}
	request := ServiceRegistrySyncRequest{
		ServiceID:      "serviceid1",
		VHostsToPublish: map[VHostKey]VHost{vhostKey: vhost},
	}

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	actualVHost := &VHost{}
	err = conn.Get("/net/vhost/master/subdomain", actualVHost)
	c.Assert(err, IsNil)
	actualVHost.SetVersion(nil)
	c.Check(*actualVHost, DeepEquals, vhost)

	// Verify updating a vhost
	vhost.Application = "someOtherApp"
	request.VHostsToPublish[vhostKey] = vhost

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	actualVHost = &VHost{}
	err = conn.Get("/net/vhost/master/subdomain", actualVHost)
	c.Assert(err, IsNil)
	actualVHost.SetVersion(nil)
	c.Check(*actualVHost, DeepEquals, vhost)

	// Verify deleting a vhost
	delete(request.VHostsToPublish, vhostKey)
	request.VHostsToDelete = append(request.VHostsToDelete, vhostKey)

	err = SyncServiceRegistry(conn, request)
	c.Assert(err, IsNil)

	ok, err := conn.Exists("/net/vhost/master/subdomain")
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}
