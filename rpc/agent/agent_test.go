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

// +build unit

package agent

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/utils"

	"reflect"
	"testing"
)

func TestGetInfo(t *testing.T) {

	ip, err := utils.GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	staticIPs := []string{ip}
	agent := NewServer(staticIPs)

	// Test that our IPs made it into the agent.
	if !reflect.DeepEqual(agent.staticIPs, []string{ip}) {
		t.Fatal("Failed to apply static IPs to AgentServer.")
	}

	// Test that we successfully build a host with no IP in the request
	// and static IPs defined in the AgentServer.
	h := host.New()
	request := BuildHostRequest{IP: "", PoolID: "testpool"}
	err = agent.BuildHost(request, h)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	// Test that we're not getting a wrong number of IPs for the above condition.
	if len(h.IPs) != 1 {
		t.Fatalf("Unexpected result %v (%d)", h.IPs, len(h.IPs))
	}
	// Test that we're getting the right IP for the above condition.
	if h.IPs[0].IPAddress != ip {
		t.Fatalf("Unexpected result %s (%s)", h.IPs[0].IPAddress, ip)
	}

	// Test that we successfully build a host with an IP in the request
	// and no static IPs defined in the AgentServer.
	agent = NewServer([]string{})
	h = host.New()
	request = BuildHostRequest{IP: ip, PoolID: "testpool"}
	err = agent.BuildHost(request, h)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	// Test that we're not getting a wrong number of IPs for the above condition.
	if len(h.IPs) != 1 {
		t.Fatalf("Unexpected result %v (%d)", h.IPs, len(h.IPs))
	}
	// Test that we're getting the right IP for the above condition.
	if h.IPs[0].IPAddress != ip {
		t.Fatalf("Unexpected result %s (%s)", h.IPs[0].IPAddress, ip)
	}

	// Test that we build a host with no IPs provided.
	agent = NewServer([]string{})
	h = host.New()
	request = BuildHostRequest{IP: "", PoolID: "testpool"}
	err = agent.BuildHost(request, h)
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	// Test that we're not getting a wrong number of IPs for the above condition.
	if len(h.IPs) != 1 {
		t.Fatalf("Unexpected result %v (%d)", h.IPs, len(h.IPs))
	}
	// Test that we're getting the right IP for the above condition.
	if h.IPs[0].IPAddress != ip {
		t.Fatalf("Unexpected result %s (%s)", h.IPs[0].IPAddress, ip)
	}

	// Test that we can't build a host with bad IPs provided.
	agent = NewServer([]string{})
	h = host.New()
	request = BuildHostRequest{IP: "1.2.3.4", PoolID: "testpool"}
	err = agent.BuildHost(request, h)
	if _, ok := err.(host.InvalidIPAddress); !ok {
		t.Errorf("Unexpected error %v", err)
	}

	// Test that we can't use loopback.
	request = BuildHostRequest{IP: "127.0.0.1", PoolID: "testpool"}
	err = agent.BuildHost(request, h)
	if _, ok := err.(host.IsLoopbackError); !ok {
		t.Fatalf("Unexpected error %v", err)
	}
}
