/*
The 'proxy' service implemented here provides both a proxy for outbound
service requests and a multiplexer for inbound requests. The diagram below
illustrates one way proxies interoperate.

      Proxy A                   Proxy B
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

Proxy A exposes MySQL and RabbitMQ ports, 3306 and 4369 respectively, to its
zensvc. When zensvc connects to those ports Proxy A forwards the resulting
traffic to the appropriate remote services via the TCPMux port exposed by
Proxy B.

Start the service from the command line by typing

proxy [OPTIONS] SERVICE_ID

  -certfile="": path to public certificate file (defaults to compiled in public cert)
  -endpoint="127.0.0.1:4979": serviced endpoint address
  -keyfile="": path to private key file (defaults to compiled in private key)
  -mux=true: enable port multiplexing
  -muxport=22250: multiplexing port to use
  -tls=true: enable TLS

To terminate the proxy service connect to it via port 4321 and it will exit.
The netcat (nc) command is particularly useful for this:

    nc 127.0.0.1 4321
*/
package proxy

import (
	"crypto/tls"
	"fmt"
	"github.com/zenoss/glog"
	"io"
	"net"
	"strconv"
	"strings"
)

type Proxy struct {
	name         string          // Name of the remote service
	addresses    []string        // Public IP:Port of the remote service
	tcpMuxPort   uint16          // the port to use for TCP Muxing, 0 is disabled
	useTLS       bool            // use encryption over mux port
	closing      chan chan error // internal shutdown signal
	newAddresses chan []string   // a stream of updates to the addresses
	listener     net.Listener    // handle on the listening socket
}

// NewProxy create a new Proxy object. It starts listening on the proxy port asynchronously.
func NewProxy(name string, tcpMuxPort uint16, useTLS bool, listener net.Listener) (proxy *Proxy, err error) {
	if len(name) == 0 {
		return nil, fmt.Errorf("proxy: name can not be empty")
	}
	proxy = &Proxy{
		name:       name,
		addresses:  make([]string, 0),
		tcpMuxPort: tcpMuxPort,
		useTLS:     useTLS,
		listener:   listener,
	}
	proxy.newAddresses = make(chan []string, 2)
	go proxy.listenAndProxy()
	return proxy, nil
}


// Name() returns the application name associated with the proxy
func (p *Proxy) Name() string {
	return p.name
}

// String() pretty prints the Proxy struct.
func (p *Proxy) String() string {
	return fmt.Sprintf("Proxy[%s; %s]=>%v", p.name, p.listener, p.addresses)
}

// TCPMuxPort() returns the tcp port use for muxing, 0 if not used.
func (p *Proxy) TCPMuxPort() uint16 {
	return p.tcpMuxPort
}

// UseTLS() returns true if TLS is used during tcp muxing.
func (p *Proxy) UseTLS() bool {
	return p.useTLS
}

// Set a new Destination Address set for the proxy
func (p *Proxy) SetNewAddresses(addresses []string) {
	p.newAddresses <- addresses
}

// Close() terminates the proxy; it can not be restarted.
func (p *Proxy) Close() error {
	p.listener.Close()
	errc := make(chan error)
	p.closing <- errc
	return <-errc
}

// listenAndProxy listens, locally, on the proxy's specified Port. For each
// incoming connection a goroutine running the proxy method is created.
func (p *Proxy) listenAndProxy() {

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
				glog.Warningf("No remote services available for proxying %s", p)
				conn.Close()
				continue
			}
			i += 1
			// round robin connections to list of addresses
			glog.Infof("choosing address from %v", p.addresses)
			go p.proxy(conn, p.addresses[i%len(p.addresses)])
		case p.addresses = <-p.newAddresses:
		case errc := <-p.closing:
			p.listener.Close()
			errc <- nil
			return
		}
	}
}

// proxy takes an established local connection, Dials the remote address specified
// by the Proxy structure and then copies data to and from the resulting pair
// of endpoints.
func (p *Proxy) proxy(local net.Conn, address string) {
	remoteAddr := address
	// NOTE: here we are relying on the initial remoteAddr to have the
	//       publicly exposed port for the target service. If TCPMux is
	//       in play that port will be replaced with the TCPMux port, so
	//       we grab it here in order to be able to create a proper Zen-Service
	//       header later.
	remotePort, err := strconv.Atoi(strings.Split(remoteAddr, ":")[1])
	if err != nil {
		glog.Error("Error (strconv.Atoi): ", err)
	}

	if p.tcpMuxPort > 0 {
		remoteAddr = fmt.Sprintf("%s:%d", strings.Split(remoteAddr, ":")[0], p.tcpMuxPort)
	}

	var remote net.Conn

	glog.Info("Abount to dial: %s", remoteAddr)
	if p.useTLS && (p.tcpMuxPort > 0) { // Only do TLS if connecting to a TCPMux
		config := tls.Config{InsecureSkipVerify: true}
		remote, err = tls.Dial("tcp4", remoteAddr, &config)
	} else {
		remote, err = net.Dial("tcp4", remoteAddr)
	}
	if err != nil {
		glog.Error("Error (net.Dial): ", err)
		return
	}

	if p.tcpMuxPort > 0 {
		io.WriteString(remote, fmt.Sprintf("Zen-Service: %s/%d\n\n", p.name, remotePort))
	}

	go io.Copy(local, remote)
	go io.Copy(remote, local)
}
