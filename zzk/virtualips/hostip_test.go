// Copyright 2017 The Serviced Authors.
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

package virtualips_test

import (
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/virtualips"
	. "gopkg.in/check.v1"
)

type ZZKTest struct {
	zzk.ZZKTestSuite
}

func (t *ZZKTest) TestParseIPID(c *C) {
	// invalid id
	hostID, ipAddr, err := ParseIPID("bdlfdshfdsl")
	c.Assert(err, Equals, ErrInvalidIPID)
	c.Assert(hostID, Equals, "")
	c.Assert(ipAddr, Equals, "")

	// valid id
	hostID, ipAddr, err = ParseIPID("123abcd-1.2.3.4")
	c.Assert(err, IsNil)
	c.Assert(hostID, Equals, "123abcd")
	c.Assert(ipAddr, Equals, "1.2.3.4")
}

func (t *ZZKTest) TestCRUDIP(c *C) {
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// add a pool
	err = conn.CreateDir("/pools/poolid")
	c.Assert(err, IsNil)

	// add a host
	err = conn.CreateDir("/pools/poolid/hosts/hostid")
	c.Assert(err, IsNil)

	req := IPRequest{
		PoolID:    "poolid",
		HostID:    "hostid",
		IPAddress: "1.2.3.4",
	}

	// get ip that doesn't exist
	ip, err := GetIP(conn, req)
	// TODO: error
	c.Assert(err, NotNil)
	c.Assert(ip, IsNil)

	// update ip that doesn't exist (no commit)
	err = UpdateIP(conn, req, func(ip *IP) bool {
		ip.Netmask = "0.0.0.0"
		ip.BindInterface = "eth1"
		return false
	})
	// TODO: error
	c.Assert(err, NotNil)

	// update ip that doesn't exist (commit)
	err = UpdateIP(conn, req, func(ip *IP) bool {
		ip.Netmask = "0.0.0.0"
		ip.BindInterface = "eth1"
		return true
	})
	// TODO: error
	c.Assert(err, NotNil)

	// create ip
	err = CreateIP(conn, req, "255.255.255.0", "eth0")
	c.Assert(err, IsNil)
	ok, err := conn.Exists("/pools/poolid/ips/hostid-1.2.3.4")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)
	ok, err = conn.Exists("/pools/poolid/hosts/hostid/ips/hostid-1.2.3.4")
	c.Assert(err, IsNil)
	c.Assert(ok, Equals, true)

	// create duplicate ip
	err = CreateIP(conn, req, "0.0.0.0", "eth1")
	// TODO: error
	c.Assert(err, NotNil)

	// ip exists
	ip, err = GetIP(conn, req)
	c.Assert(err, IsNil)
	c.Check(ip.Netmask, Equals, "255.255.255.0")
	c.Check(ip.BindInterface, Equals, "eth0")
	c.Check(ip.HostID, Equals, "hostid")
	c.Check(ip.IPAddress, Equals, "1.2.3.4")

	// update ip (no commit)
	err = UpdateIP(conn, req, func(ip *IP) bool {
		ip.Netmask = "0.0.0.0"
		ip.BindInterface = "eth1"
		return false
	})
	c.Assert(err, IsNil)
	ip, err = GetIP(conn, req)
	c.Assert(err, IsNil)
	c.Check(ip.Netmask, Equals, "255.255.255.0")
	c.Check(ip.BindInterface, Equals, "eth0")
	c.Check(ip.HostID, Equals, "hostid")
	c.Check(ip.IPAddress, Equals, "1.2.3.4")

	// update ip (commit)
	err = UpdateIP(conn, req, func(ip *IP) bool {
		ip.Netmask = "0.0.0.0"
		ip.BindInterface = "eth1"
		return true
	})
	c.Assert(err, IsNil)
	ip, err = GetIP(conn, req)
	c.Assert(err, IsNil)
	c.Check(ip.Netmask, Equals, "0.0.0.0")
	c.Check(ip.BindInterface, Equals, "eth1")
	c.Check(ip.HostID, Equals, "hostid")
	c.Check(ip.IPAddress, Equals, "1.2.3.4")
}
