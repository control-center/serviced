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
	"math/rand"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/zenoss/glog"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

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
  tls is always enabled

To terminate the prxy service connect to it via port 4321 and it will exit.
The netcat (nc) command is particularly useful for this:

    nc 127.0.0.1 4321
*/

type addressTuple struct {
	host          string // IP of the host on which the container is running
	containerAddr string // Container IP:port of the remote service
}

type proxy struct {
	name             string              // Name of the remote service
	tenantEndpointID string              // Tenant endpoint ID
	addresses        []addressTuple      // Public/container IP:Port of the remote service
	tcpMuxPort       uint16              // the port to use for TCP Muxing, 0 is disabled
	useTLS           bool                // use encryption over mux port
	closing          chan chan error     // internal shutdown signal
	newAddresses     chan []addressTuple // a stream of updates to the addresses
	listener         net.Listener        // handle on the listening socket
	allowDirectConn  bool                // allow container to container connections
}

// Newproxy create a new proxy object. It starts listening on the prxy port asynchronously.
func newProxy(name, tenantEndpointID string, tcpMuxPort uint16, useTLS bool, listener net.Listener, allowDirectConn bool) (p *proxy, err error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("prxy: name can not be empty")
	}
	p = &proxy{
		name:             name,
		tenantEndpointID: tenantEndpointID,
		addresses:        make([]addressTuple, 0),
		tcpMuxPort:       tcpMuxPort,
		useTLS:           useTLS,
		listener:         listener,
		allowDirectConn:  allowDirectConn,
	}
	p.newAddresses = make(chan []addressTuple, 2)
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
func (p *proxy) SetNewAddresses(addresses []addressTuple) {
	// Randomize the addresses so not all instances get them in the same order
	dest := make([]addressTuple, len(addresses))
	perm := rand.Perm(len(addresses))
	for i, v := range perm {
		dest[v] = addresses[i]
	}
	p.newAddresses <- dest
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
				glog.Warningf("No remote services available for prxying %v", p)
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

func getPort(addr string) (int, error) {
	parts := strings.Split(addr, ":")
	if len(parts) == 0 {
		return 0, fmt.Errorf("could not determint port from %v", addr)
	}
	port := parts[len(parts)-1]
	return strconv.Atoi(port)
}

// prxy takes an established local connection, Dials the remote address specified
// by the proxy structure and then copies data to and from the resulting pair
// of endpoints.
func (p *proxy) prxy(local net.Conn, address addressTuple) {

	var (
		remote net.Conn
		err    error
	)
	glog.V(2).Infof("Setting up proxy for %#v", address)
	isLocalContainer := false
	localAddr := address.containerAddr
	if p.allowDirectConn {
		//check if the host for the container is running on the same host
		isLocalContainer = isLocalAddress(address.host)
		glog.V(4).Infof("Checking is local for %s %s in %#v", address.host, isLocalContainer, hostIPs)
		// don't proxy localhost addresses, we'll end up in a loop
		if isLocalContainer {
			switch {
			case strings.HasPrefix(address.host, "127"):
				isLocalContainer = false
			case address.host == "localhost":
				isLocalContainer = false
			case strings.HasPrefix(address.containerAddr, "127") || strings.HasPrefix(address.containerAddr, "localhost:"):
				//if the host is local and the container has a local style addr
				//then container is exposing port directly on host; go to host and use container port
				if containerPort, err := getPort(address.containerAddr); err != nil {
					glog.Warningf("could not get port %v", err)
					isLocalContainer = false
				} else {
					localAddr = fmt.Sprintf("%s:%d", address.host, containerPort)
				}
			}
		}
	}
	if p.tcpMuxPort == 0 {
		// TODO: Do this properly
		glog.Errorf("Mux port is unspecified. Using default of 22250.")
		p.tcpMuxPort = 22250
	}

	muxAddr := fmt.Sprintf("%s:%d", address.host, p.tcpMuxPort)

	switch {
	case isLocalContainer:
		glog.V(2).Infof("dialing local addr=> %s", localAddr)
		remote, err = net.Dial("tcp4", localAddr)
	case p.useTLS:
		glog.V(2).Infof("dialing remote tls => %s", muxAddr)
		config := tls.Config{InsecureSkipVerify: true}
		remote, err = tls.Dial("tcp4", muxAddr, &config)
	default:
		glog.V(2).Infof("dialing remote => %s", muxAddr)
		remote, err = net.Dial("tcp4", muxAddr)
	}
	if err != nil {
		glog.Error("Error (net.Dial): ", err)
		return
	}

	if !isLocalContainer {
		muxHeader := fmt.Sprintf("%s:%s:%s\n", p.tenantEndpointID, p.name, address.containerAddr)
		glog.V(1).Infof("writing socket protocol %s", muxHeader)
		// Write the container address as the first line, if we use the mux
		io.WriteString(remote, muxHeader)
	}

	glog.V(2).Infof("Using hostAgent:%v to prxy %v<->%v<->%v<->%v",
		remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	go func(address string) {
		defer local.Close()
		defer remote.Close()
		io.Copy(local, remote)
		glog.V(2).Infof("Closing hostAgent:%v to prxy %v<->%v<->%v<->%v",
			remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	}(address.containerAddr)
	go func(address string) {
		defer local.Close()
		defer remote.Close()
		io.Copy(remote, local)
		glog.V(2).Infof("closing hostAgent:%v to prxy %v<->%v<->%v<->%v",
			remote.RemoteAddr(), local.LocalAddr(), local.RemoteAddr(), remote.LocalAddr(), address)
	}(address.containerAddr)
}
