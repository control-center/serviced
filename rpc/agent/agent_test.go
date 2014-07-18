// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package agent

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/utils"

	"strings"
	"testing"
)

func TestGetInfo(t *testing.T) {

	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	staticIPs := []string{ip}
	agent := NewServer(staticIPs)
	h := host.New()
	request := BuildHostRequest{IP: "", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err != nil && !strings.Contains(err.Error(), "not valid for this host") {
		t.Fatalf("Unexpected error %v", err)
	}
	if len(h.IPs) != 0 {
		t.Fatalf("Unexpected result %v (%d)", h.IPs, len(h.IPs))
	}

	request = BuildHostRequest{IP: "127.0.0.1", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err == nil || err.Error() != "loopback address 127.0.0.1 cannot be used to register a host" {
		t.Fatalf("Unexpected error %v", err)
	}

	request = BuildHostRequest{IP: "", PoolID: "testpool"}

	err = agent.BuildHost(request, h)
	if err == nil || !strings.Contains(err.Error(), "not valid for this host") {
		t.Errorf("Unexpected error %v", err)
	}

}
