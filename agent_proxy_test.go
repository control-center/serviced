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
	"github.com/zenoss/serviced/isvcs"

	"testing"
)

// assert that the HostAgent implements the LoadBalancer interface
var _ LoadBalancer = &HostAgent{}

var testManager *isvcs.Manager

func createTestService() {
	testManager = isvcs.NewManager("unix:///var/run/docker.sock", "/tmp")
}

func TestGetServiceEndpoints(t *testing.T) {

}
