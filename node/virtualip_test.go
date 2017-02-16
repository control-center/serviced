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

//+build integration,root

package node_test

import (
	"os/exec"

	. "github.com/control-center/serviced/node"
	. "gopkg.in/check.v1"
)

const testDev = "test99"

var _ = Suite(&VIPManagerTestSuite{})

type VIPManagerTestSuite struct{}

func (t *VIPManagerTestSuite) SetUpSuite(c *C) {
	// destroy the dummy virtual interface
	cmd := exec.Command("ip", "link", "del", "dev", testDev)
	cmd.Run()

	// create a dummy virtual interface
	cmd = exec.Command("ip", "link", "add", "name", testDev, "type", "dummy")
	err := cmd.Run()
	c.Assert(err, IsNil)
}

func (t *VIPManagerTestSuite) TearDownSuite(c *C) {
	// destroy the dummy virtual interface
	cmd := exec.Command("ip", "link", "del", "dev", testDev)
	err := cmd.Run()
	c.Check(err, IsNil)
}

func (t *VIPManagerTestSuite) TestCRUD(c *C) {
	v := NewVirtualIPManager("cctest")

	// no ips added
	ips := v.GetAll()
	c.Check(ips, HasLen, 0)

	// ip not found
	ip := v.Find("100.12.17.1")
	c.Check(ip, IsNil)

	// create an ip
	err := v.Bind("100.12.17.1/16", testDev)
	c.Assert(err, IsNil)

	ip1 := &IP{Addr: "100.12.17.1/16", Device: testDev, Label: testDev + ":cctest0"}
	ips = v.GetAll()
	c.Check(ips, HasLen, 1)
	c.Check(ips, DeepEquals, []IP{*ip1})

	ip = v.Find("100.12.17.1")
	c.Check(ip, DeepEquals, ip1)

	// create another ip
	err = v.Bind("100.12.49.2/24", testDev)
	c.Assert(err, IsNil)

	ip2 := &IP{Addr: "100.12.49.2/24", Device: testDev, Label: testDev + ":cctest1"}
	ips = v.GetAll()
	c.Check(ips, HasLen, 2)
	c.Check(ips, DeepEquals, []IP{*ip1, *ip2})

	ip = v.Find("100.12.49.2")
	c.Check(ip, DeepEquals, ip2)

	// delete ips
	err = v.Release(ip1.Addr, ip1.Device)
	c.Assert(err, IsNil)
	ips = v.GetAll()
	c.Check(ips, HasLen, 1)
	c.Check(ips, DeepEquals, []IP{*ip2})

	ip = v.Find("100.12.17.1")
	c.Check(ip, IsNil)

	err = v.Release(ip2.Addr, ip2.Device)
	c.Assert(err, IsNil)
	ips = v.GetAll()
	c.Check(ips, HasLen, 0)

	ip = v.Find("100.12.49.2")
	c.Check(ip, IsNil)
}
