// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/utils"

	"testing"
)

func TestGetInfo(t *testing.T) {

	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	agent := NewServer()
	h := host.New()
	request := BuildHostRequest{IP: "", PoolID: "testpool", IPResources: make([]string, 0)}

	err = agent.BuildHost(request, h)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if len(h.IPs) != 1 {
		t.Errorf("Unexpected result %v", h.IPs)
	}
	if h.IPAddr != ip {
		t.Errorf("Expected ip %v, got %v", ip, h.IPs)
	}
	if h.IPs[0].IPAddress != ip {
		t.Errorf("Expected ip %v, got %v", ip, h.IPs)
	}

	request = BuildHostRequest{IP: "127.0.0.1", PoolID: "testpool", IPResources: []string{}}

	err = agent.BuildHost(request, h)
	if err == nil || err.Error() != "loopback address 127.0.0.1 cannot be used to register a host" {
		t.Errorf("Unexpected error %v", err)
	}

	request = BuildHostRequest{IP: "", PoolID: "testpool", IPResources: []string{"127.0.0.1"}}

	err = agent.BuildHost(request, h)
	if err == nil || err.Error() != "loopback address 127.0.0.1 cannot be used as an IP Resource" {
		t.Errorf("Unexpected error %v", err)
	}

}
