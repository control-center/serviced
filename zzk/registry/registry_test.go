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

	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/registry"
	. "gopkg.in/check.v1"
)

func (t *ZZKTest) TestSyncRegistry(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// test create
	pub1Key := PublicPortKey{HostID: "master", PortAddress: ":2181"}
	pub1 := PublicPort{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Enabled:     true,
		Protocol:    "http",
		UseTLS:      false,
	}
	pubs1 := map[PublicPortKey]PublicPort{pub1Key: pub1}

	vhost1Key := VHostKey{HostID: "master", Subdomain: "test1"}
	vhost1 := VHost{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Enabled:     true,
	}
	vhosts1 := map[VHostKey]VHost{vhost1Key: vhost1}

	err = SyncServiceRegistry(conn, "serviceid1", pubs1, vhosts1)
	c.Assert(err, IsNil)

	actualpub := &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualpub)
	c.Assert(err, IsNil)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub1)

	actualvhost := &VHost{}
	err = conn.Get("/net/vhost/master/test1", actualvhost)
	c.Assert(err, IsNil)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost1)

	pub2Key := PublicPortKey{HostID: "master", PortAddress: ":22181"}
	pub2 := PublicPort{
		TenantID:    "tenantid",
		Application: "app2",
		ServiceID:   "serviceid2",
		Enabled:     true,
		Protocol:    "https",
		UseTLS:      true,
	}
	pubs2 := map[PublicPortKey]PublicPort{pub2Key: pub2}

	vhost2Key := VHostKey{HostID: "master", Subdomain: "test2"}
	vhost2 := VHost{
		TenantID:    "tenantid",
		Application: "app2",
		ServiceID:   "serviceid2",
		Enabled:     true,
	}
	vhosts2 := map[VHostKey]VHost{vhost2Key: vhost2}

	err = SyncServiceRegistry(conn, "serviceid2", pubs2, vhosts2)
	c.Assert(err, IsNil)

	actualpub = &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualpub)
	c.Assert(err, IsNil)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub1)

	actualvhost = &VHost{}
	err = conn.Get("/net/vhost/master/test1", actualvhost)
	c.Assert(err, IsNil)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost1)

	actualpub = &PublicPort{}
	err = conn.Get("/net/pub/master/:22181", actualpub)
	c.Assert(err, IsNil)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub2)

	actualvhost = &VHost{}
	err = conn.Get("/net/vhost/master/test2", actualvhost)
	c.Assert(err, IsNil)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost2)

	// test update
	pub1Key = PublicPortKey{HostID: "master", PortAddress: ":2181"}
	pub1 = PublicPort{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Enabled:     false,
		Protocol:    "",
		UseTLS:      true,
	}
	pubs1 = map[PublicPortKey]PublicPort{pub1Key: pub1}

	vhost1Key = VHostKey{HostID: "master", Subdomain: "test1"}
	vhost1 = VHost{
		TenantID:    "tenantid",
		Application: "app1",
		ServiceID:   "serviceid1",
		Enabled:     false,
	}
	vhosts1 = map[VHostKey]VHost{vhost1Key: vhost1}

	err = SyncServiceRegistry(conn, "serviceid1", pubs1, vhosts1)
	c.Assert(err, IsNil)

	actualpub = &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualpub)
	c.Assert(err, IsNil)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub1)

	actualvhost = &VHost{}
	err = conn.Get("/net/vhost/master/test1", actualvhost)
	c.Assert(err, IsNil)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost1)

	actualpub = &PublicPort{}
	err = conn.Get("/net/pub/master/:22181", actualpub)
	c.Assert(err, IsNil)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub2)

	actualvhost = &VHost{}
	err = conn.Get("/net/vhost/master/test2", actualvhost)
	c.Assert(err, IsNil)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost2)

	// test delete
	err = SyncServiceRegistry(conn, "serviceid2", make(map[PublicPortKey]PublicPort), make(map[VHostKey]VHost))
	c.Assert(err, IsNil)

	actualpub = &PublicPort{}
	err = conn.Get("/net/pub/master/:2181", actualpub)
	actualpub.SetVersion(nil)
	c.Check(*actualpub, DeepEquals, pub1)

	actualvhost = &VHost{}
	err = conn.Get("/net/vhost/master/test1", actualvhost)
	actualvhost.SetVersion(nil)
	c.Check(*actualvhost, DeepEquals, vhost1)

	ok, err := conn.Exists(path.Join("/net/pub", pub2Key.HostID, pub2Key.PortAddress))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)

	ok, err = conn.Exists(path.Join("/net/vhost", vhost2Key.HostID, vhost2Key.Subdomain))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}
