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

package service_test

import (
	"errors"
	"path"
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/zzk"
	. "github.com/control-center/serviced/zzk/service"
	"github.com/control-center/serviced/zzk/service/mocks"
	. "gopkg.in/check.v1"
)

const (
	testhostid  = "hostid1"
	testcacheip = "199.115.12.18"
)

var _ = Suite(&HostIPListenerSuite{})

type HostIPListenerSuite struct {
	zzk.ZZKTestSuite
	conn     client.Connection
	listener *HostIPListener
	handler  *mocks.HostIPHandler
}

func (t *HostIPListenerSuite) SetUpTest(c *C) {
	t.ZZKTestSuite.SetUpTest(c)

	// initialize the zookeeper connection
	conn, err := zzk.GetLocalConnection("/")
	c.Assert(err, IsNil)

	// create the host
	err = conn.CreateDir(path.Join("/hosts", testhostid))
	c.Assert(err, IsNil)

	// initialize the handler
	handler := &mocks.HostIPHandler{}

	// initialize the listener
	listener := NewHostIPListener(testhostid, handler, []string{testcacheip})
	listener.SetConn(conn)

	t.conn = conn
	t.handler = handler
	t.listener = listener
}

func (t *HostIPListenerSuite) TearDownTest(c *C) {
	t.handler.AssertExpectations(c)
}

// assertCreateIP ensures the ip data is in zookeeper
func (t *HostIPListenerSuite) assertCreateIP(c *C, ipaddr, netmask, iface string) IPRequest {
	req := IPRequest{
		HostID:    testhostid,
		IPAddress: ipaddr,
	}
	err := CreateIP(t.conn, req, netmask, iface)
	c.Assert(err, IsNil)

	ip, err := GetIP(t.conn, req)
	c.Assert(err, IsNil)
	c.Assert(ip.OK, Equals, false)
	return req
}

// assertSpawnStarts ensures the spawn starts and is running
func (t *HostIPListenerSuite) assertSpawnStarts(c *C, cancel <-chan struct{}, ipaddr string) (IPRequest, <-chan struct{}) {
	c.Logf("Starting spawn for ip %s", ipaddr)

	const (
		netmask = "255.255.255.0"
		iface   = "dummy0"
	)
	t.handler.On("BindIP", ipaddr, netmask, iface).Return(nil).Twice()
	req := t.assertCreateIP(c, ipaddr, netmask, iface)

	done := make(chan struct{})
	t.listener.Pre()
	go func() {
		t.listener.Spawn(cancel, req.IPID())
		close(done)
	}()
	select {
	case <-done:
		c.Errorf("listener exited unexpectedly")
	case <-time.After(time.Second):
	}

	ip, err := GetIP(t.conn, req)
	c.Assert(err, IsNil)
	c.Assert(ip.OK, Equals, true)

	return req, done
}

// assertSpawnStops ensures the spawn has stopped
func (t *HostIPListenerSuite) assertSpawnStops(c *C, done <-chan struct{}) {
	c.Logf("Ensuring spawn exits")
	select {
	case <-done:
	case <-time.After(time.Second):
		c.Errorf("listener did not exit")
	}
}

// assertClean ensures the nodes are removed from zookeeeper
func (t *HostIPListenerSuite) assertClean(c *C, ipid string) {
	c.Logf("Ensuring ip is cleaned up for %s", ipid)

	ok, err := t.conn.Exists(path.Join("/ips", ipid))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)

	ok, err = t.conn.Exists(path.Join(t.listener.Path(), ipid))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, false)
}

// assertDirty ensures the nodes remain in zookeeper
func (t *HostIPListenerSuite) assertDirty(c *C, ipid string) {
	c.Logf("Ensuring ip is not cleaned up for %s", ipid)

	ok, err := t.conn.Exists(path.Join("/ips", ipid))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, true)

	ok, err = t.conn.Exists(path.Join(t.listener.Path(), ipid))
	c.Assert(err, IsNil)
	c.Check(ok, Equals, true)
}

// assertSpawnExits ensures that the spawn exits upon startup
func (t *HostIPListenerSuite) assertSpawnExits(c *C, ipid string) {
	c.Logf("IP spawn loop exits after starting %s", ipid)

	cancel := make(chan struct{})
	done := make(chan struct{})
	t.listener.Pre()
	go func() {
		t.listener.Spawn(cancel, ipid)
		close(done)
	}()

	t.assertSpawnStops(c, done)
	t.assertClean(c, ipid)
}

// assertBindRelease tests both cases where ReleaseIP does and does not return
// an error
func (t *HostIPListenerSuite) assertIPs(c *C, f func(ipaddr string), ips ...string) {
	for _, ipaddr := range ips {
		c.Logf("Running against ip %s", ipaddr)
		t.handler.On("ReleaseIP", ipaddr).Return(nil).Once()
		f(ipaddr)
		t.handler.On("ReleaseIP", ipaddr).Return(errors.New("release ip failed")).Once()
		f(ipaddr)
	}
}

func (t *HostIPListenerSuite) TestPath(c *C) {
	c.Assert(t.listener.Path(), Equals, path.Join("/hosts", testhostid, "/ips"))
}

func (t *HostIPListenerSuite) TestPost_NoDelete(c *C) {
	const (
		netmask = "255.255.255.0"
		iface   = "dummy0"
	)
	req := t.assertCreateIP(c, testcacheip, netmask, iface)
	t.listener.Post(map[string]struct{}{
		req.IPID(): struct{}{},
	})
	t.assertDirty(c, req.IPID())
}

