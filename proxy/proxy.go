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

    proxy -config <config filename>

Where the config file is a JSON file with the structure

{
    "TCPMux": {
        "Enabled": <true | false>,
        "UseTLS" : <true | false>
    },
    "Proxies": [
        { "Name": <service name>, "Address": "n.n.n.n:nnnn", "TCPMux": <true | false>, "UseTLS": <true | false>, "Port": nnnn },
        { "Name": <service name>, "Address": "n.n.n.n:nnnn", "TCPMux": <true | false>, "UseTLS": <true | false>, "Port": nnnn }
    ]
}

TCPMux determines whether or not the proxy service will multiplex listening for incoming
service requests on the 'standard' TCPMux port: 22250 and whether or not those requests
are secured via TLS.

Proxies is an array of proxied service definitions. The Address field specifies
the remote IP address and port for the named service. The Port field specifies
the local port on which the proxies will accept service requests for the named
service. The TCPMux field indicates whether or not connections should be proxied
to the TCPMux port at the remote IP address.

To terminate the proxy service connect to it via port 4321 and it will exit.
The netcat (nc) command is particularly useful for this:

    nc 127.0.0.1 4321
*/
package main

import (
	"bufio"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/zenoss/serviced"
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

// Store the command line options
var options struct {
	muxport          int
	mux              bool
	servicedId       string
	tls              bool
	keyPEMFile       string
	certPEMFile      string
	servicedEndpoint string
}

// Setup flag options (static block)
func init() {
	flag.IntVar(&options.muxport, "muxport", 22250, "multiplexing port to use")
	flag.BoolVar(&options.mux, "mux", true, "enable port multiplexing")
	flag.BoolVar(&options.tls, "tls", true, "enable TLS")
	flag.StringVar(&options.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	flag.StringVar(&options.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	flag.StringVar(&options.servicedEndpoint, "endpoint", "127.0.0.1:4979", "serviced endpoint address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage: proxy [OPTIONS] SERVICE_ID\n\n")
		flag.PrintDefaults()
	}
}

type Proxy struct {
	Name    string
	Address string
	TCPMux  bool
	UseTLS  bool
	Port    int
}

type TCPMux struct {
	Enabled bool
	UseTLS  bool
}

type Config struct {
	Proxies   []Proxy
	TCPMux    TCPMux
	ServiceId string
}

// listenAndProxy listens, locally, on the proxy's specified Port. For each
// incoming connection a goroutine running the proxy method is created.
func (p *Proxy) listenAndProxy() error {
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

		go p.proxy(conn)
	}
}

// proxy takes an established local connection, Dials the remote address specified
// by the Proxy structure and then copies data to and from the resulting pair
// of endpoints.
func (p *Proxy) proxy(local net.Conn) {
	remoteAddr := p.Address
	if p.TCPMux {
		remoteAddr = fmt.Sprintf("%s:%d", strings.Split(remoteAddr, ":")[0], options.muxport)
	}

	var remote net.Conn
	var err error

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
func (mux TCPMux) muxConnection(conn net.Conn) {
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
func (mux *TCPMux) listenAndMux() {
	var l net.Listener
	var err error

	if mux.UseTLS == false {
		l, err = net.Listen("tcp", fmt.Sprintf(":%d", options.muxport))
	} else {
		cert, cerr := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
		if cerr != nil {
			log.Println("listenAndMux Error (tls.X509KeyPair): ", cerr)
			return
		}

		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		l, err = tls.Listen("tcp", fmt.Sprintf(":%d", options.muxport), &tlsConfig)
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

		go mux.muxConnection(conn)
	}
}

// parseConfig reads JSON encoded configuration information from the
// given file and returns a corresponding Config struct.
func parseConfig(rdr io.ReadCloser) (*Config, error) {
	config := &Config{}

	decoder := json.NewDecoder(rdr)
	if err := decoder.Decode(&config); err != nil {
		return nil, err
	}

	return config, nil
}

func main() {
	flag.Parse()

	if len(flag.Args()) <= 0 {
		flag.Usage()
		os.Exit(2)
	}

	config := Config{}
	config.TCPMux.Enabled = options.mux
	config.TCPMux.UseTLS = options.tls
	config.ServiceId = flag.Args()[0]

	if config.TCPMux.Enabled {
		go config.TCPMux.listenAndMux()
	}

	func() {
		client, err := NewLBClient(options.servicedEndpoint)
		if err != nil {
			log.Printf("Could not create a client to endpoint %s: %s", options.servicedEndpoint, err)
			return
		}
		defer client.Close()

		var endpoints []serviced.ApplicationEndpoint
		err = client.GetServiceEndpoints(config.ServiceId, &endpoints)
		if err != nil {
			log.Printf("Error getting application endpoints for service %s: %s", config.ServiceId, err)
			return
		}

		for _, endpoint := range endpoints {
			proxy := Proxy{}
			proxy.Name = fmt.Sprintf("%v", endpoint)
			proxy.Address = fmt.Sprintf("%s:%d", endpoint.HostIp, endpoint.Port)
			proxy.TCPMux = config.TCPMux.Enabled
			proxy.UseTLS = config.TCPMux.UseTLS
			go proxy.listenAndProxy()
		}
	}()

	if l, err := net.Listen("tcp", ":4321"); err == nil {
		l.Accept()
	}

	os.Exit(0)
}
