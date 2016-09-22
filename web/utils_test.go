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

// +build unit

package web

import (
	"fmt"
	"net"

	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/web/mocks"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/zzk/service"
	. "gopkg.in/check.v1"
)

var (
	privateIp   string = "Local-IP"
	privatePort uint16 = 12345
	hostIp      string = "8.8.8.8" // Should never be a local ip.
	hostPort    uint16 = 22250

	serviceAddress = fmt.Sprintf("%s:%d", privateIp, privatePort)
	muxAddress     = fmt.Sprintf("%s:%d", hostIp, hostPort)
)

// Tests that Dial is called with serviceAddress when the hostIp is
// in the local host ip map.
func (s *TestWebSuite) TestDoNotMuxLocalConnections(c *C) {
	// Make sure hostIp is in our ip map for this test.
	ipmap[hostIp] = struct{}{}
	defer delete(ipmap, hostIp)

	var unusedConnection net.Conn
	dialer := &mocks.Dialer{}
	export := getExportDetails()

	dialer.On("Dial", "tcp4", serviceAddress).Return(unusedConnection, nil)

	fmt.Printf("dialer=%T\n", dialer)
	fmt.Printf("dialer=%v\n", dialer)

	_, err := getRemoteConnection(&export, dialer)

	c.Assert(err, IsNil)
	dialer.AssertExpectations(c)
}

// Tests that Dial is called with muxAddress when the hostIp isn't
// found in the local host ip map.
func (s *TestWebSuite) TestMuxRemoteConnections(c *C) {
	conn   := &mocks.Conn{}
	dialer := &mocks.Dialer{}
	export := getExportDetails()
	muxHeader, _ := utils.PackTCPAddress(export.PrivateIP, export.PortNumber)
	
	dialer.On("Dial", "tcp4", muxAddress).Return(conn, nil)
	conn.On("Write", muxHeader).Return(0, nil)

	fmt.Printf("dialer=%T\n", dialer)
	fmt.Printf("dialer=%v\n", dialer)

	_, err := getRemoteConnection(&export, dialer)

	c.Assert(err, Equals, utils.ErrInvalidTCPAddress)
	dialer.AssertExpectations(c)
}

func getExportDetails() registry.ExportDetails {
	return registry.ExportDetails{
		ExportBinding: service.ExportBinding{
			PortNumber: privatePort,
		},
		HostIP:    hostIp,
		MuxPort:   hostPort,
		PrivateIP: privateIp,
	}
}
