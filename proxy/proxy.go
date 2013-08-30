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
            "UseTLS" : <true | false>,
        }
        "Proxies": [
            { "Name": <service name>, "Address": "n.n.n.n:nnnn", "TCPMux": <true | false>, "UseTLS": <true | false>, "Port": nnnn },
            { "Name": <service name>, "Address": "n.n.n.n:nnnn", "TCPMux": <true | false>, "UseTLS": <true | false>, "Port": nnnn },
        ],
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
	"io"
	"log"
	"net"
	"net/textproto"
	"os"
	"strconv"
	"strings"
)

const muxport = 22250

var (
	configFileName = flag.String("config", "/dev/null", "proxy configuration file")

	proxyCertPEM = `-----BEGIN CERTIFICATE-----
MIICaDCCAdGgAwIBAgIJAMsgJclpgZqTMA0GCSqGSIb3DQEBBQUAME0xCzAJBgNV
BAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQ8wDQYDVQQK
DAZaZW5vc3MxDDAKBgNVBAsMA0RldjAeFw0xMzA4MzAyMTE0MTBaFw0yMzA4Mjgy
MTE0MTBaME0xCzAJBgNVBAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwG
QXVzdGluMQ8wDQYDVQQKDAZaZW5vc3MxDDAKBgNVBAsMA0RldjCBnzANBgkqhkiG
9w0BAQEFAAOBjQAwgYkCgYEAyY8M1eXgU+QJYyg/X3zKOfZf2NKOC1PEFzCJ9EUz
0tMkArHKCm3yid7Y2Jci2BMGlPKSgbp3wTGc32ONtSYxBOx7musmqgmD1LIADToL
UGaXPiolmMpv+GstaMFqpkWYfNCtlnzcTquMN+1jfKOd8+Ultodu4bZL4CJygfai
KRUCAwEAAaNQME4wHQYDVR0OBBYEFEes0lhAiq/5hAh01VxmE/eqqo2QMB8GA1Ud
IwQYMBaAFEes0lhAiq/5hAh01VxmE/eqqo2QMAwGA1UdEwQFMAMBAf8wDQYJKoZI
hvcNAQEFBQADgYEASJhY7kmME5Xv3k58C5IJXuVDCTJ1O/liHTCqzk3/GTvdvfKg
NiSsD6AUC/PVunaTs6ivwEFXcz7HFd94jsLfnEbfQ+tsTzct72vLknORxsuwAxpL
hXBOYfF12lYGYNlRN1HKFLSXysyHwCcWtGz886EUwzUWeCKOm7YGHYHUBaY=
-----END CERTIFICATE-----`

	proxyKeyPEM = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBAMmPDNXl4FPkCWMo
P198yjn2X9jSjgtTxBcwifRFM9LTJAKxygpt8one2NiXItgTBpTykoG6d8ExnN9j
jbUmMQTse5rrJqoJg9SyAA06C1Bmlz4qJZjKb/hrLWjBaqZFmHzQrZZ83E6rjDft
Y3yjnfPlJbaHbuG2S+AicoH2oikVAgMBAAECgYEAoQVK98aBhAN1DGYm2p3S4KNW
xtzO5XWx/eSlESQH1rEe35gxFEvpqwMAsWdsSrpIU83GBSV2bjy4Wi4qE0HDfgJ2
m3/IKGISTRyUZrnXprj1eIpwHbR5lhcKebohvtZeALKFH/8xdun0YzMkfJ4B2kxQ
7j28BBjKaowrBGOUeoECQQD83VkkJRCIVxxuSp+pvCJE2P9g6+j9NXlVcz3C0W0d
jJMIeVBQnqjy6bqpWqYwPCcroZ34Krc/o7OZAri2l+HdAkEAzA7YWIrDDih6x0BL
y4A/3kkGPj119u40woXicw1HMuW4X/zzXGfxHynO7KYqrTREKEJtBPUuGPC4JtXH
z0gcmQJAVyITEIBxJPoXgu3V/NAmYuD/hy9jlrUxfT97vcEav37sP5RGF7HEeAgQ
WUEyWRaxTLihTZ2yjYxkW8pzSgAmRQJBAJz5QoaCYGCQ1TpYBLaMdxVZWZshjpCh
WCbX9YaKDV5jBz2YCeHo970AXXUAss3A6jmKN/FbZtW6v/7n76hOEekCQQCV3ZhU
lhu+Iu4HUZGgpDg6tgnlB5Tv7zuyUlzPXgbNAsIsTvQfnmWa1/WpOvNOy2Ix5aJB
sl9SYPJBOM7G8o1p
-----END PRIVATE KEY-----`
)

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
	Proxies []Proxy
	TCPMux  TCPMux
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
		remoteAddr = fmt.Sprintf("%s:%d", strings.Split(remoteAddr, ":")[0], muxport)
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
		l, err = net.Listen("tcp", fmt.Sprintf(":%d", muxport))
	} else {
		cert, cerr := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
		if cerr != nil {
			log.Println("listenAndMux Error (tls.X509KeyPair): ", cerr)
			return
		}

		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		l, err = tls.Listen("tcp", fmt.Sprintf(":%d", muxport), &tlsConfig)
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

	if *configFileName == "/dev/null" {
		fmt.Fprintf(os.Stderr, "usage: %s [flags]\n", os.Args[0])
		flag.PrintDefaults()
		os.Exit(2)
	}

	configFile, err := os.Open(*configFileName)
	if err != nil {
		log.Fatal(err)
	}

	config, err := parseConfig(configFile)
	if err != nil {
		log.Fatal(err)
	}

	for i, _ := range config.Proxies {
		go config.Proxies[i].listenAndProxy()
	}

	if config.TCPMux.Enabled {
		go config.TCPMux.listenAndMux()
	}

	if l, err := net.Listen("tcp", ":4321"); err == nil {
		l.Accept()
	}

	os.Exit(0)
}