func (t *HostIPListenerSuite) TestPost_Delete(c *C) {
	const (
		netmask = "255.255.255.0"
		iface   = "dummy0"
	)
	t.handler.On("ReleaseIP", testcacheip).Return(nil)
	req := t.assertCreateIP(c, testcacheip, netmask, iface)
	t.listener.Post(map[string]struct{}{})
	t.assertClean(c, req.IPID())
}

func (t *HostIPListenerSuite) TestSpawn_BadIPID(c *C) {
	err := t.conn.CreateDir(path.Join(t.listener.Path(), "foo"))
	c.Assert(err, IsNil)
	t.assertSpawnExits(c, "foo")
}

func (t *HostIPListenerSuite) TestSpawn_LoadCacheNoIP(c *C) {
	assertNoIP := func(ipaddr string) {
		c.Logf("No IP found for %s", ipaddr)
		req := IPRequest{
			HostID:    testhostid,
			IPAddress: ipaddr,
		}
		t.assertSpawnExits(c, req.IPID())
	}

	t.assertIPs(c, assertNoIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_LoadNoPoolIP(c *C) {
	assertNoPoolIP := func(ipaddr string) {
		c.Logf("Pool IP not found for %s", ipaddr)
		const (
			netmask = "255.255.255.0"
			iface   = "dummy0"
		)
		req := t.assertCreateIP(c, ipaddr, netmask, iface)
		err := t.conn.Delete(path.Join("/ips", req.IPID()))
		c.Assert(err, IsNil)
		t.assertSpawnExits(c, req.IPID())
	}

	t.assertIPs(c, assertNoPoolIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_LoadNoHostIP(c *C) {
	assertNoHostIP := func(ipaddr string) {
		c.Logf("Host IP not found for %s", ipaddr)
		const (
			netmask = "255.255.255.0"
			iface   = "dummy0"
		)
		req := t.assertCreateIP(c, ipaddr, netmask, iface)
		err := t.conn.Delete(path.Join(t.listener.Path(), req.IPID()))
		c.Assert(err, IsNil)
		t.assertSpawnExits(c, req.IPID())
	}

	t.assertIPs(c, assertNoHostIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_LoadBindFails(c *C) {
	assertBindFails := func(ipaddr string) {
		c.Logf("BindIP fails for %s", ipaddr)
		const (
			netmask = "255.255.255.0"
			iface   = "dummy0"
		)
		t.handler.On("BindIP", ipaddr, netmask, iface).Return(errors.New("bind ip failed")).Once()
		req := t.assertCreateIP(c, ipaddr, netmask, iface)
		t.assertSpawnExits(c, req.IPID())
	}

	t.assertIPs(c, assertBindFails, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_StartsNoIP(c *C) {
	assertNoIP := func(ipaddr string) {
		c.Logf("IP is deleted from zookeeper after startup for ip %s", ipaddr)
		cancel := make(chan struct{})
		req, done := t.assertSpawnStarts(c, cancel, ipaddr)
		err := DeleteIP(t.conn, req)
		c.Assert(err, IsNil)
		t.assertSpawnStops(c, done)
		t.assertClean(c, req.IPID())
	}

	t.assertIPs(c, assertNoIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_StartsNoPoolIP(c *C) {
	assertNoPoolIP := func(ipaddr string) {
		c.Logf("Pool IP is deleted from zookeeper after startup for ip %s", ipaddr)
		cancel := make(chan struct{})
		req, done := t.assertSpawnStarts(c, cancel, ipaddr)
		err := t.conn.Delete(path.Join("/ips", req.IPID()))
		c.Assert(err, IsNil)
		t.assertSpawnStops(c, done)
		t.assertClean(c, req.IPID())
	}

	t.assertIPs(c, assertNoPoolIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_StartsNoHostIP(c *C) {
	assertNoHostIP := func(ipaddr string) {
		c.Logf("Host IP is deleted from zookeeper after startup for ip %s", ipaddr)
		cancel := make(chan struct{})
		req, done := t.assertSpawnStarts(c, cancel, ipaddr)
		err := t.conn.Delete(path.Join(t.listener.Path(), req.IPID()))
		c.Assert(err, IsNil)
		t.assertSpawnStops(c, done)
		t.assertClean(c, req.IPID())
	}

	t.assertIPs(c, assertNoHostIP, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_StartsBindFails(c *C) {
	assertBindFails := func(ipaddr string) {
		c.Logf("Bind fails after startup for ip %s", ipaddr)
		cancel := make(chan struct{})
		req, done := t.assertSpawnStarts(c, cancel, ipaddr)
		t.handler.On("BindIP", ipaddr, "224.244.24.24", "dummy1").Return(errors.New("bind failed")).Once()
		err := UpdateIP(t.conn, req, func(ip *IP) bool {
			ip.Netmask = "224.244.24.24"
			ip.BindInterface = "dummy1"
			return true
		})
		c.Assert(err, IsNil)
		t.assertSpawnStops(c, done)
		t.assertClean(c, req.IPID())
	}

	t.assertIPs(c, assertBindFails, "199.18.12.1", testcacheip)
}

func (t *HostIPListenerSuite) TestSpawn_StartsCancels(c *C) {
	assertCancels := func(ipaddr string) {
		c.Logf("Shutdown spawn after startup for ip %s", ipaddr)
		cancel := make(chan struct{})
		req, done := t.assertSpawnStarts(c, cancel, ipaddr)
		close(cancel)
		t.assertSpawnStops(c, done)
		t.assertDirty(c, req.IPID())
	}

	assertCancels("199.18.12.1")
	assertCancels(testcacheip)
}
