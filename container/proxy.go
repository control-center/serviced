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

package container

import (
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"strings"

	"github.com/zenoss/glog"
)

/*
The 'prxy' service implemented here provides both a prxy for outbound
service requests and a multiplexer for inbound requests. The diagram below
illustrates one way proxies interoperate.

      proxy A                   proxy B
      +-----------+             +-----------+
    22250         |     +---->22250 ---------------+
      |           |     |       |           |      |
 +-->3306 --------------+       |           |      |
 +-->4369 --------------+       |           |      |
 |    |           |             |           |      |
 |    +-----------+             +-----------+      |
 |                                                 |
 +----zensvc                    mysql/3306 <-------+
                                rabbitmq/4369 <----+

proxy A exposes MySQL and RabbitMQ ports, 3306 and 4369 respectively, to its
zensvc. When zensvc connects to those ports proxy A forwards the resulting
traffic to the appropriate remote services via the TCPMux port exposed by
proxy B.

Start the service from the command line by typing

prxy [OPTIONS] SERVICE_ID

  -certfile="": path to public certificate file (defaults to compiled in public cert)
  -endpoint="127.0.0.1:4979": serviced endpoint address
  -keyfile="": path to private key file (defaults to compiled in private key)
  -mux=true: enable port multiplexing
  -muxport=22250: multiplexing port to use
  -tls=true: enable TLS

To terminate the prxy service connect to it via port 4321 and it will exit.
The netcat (nc) command is particularly useful for this:

    nc 127.0.0.1 4321
*/

type proxy struct {
	name             string          // Name of the remote service
	tenantEndpointID string          // Tenant endpoint ID
	addresses        []string        // Public IP:Port of the remote service
	tcpMuxPort       uint16          // the port to use for TCP Muxing, 0 is disabled
	useTLS           bool            // use encryption over mux port
	closing          chan chan error // internal shutdown signal
	newAddresses     chan []string   // a stream of updates to the addresses
	listener         net.Listener    // handle on the listening socket
}

// Newproxy create a new proxy object. It starts listening on the prxy port asynchronously.
func newProxy(name, tenantEndpointID string, tcpMuxPort uint16, useTLS bool, listener net.Listener) (p *proxy, err error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("prxy: name can not be empty")
	}
	p = &proxy{
		name:             name,
		tenantEndpointID: tenantEndpointID,
		addresses:        make([]string, 0),
		tcpMuxPort:       tcpMuxPort,
		useTLS:           useTLS,
		listener:         listener,
	}
	p.newAddresses = make(chan []string, 2)
	go p.listenAndproxy()
	return p, nil
}

// Name() returns the application name associated with the prxy
func (p *proxy) Name() string {
	return p.name
}

// String() pretty prints the proxy struct.
func (p *proxy) String() string {
	return fmt.Sprintf("proxy[%s; %s]=>%v", p.name, p.listener, p.addresses)
}

// TCPMuxPort() returns the tcp port use for muxing, 0 if not used.
func (p *proxy) TCPMuxPort() uint16 {
	return p.tcpMuxPort
}

// UseTLS() returns true if TLS is used during tcp muxing.
func (p *proxy) UseTLS() bool {
	return p.useTLS
}

// Set a new Destination Address set for the prxy
func (p *proxy) SetNewAddresses(addresses []string) {
	p.newAddresses <- addresses
}

// Close() terminates the prxy; it can not be restarted.
func (p *proxy) Close() error {
	p.listener.Close()
	errc := make(chan error)
	p.closing <- errc
	return <-errc
}

// listenAndproxy listens, locally, on the prxy's specified Port. For each
// incoming connection a goroutine running the prxy method is created.
func (p *proxy) listenAndproxy() {

	connections := make(chan net.Conn)
	go func(lsocket net.Listener, conns chan net.Conn) {
		for {
			conn, err := lsocket.Accept()
			if err != nil {
				glog.Fatal("Error (net.Accept): ", err)
			}
			conns <- conn
		}
	}(p.listener, connections)

	i := 0
	for {
		select {
		case conn := <-connections:
			if len(p.addresses) == 0 {
				glog.Warningf("No remote services available for prxying %s", p)
				conn.Close()
				continue
			}
			i++
			// round robin connections to list of addresses
			glog.V(1).Infof("choosing address from %v", p.addresses)
			go p.prxy(conn, p.addresses[i%len(p.addresses)])
		case p.addresses = <-p.newAddresses:
		case errc := <-p.closing:
			p.listener.Close()
			errc <- nil
			return
		}
	}
}

// prxy takes an established local connection, Dials the remote address specified
// by the proxy structure and then copies data to and from the resulting pair
// of endpoints.
func (p *proxy) prxy(local net.Conn, address string) {
	remoteAddr := address
	// NOTE: here we are relying on the initial remoteAddr to have the
	//       publicly exposed port for the target service. If TCPMux is
	//       in play that port will be replaced with the TCPMux port, so
	//       we grab it here in order to be able to create a proper Zen-Service
	//       header later.
	isMux := false
	hostIP := strings.Split(remoteAddr, ":")[0]
	if p.tcpMuxPort > 0 && !isLocalAddress(hostIP) {
		isMux = true
		remoteAddr = fmt.Sprintf("%s:%d", strings.Split(remoteAddr, ":")[0], p.tcpMuxPort)
	}

	var remote net.Conn
	var err error

	glog.V(2).Infof("Dialing hostAgent:%v to prxy %v<->%v<->%v",
		remoteAddr, local.LocalAddr(), local.RemoteAddr(), address)
	if p.useTLS && isMux { // Only do TLS if connecting to a TCPMux
		config := tls.Config{InsecureSkipVerify: true}
		remote, err = tls.Dial("tcp4", remoteAddr, &config)
	} else {
		remote, err = net.Dial("tcp4", remoteAddr)
	}
	if err != nil {
		glog.Error("Error (net.Dial): ", err)
		return
	}

	if isMux {
		io.WriteString(remote, fmt.Sprintf("%s:%s:%s\n", p.tenantEndpointID, p.name, address))
	}

	glog.V(2).Infof("Using hostAgent:%v to prxy %v<->%v<->%v<->%v",
		remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	go func(address string) {
		defer local.Close()
		defer remote.Close()
		io.Copy(local, remote)
		glog.V(2).Infof("Closing hostAgent:%v to prxy %v<->%v<->%v<->%v",
			remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	}(address)
	go func(address string) {
		defer local.Close()
		defer remote.Close()
		io.Copy(remote, local)
		glog.V(2).Infof("closing hostAgent:%v to prxy %v<->%v<->%v<->%v",
			remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	}(address)
}
