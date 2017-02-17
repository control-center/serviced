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

// +build unit

package node_test

import (
	"errors"
	"fmt"
	"net"

	. "github.com/control-center/serviced/node"
	"github.com/control-center/serviced/node/mocks"
	. "gopkg.in/check.v1"
)

var _ = Suite(&HostAgentTestSuite{})

type HostAgentTestSuite struct {
	m *mocks.VIP
	a *HostAgent
}

func (t *HostAgentTestSuite) SetUpTest(c *C) {
	t.m = &mocks.VIP{}
	t.a = &HostAgent{}
	t.a.SetVIP(t.m)
}

func (t *HostAgentTestSuite) TearDownTest(c *C) {
	t.m.AssertExpectations(c)
}

func (t *HostAgentTestSuite) TestBindIP_Match(c *C) {
	var (
		ipprefix = "1.2.3.4"
		netmask  = "255.255.255.0"
		iface    = "dummy0"
	)
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()

	// vip exists and is a match
	vip := &IP{
		Addr:   fmt.Sprintf("%s/%d", ipprefix, cidr),
		Device: iface,
		Label:  fmt.Sprintf("%s:cc12", iface),
	}
	t.m.On("Find", "1.2.3.4").Return(vip).Once()
	err := t.a.BindIP(ipprefix, netmask, iface)
	c.Assert(err, IsNil)
}

func (t *HostAgentTestSuite) TestBindIP_FailRelease(c *C) {
	var (
		ipprefix = "1.2.3.4"
		netmask  = "255.255.255.0"
		iface    = "dummy0"
	)
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()

	// vip exists but not a match
	vip := &IP{
		Addr:   fmt.Sprintf("%s/%d", ipprefix, cidr),
		Device: "dum0",
		Label:  "dum0:cc12",
	}
	t.m.On("Find", "1.2.3.4").Return(vip).Once()

	// fail on release
	expectedErr := errors.New("could not release ip")
	t.m.On("Release", vip.Addr, vip.Device).Return(expectedErr).Once()
	err := t.a.BindIP(ipprefix, netmask, iface)
	c.Assert(err, Equals, expectedErr)
}

func (t *HostAgentTestSuite) TestBindIP_NoInterface(c *C) {
	var (
		ipprefix = "1.2.3.4"
		netmask  = "255.255.255.0"
		iface    = "qqqqqqq99"
	)
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()

	// vip exists but not a match
	vip := &IP{
		Addr:   fmt.Sprintf("%s/%d", ipprefix, cidr),
		Device: "dum0",
		Label:  "dum0:cc12",
	}
	t.m.On("Find", "1.2.3.4").Return(vip).Once()
	t.m.On("Release", vip.Addr, vip.Device).Return(nil)
	err := t.a.BindIP(ipprefix, netmask, iface)
	c.Assert(err, NotNil)
}

func (t *HostAgentTestSuite) TestBindIP_BindFailed(c *C) {
	ifs, err := net.Interfaces()
	c.Assert(err, IsNil)
	if len(ifs) == 0 {
		c.Skip("no available interfaces")
	}

	var (
		ipprefix = "1.2.3.4"
		netmask  = "255.255.255.0"
		iface    = ifs[0].Name
	)
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()

	// vip not found
	t.m.On("Find", "1.2.3.4").Return(nil).Once()
	expectedErr := errors.New("could not bind ip")
	t.m.On("Bind", fmt.Sprintf("%s/%d", ipprefix, cidr), iface).Return(expectedErr).Once()
	err = t.a.BindIP(ipprefix, netmask, iface)
	c.Assert(err, Equals, expectedErr)
}

func (t *HostAgentTestSuite) TestBindIP_BindSuccess(c *C) {
	ifs, err := net.Interfaces()
	c.Assert(err, IsNil)
	if len(ifs) == 0 {
		c.Skip("no available interfaces")
	}

	var (
		ipprefix = "1.2.3.4"
		netmask  = "255.255.255.0"
		iface    = ifs[0].Name
	)
	cidr, _ := net.IPMask(net.ParseIP(netmask).To4()).Size()

	// vip not found
	t.m.On("Find", "1.2.3.4").Return(nil).Once()
	t.m.On("Bind", fmt.Sprintf("%s/%d", ipprefix, cidr), iface).Return(nil).Once()
	err = t.a.BindIP(ipprefix, netmask, iface)
	c.Assert(err, IsNil)
}

func (t *HostAgentTestSuite) TestReleaseIP_NoBind(c *C) {
	var ipprefix = "1.2.3.4"

	t.m.On("Find", "1.2.3.4").Return(nil)
	err := t.a.ReleaseIP(ipprefix)
	c.Assert(err, IsNil)
}

func (t *HostAgentTestSuite) TestReleaseIP_ReleaseFail(c *C) {
	var ipprefix = "1.2.3.5"
	vip := &IP{
		Addr:   fmt.Sprintf("%s/24", ipprefix),
		Device: "dum0",
		Label:  "dum0:cc12",
	}
	t.m.On("Find", ipprefix).Return(vip)
	expectedErr := errors.New("could not release ip")
	t.m.On("Release", vip.Addr, vip.Device).Return(expectedErr)
	err := t.a.ReleaseIP(ipprefix)
	c.Assert(err, Equals, expectedErr)
}

func (t *HostAgentTestSuite) TestReleaseIP_ReleaseSuccess(c *C) {
	var ipprefix = "1.2.3.5"
	vip := &IP{
		Addr:   fmt.Sprintf("%s/24", ipprefix),
		Device: "dum0",
		Label:  "dum0:cc12",
	}
	t.m.On("Find", ipprefix).Return(vip)
	t.m.On("Release", vip.Addr, vip.Device).Return(nil)
	err := t.a.ReleaseIP(ipprefix)
	c.Assert(err, IsNil)
}
