/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

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
	endpoints := make(map[string][]*dao.ApplicationEndpoint)

	consumer_endpoint := dao.ApplicationEndpoint{}
	consumer_endpoint.ServiceId = "controlplane_consumer"
	consumer_endpoint.ContainerIp = "127.0.0.1"
	consumer_endpoint.ContainerPort = 8444
	consumer_endpoint.HostPort = 8443
	consumer_endpoint.HostIp = "127.0.0.1"
	consumer_endpoint.Protocol = "tcp"

	controlplane_endpoint := dao.ApplicationEndpoint{}
	controlplane_endpoint.ServiceId = "controlplane"
	controlplane_endpoint.ContainerIp = "127.0.0.1"
	controlplane_endpoint.ContainerPort = 8787
	controlplane_endpoint.HostPort = 8787
	controlplane_endpoint.HostIp = "127.0.0.1"
	controlplane_endpoint.Protocol = "tcp"

	agent.addContolPlaneEndpoint(endpoints)
	agent.addContolPlaneConsumerEndpoint(endpoints)

	if _, ok := endpoints["tcp:8444"]; !ok {
		t.Fatalf(" mapping failed missing key[\"tcp:8444\"]")
	}

	if _, ok := endpoints["tcp:8787"]; !ok {
		t.Fatalf(" mapping failed missing key[\"tcp:8787\"]")
	}

	if len(endpoints["tcp:8444"]) != 1 {
		t.Fatalf(" mapping failed len(\"tcp:8444\"])=%d expected 1", len(endpoints["tcp:8444"]))
	}

	if len(endpoints["tcp:8787"]) != 1 {
		t.Fatalf(" mapping failed len(\"tcp:8787\"])=%d expected 1", len(endpoints["tcp:8787"]))
	}

	if *endpoints["tcp:8444"][0] != consumer_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", *endpoints["tcp:8444"][0], consumer_endpoint)
	}

	if *endpoints["tcp:8787"][0] != controlplane_endpoint {
		t.Fatalf(" mapping failed %+v expected %+v", *endpoints["tcp:8787"][0], controlplane_endpoint)
	}
}
