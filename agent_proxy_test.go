// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

// This file tests the LoadBalancer interface aspect of the host agent.
package serviced

import (
	"github.com/zenoss/serviced/dao"

	"testing"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

func TestAddControlPlaneEndpoints(t *testing.T) {
	agent := &HostAgent{}
	agent.master = "127.0.0.1:0"
	agent.uiport = ":443"
	endpoints := make(map[string][]*dao.ApplicationEndpoint)

	consumer_endpoint := dao.ApplicationEndpoint{}
	consumer_endpoint.ServiceId = "controlplane_consumer"
	consumer_endpoint.ContainerIP = "127.0.0.1"
	consumer_endpoint.ContainerPort = 8444
	consumer_endpoint.HostPort = 8443
	consumer_endpoint.HostIp = "127.0.0.1"
	consumer_endpoint.Protocol = "tcp"

	controlplane_endpoint := dao.ApplicationEndpoint{}
	controlplane_endpoint.ServiceId = "controlplane"
	controlplane_endpoint.ContainerIP = "127.0.0.1"
	controlplane_endpoint.ContainerPort = 443
	controlplane_endpoint.HostPort = 443
	controlplane_endpoint.HostIp = "127.0.0.1"
	controlplane_endpoint.Protocol = "tcp"

	agent.addContolPlaneEndpoint(endpoints)
	agent.addContolPlaneConsumerEndpoint(endpoints)

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

	if *endpoints["tcp:8444"][0] != consumer_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", *endpoints["tcp:8444"][0], consumer_endpoint)
	}

	if *endpoints["tcp:443"][0] != controlplane_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", *endpoints["tcp:443"][0], controlplane_endpoint)
	}
}
