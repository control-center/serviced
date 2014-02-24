package serviced

import (
	"bufio"
	"crypto/tls"
	"fmt"
	"github.com/zenoss/glog"
	"io"
	"net"
	"net/textproto"
	"strconv"
	"strings"
)

type TCPMux struct {
	Enabled     bool
	UseTLS      bool
	CertPEMFile string
	KeyPEMFile  string
	Port        int
}

type MuxConfig struct {
	Proxies   []Proxy
	TCPMux    TCPMux
	ServiceId string
	Command   string
}

// sendMuxError logs an error message and attempts to write it to the connected
// endpoint
func sendMuxError(conn net.Conn, source, facility, msg string, err error) {
	glog.Errorf("%s Error (%s): %v\n", source, facility, err)
	if _, e := conn.Write([]byte(msg)); e != nil {
		glog.Errorf("%s", e)
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
		sendMuxError(conn, "MuxConnection", "textproto.ReadMIMEHeader", "bad request (no headers)", err)
		conn.Close()
		return
	}

	zs, ok := hdr["Zen-Service"]
	if ok == false {
		sendMuxError(conn, "MuxConnection", "MIMEHeader", "bad request (no Zen-Service header)", err)
		conn.Close()
		return
	}

	port, err := strconv.Atoi(strings.Split(zs[0], "/")[1])
	if err != nil {
		sendMuxError(conn, "MuxConnection", "Zen-Service Header", "bad Zen-Service spec", err)
		conn.Close()
		return
	}

	svc, err := net.Dial("tcp4", fmt.Sprintf("172.17.42.1:%d", port))
	if err != nil {
		sendMuxError(conn, "MuxConnection", "net.Dial", "cannot connect to service", err)
		conn.Close()
		return
	}

	go io.Copy(conn, svc)
	go io.Copy(svc, conn)
}

// ListenAndMux listens for incoming connections and attempts to multiplex them
// to the local service that they request via a Zen-Service header in their
// initial message.
func (mux *TCPMux) ListenAndMux() {
	var l net.Listener
	var err error

	if mux.UseTLS == false {
		l, err = net.Listen("tcp4", fmt.Sprintf(":%d", mux.Port))
	} else {
		cert, cerr := tls.X509KeyPair([]byte(proxyCertPEM), []byte(proxyKeyPEM))
		if cerr != nil {
			glog.Error("ListenAndMux Error (tls.X509KeyPair): ", cerr)
			return
		}

		tlsConfig := tls.Config{Certificates: []tls.Certificate{cert}}
		l, err = tls.Listen("tcp4", fmt.Sprintf(":%d", mux.Port), &tlsConfig)
	}
	if err != nil {
		glog.Error("ListenAndMux Error (net.Listen): ", err)
		return
	}
	defer l.Close()

	for {
		conn, err := l.Accept()
		if err != nil {
			glog.Error("ListenAndMux Error (net.Accept): ", err)
			return
		}

		go mux.MuxConnection(conn)
	}
}
