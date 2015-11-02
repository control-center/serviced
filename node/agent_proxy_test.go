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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// This file tests the LoadBalancer interface aspect of the host agent.
package node

import (
	"github.com/control-center/serviced/domain/applicationendpoint"

	"testing"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

func TestAddControlPlaneEndpoints(t *testing.T) {
	agent := &HostAgent{}
	agent.master = "127.0.0.1:0"
	agent.uiport = ":443"
	endpoints := make(map[string][]applicationendpoint.ApplicationEndpoint)

	consumer_endpoint := applicationendpoint.ApplicationEndpoint{}
	consumer_endpoint.ServiceID = "controlplane_consumer"
	consumer_endpoint.Application = "controlplane_consumer"
	consumer_endpoint.ContainerIP = "127.0.0.1"
	consumer_endpoint.ContainerPort = 8443
	consumer_endpoint.ProxyPort = 8444
	consumer_endpoint.HostPort = 8443
	consumer_endpoint.HostIP = "127.0.0.1"
	consumer_endpoint.Protocol = "tcp"

	controlplane_endpoint := applicationendpoint.ApplicationEndpoint{}
	controlplane_endpoint.ServiceID = "controlplane"
	controlplane_endpoint.Application = "controlplane"
	controlplane_endpoint.ContainerIP = "127.0.0.1"
	controlplane_endpoint.ContainerPort = 443
	controlplane_endpoint.ProxyPort = 443
	controlplane_endpoint.HostPort = 443
	controlplane_endpoint.HostIP = "127.0.0.1"
	controlplane_endpoint.Protocol = "tcp"

	agent.addControlPlaneEndpoint(endpoints)
	agent.addControlPlaneConsumerEndpoint(endpoints)

	if _, ok := endpoints["tcp:8444"]; !ok {
		t.Fatalf(" mapping failed missing key[\"tcp:8444\"]")
	}

	if _, ok := endpoints["tcp:443"]; !ok {
		t.Fatalf(" mapping failed missing key[\"tcp:443\"]")
	}

	if len(endpoints["tcp:8444"]) != 1 {
		t.Fatalf(" mapping failed len(\"tcp:8444\"])=%d expected 1", len(endpoints["tcp:8444"]))
	}

	if len(endpoints["tcp:443"]) != 1 {
		t.Fatalf(" mapping failed len(\"tcp:443\"])=%d expected 1", len(endpoints["tcp:443"]))
	}

	if endpoints["tcp:8444"][0] != consumer_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", endpoints["tcp:8444"][0], consumer_endpoint)
	}

	if endpoints["tcp:443"][0] != controlplane_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", endpoints["tcp:443"][0], controlplane_endpoint)
	}
}
