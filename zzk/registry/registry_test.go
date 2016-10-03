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
	"path"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/registry"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestSyncRegistry(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// test create
	pub1aKey := PublicPortKey{HostID: "master", PortAddress: ":2181"}
	pub1a := PublicPort{
		TenantID:    "tenantid",
		Application: "app1a",
		ServiceID:   "serviceid1",
		Protocol:    "http",
		UseTLS:      false,
	}
	pubs1 := map[PublicPortKey]PublicPort{pub1aKey: pub1a}

	vhost1aKey := VHostKey{HostID: "master", Subdomain: "test1a"}
	vhost1a := VHost{
		TenantID:    "tenantid",
		Application: "app1a",
		ServiceID:   "serviceid1",
	}
	vhosts1 := map[VHostKey]VHost{vhost1aKey: vhost1a}

	c.Logf("Create initial registry entries")
	err = SyncServiceRegistry(conn, "serviceid1", copyPublicPortMap(pubs1), copyVHostMap(vhosts1))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)

	// test create/update for a service that's already cached
	pub1bKey := PublicPortKey{HostID: "master", PortAddress: ":2182"}
	pub1b := PublicPort{
		TenantID:    "tenantid",
		Application: "app1b",
		ServiceID:   "serviceid1",
		Protocol:    "http",
		UseTLS:      false,
	}
	pubs1 = map[PublicPortKey]PublicPort{pub1aKey: pub1a, pub1bKey: pub1b}

	vhost1bKey := VHostKey{HostID: "master", Subdomain: "test1b"}
	vhost1b := VHost{
		TenantID:    "tenantid",
		Application: "app1b",
		ServiceID:   "serviceid1",
	}
	vhosts1 = map[VHostKey]VHost{vhost1aKey: vhost1a, vhost1bKey: vhost1b}

	c.Logf("Update/add registry entries for an existing service")
	err = SyncServiceRegistry(conn, "serviceid1", copyPublicPortMap(pubs1), copyVHostMap(vhosts1))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)

	// Create additional entries for a new service, "serviceid2"
	pub2Key := PublicPortKey{HostID: "master", PortAddress: ":22181"}
	pub2 := PublicPort{
		TenantID:    "tenantid",
		Application: "app2",
		ServiceID:   "serviceid2",
		Protocol:    "https",
		UseTLS:      true,
	}
	pubs2 := map[PublicPortKey]PublicPort{pub2Key: pub2}

	vhost2Key := VHostKey{HostID: "master", Subdomain: "test2"}
	vhost2 := VHost{
		TenantID:    "tenantid",
		Application: "app2",
		ServiceID:   "serviceid2",
	}
	vhosts2 := map[VHostKey]VHost{vhost2Key: vhost2}

	c.Logf("Create additional registry entries")
	err = SyncServiceRegistry(conn, "serviceid2", copyPublicPortMap(pubs2), copyVHostMap(vhosts2))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)
	t.assertRegistryContents(c, conn, pubs2, vhosts2)

	// test update
	pub1aKey = PublicPortKey{HostID: "master", PortAddress: ":2181"}
	pub1a = PublicPort{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Protocol:    "",
		UseTLS:      true,
	}
	pubs1 = map[PublicPortKey]PublicPort{pub1aKey: pub1a}

	vhost1aKey = VHostKey{HostID: "master", Subdomain: "test1a"}
	vhost1a = VHost{
		TenantID:    "tenantid",
		Application: "app1a",
		ServiceID:   "serviceid1",
	}
	vhosts1 = map[VHostKey]VHost{vhost1aKey: vhost1a}

	c.Logf("Update selected registry entries")
	err = SyncServiceRegistry(conn, "serviceid1", copyPublicPortMap(pubs1), copyVHostMap(vhosts1))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)
	t.assertRegistryContents(c, conn, pubs2, vhosts2)

	// test delete some things for one service
	c.Logf("Delete some registry entries")
	err = SyncServiceRegistry(conn, "serviceid1", copyPublicPortMap(pubs1), copyVHostMap(vhosts1))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)		// '1a' entries for 'serviceid1' remain unchanged
	t.assertRegistryContents(c, conn, pubs2, vhosts2)		// all entries for 'serviceid2' remain unchanged

	// '1b' entries for 'serviceid1' were removed.
	ok, err := conn.Exists(path.Join("/net/pub", pub1bKey.HostID, pub1bKey.PortAddress))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)

	ok, err = conn.Exists(path.Join("/net/vhost", vhost1bKey.HostID, vhost1bKey.Subdomain))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)


	// test delete everything for one service
	c.Logf("Delete all registry entries")
	err = SyncServiceRegistry(conn, "serviceid2", make(map[PublicPortKey]PublicPort), make(map[VHostKey]VHost))
	c.Assert(err, IsNil)

	t.assertRegistryContents(c, conn, pubs1, vhosts1)		// entries for 'servicedid1' remain unchanged

	ok, err = conn.Exists(path.Join("/net/pub", pub2Key.HostID, pub2Key.PortAddress))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)

	ok, err = conn.Exists(path.Join("/net/vhost", vhost2Key.HostID, vhost2Key.Subdomain))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}

func (t *ZZKTest) assertRegistryContents(c *C, conn client.Connection, pubs map[PublicPortKey]PublicPort, vhosts map[VHostKey]VHost) {
	for portKey, expectedPort := range pubs {
		expectedPath := path.Join("/net/pub", portKey.HostID, portKey.PortAddress)
		c.Logf("\tChecking expected path %s", expectedPath)

		actualpub := &PublicPort{}
		err := conn.Get(expectedPath, actualpub)
		c.Assert(err, IsNil)
		actualpub.SetVersion(nil)
		c.Check(*actualpub, DeepEquals, expectedPort)
	}

	for vhostKey, expectedVhost := range vhosts {
		expectedPath :=  path.Join("/net/vhost", vhostKey.HostID, vhostKey.Subdomain)
		c.Logf("\tChecking expected path %s", expectedPath)

		actualvhost := &VHost{}
		err := conn.Get(expectedPath, actualvhost)
		c.Assert(err, IsNil)
		actualvhost.SetVersion(nil)
		c.Check(*actualvhost, DeepEquals, expectedVhost)
	}
}

// SyncServiceRegistry modifies the input map, so the test uses this method to keep a copy of the original inputs
func copyPublicPortMap(src map[PublicPortKey]PublicPort) map[PublicPortKey]PublicPort {
	target := make(map[PublicPortKey]PublicPort)
	for key, value := range src {
		target[key] = value
	}
	return target
}

// SyncServiceRegistry modifies the input map, so the test uses this method to keep a copy of the original inputs
func copyVHostMap(src map[VHostKey]VHost) map[VHostKey]VHost {
	target := make(map[VHostKey]VHost)
	for key, value := range src {
		target[key] = value
	}
	return target
}
