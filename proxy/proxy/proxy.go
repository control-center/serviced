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
package main

import (
	"flag"
	"fmt"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/proxy"
	"log"
	"net"
	"os"
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
	flag.BoolVar(&options.mux, "mux", false, "enable port multiplexing")
	flag.BoolVar(&options.tls, "tls", true, "enable TLS")
	flag.StringVar(&options.keyPEMFile, "keyfile", "", "path to private key file (defaults to compiled in private key)")
	flag.StringVar(&options.certPEMFile, "certfile", "", "path to public certificate file (defaults to compiled in public cert)")
	flag.StringVar(&options.servicedEndpoint, "endpoint", "127.0.0.1:4979", "serviced endpoint address")
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "\nUsage: proxy [OPTIONS] SERVICE_ID\n\n")
		flag.PrintDefaults()
	}
}

func main() {
	flag.Parse()

	if len(flag.Args()) <= 0 {
		flag.Usage()
		os.Exit(2)
	}

	config := proxy.Config{}
	config.TCPMux.Enabled = options.mux
	config.TCPMux.UseTLS = options.tls
	config.ServiceId = flag.Args()[0]

	if config.TCPMux.Enabled {
		go config.TCPMux.ListenAndMux()
	}

	func() {
		client, err := proxy.NewLBClient(options.servicedEndpoint)
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
			proxy := proxy.Proxy{}
			proxy.Name = fmt.Sprintf("%v", endpoint)
			proxy.Address = fmt.Sprintf("%s:%d", endpoint.HostIp, endpoint.Port)
			proxy.TCPMux = config.TCPMux.Enabled
			proxy.UseTLS = config.TCPMux.UseTLS
			proxy.Port = endpoint.ServicePort
			go proxy.ListenAndProxy()
		}
	}()

	if l, err := net.Listen("tcp", ":4321"); err == nil {
		l.Accept()
	}

	os.Exit(0)
}
