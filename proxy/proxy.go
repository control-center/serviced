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
	"bufio"
	"crypto/tls"
	"fmt"
	"io"
	"log"
	"net"
	"net/textproto"
	"strconv"
	"strings"
)

type Proxy struct {
	Name       string
	Address    string
	TCPMux     bool
	TCPMuxPort int
	UseTLS     bool
	Port       uint16
}

type TCPMux struct {
	Enabled     bool
	UseTLS      bool
	CertPEMFile string
	KeyPEMFile  string
	Port        int
}

type Config struct {
	Proxies   []Proxy
	TCPMux    TCPMux
	ServiceId string
	Command   string
}

// listenAndProxy listens, locally, on the proxy's specified Port. For each
// incoming connection a goroutine running the proxy method is created.
func (p *Proxy) ListenAndProxy() error {
	l, err := net.Listen("tcp", fmt.Sprintf(":%d", p.Port))
	if err != nil {
		log.Println("Error (net.Listen): ", err)
		return err
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("Error (net.Accept): ", err)
		}

		go p.Proxy(conn)
	}
}

// proxy takes an established local connection, Dials the remote address specified
// by the Proxy structure and then copies data to and from the resulting pair
// of endpoints.
func (p *Proxy) Proxy(local net.Conn) {
	remoteAddr := p.Address
	// NOTE: here we are relying on the initial remoteAddr to have the
	//       publicly exposed port for the target service. If TCPMux is
	//       in play that port will be replaced with the TCPMux port, so
	//       we grab it here in order to be able to create a proper Zen-Service
	//       header later.
	remotePort, err := strconv.Atoi(strings.Split(remoteAddr, ":")[1])
	if err != nil {
		log.Println("Error (strconv.Atoi): ", err)
	}

	if p.TCPMux {
		remoteAddr = fmt.Sprintf("%s:%d", strings.Split(remoteAddr, ":")[0], p.TCPMuxPort)
	}

	var remote net.Conn

	if p.UseTLS && p.TCPMux { // Only do TLS if connecting to a TCPMux
		config := tls.Config{InsecureSkipVerify: true}
		remote, err = tls.Dial("tcp", remoteAddr, &config)
	} else {
		remote, err = net.Dial("tcp", remoteAddr)
	}
	if err != nil {
		log.Println("Error (net.Dial): ", err)
		return
	}

	if p.TCPMux {
		io.WriteString(remote, fmt.Sprintf("Zen-Service: %s/%d\r\n", p.Name, remotePort))
	}

	go io.Copy(local, remote)
	go io.Copy(remote, local)
}

// sendMuxError logs an error message and attempts to write it to the connected
// endpoint
func sendMuxError(conn net.Conn, source, facility, msg string, err error) {
	log.Printf("%s Error (%s): %v\n", source, facility, err)
	if _, e := conn.Write([]byte(msg)); e != nil {
		log.Println(e)
	}
}

// muxConnection takes an inbound connection reads MIME headers from it and
// then attempts to set up a connection to the service specified by the
// Zen-Service header. If the Zen-Service header is missing or the requested
// service is not running (listening) on the local host and error message
// is sent to the requestor and its connection is closed. Otherwise data is
// proxied between the requestor and the local service.
func (mux TCPMux) MuxConnection(conn net.Conn) {
	rdr := textproto.NewReader(bufio.NewReader(conn))
	hdr, err := rdr.ReadMIMEHeader()
	if err != nil {
		sendMuxError(conn, "listenAndMux", "textproto.ReadMIMEHeader", "bad request (no headers)", err)
		conn.Close()
		return
	}

	zs, ok := hdr["Zen-Service"]
	if ok == false {
		sendMuxError(conn, "listenAndMux", "MIMEHeader", "bad request (no Zen-Service header)", err)
		conn.Close()
		return
	}

	port, err := strconv.Atoi(strings.Split(zs[0], "/")[1])
	if err != nil {
		sendMuxError(conn, "listenAndMux", "Zen-Service Header", "bad Zen-Service spec", err)
		conn.Close()
		return
	}

	svc, err := net.Dial("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		sendMuxError(conn, "listenAndMux", "net.Dial", "cannot connect to service", err)
		conn.Close()
		return
	}

	go io.Copy(conn, svc)
	go io.Copy(svc, conn)
}

// listenAndMux listens for incoming connections and attempts to multiplex them
// to the local service that they request via a Zen-Service header in their
// initial message.
func (mux *TCPMux) ListenAndMux() {
	var l net.Listener
	var err error

	if mux.UseTLS == false {
		l, err = net.Listen("tcp", fmt.Sprintf(":%d", mux.Port))
	} else {
		cert, cerr := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
		if cerr != nil {
			log.Println("listenAndMux Error (tls.X509KeyPair): ", cerr)
			return
		}

		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		l, err = tls.Listen("tcp", fmt.Sprintf(":%d", mux.Port), &tlsConfig)
	}
	if err != nil {
		log.Printf("listenAndMux Error (net.Listen): ", err)
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			log.Println("listenAndMux Error (net.Accept): ", err)
			return
		}

		go mux.MuxConnection(conn)
	}
}

