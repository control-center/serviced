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

package web

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/control-center/serviced/proxy"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/dao"
	domainService "github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk"
	"github.com/control-center/serviced/zzk/registry"
	"github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
)

var (
	allportsLock sync.RWMutex
	allports     map[string]chan bool // map of port number to channel that destroys the server
	cpDao        dao.ControlPlane
)

func init() {
	allports = make(map[string]chan bool)
}

// Removes the port from our local cache and updates the service so the UI will flip to "disabled".
//  Only needs to be called if the port is being disabled unexpectedly due to an error
func disablePort(node service.ServicePublicEndpointNode) {
	//TODO: Add control plane methods to enable/disable public endpoints so we don't have to do a GetService and then UpdateService

	// remove the port from our local cache
	delete(allports, node.Name)

	// find the endpoint that matches this port number for this service (there will only be 1)
	var myService domainService.Service
	var myEndpoint domainService.ServiceEndpoint
	var unused int
	cpDao.GetService(node.ServiceID, &myService)
	for _, endpoint := range myService.Endpoints {
		for _, endpointPort := range endpoint.PortList {
			if endpointPort.PortAddr == node.Name {
				myEndpoint = endpoint
			}
		}
	}

	// disable port
	myService.EnablePort(myEndpoint.Name, node.Name, false)
	if err := cpDao.UpdateService(myService, &unused); err != nil {
		glog.Errorf("Error in disablePort(%s:%s): %v", node.ServiceID, node.Name, err)
	}
}

func (sc *ServiceConfig) ServePublicPorts(shutdown <-chan (interface{}), dao dao.ControlPlane) {
	cpDao = dao
	go sc.syncAllPublicPorts(shutdown)
}

// tcpKeepAliveListener sets TCP keep-alive timeouts on accepted
// connections. It's used by ListenAndServe and ListenAndServeTLS so
// dead TCP connections (e.g. closing laptop mid-download) eventually
// go away.
// Adapted from golang net/http/server.go
type tcpKeepAliveListener struct {
	*net.TCPListener
	StopChan chan bool
	port     string
}

func newKeepAliveListener(listener net.Listener, stopChan chan bool, port string) *tcpKeepAliveListener {
	return &tcpKeepAliveListener{listener.(*net.TCPListener), stopChan, port}
}

func (ln tcpKeepAliveListener) Accept() (c net.Conn, err error) {
	for {
		ln.SetDeadline(time.Now().Add(time.Second))
		select {
		case <-ln.StopChan:
			glog.V(2).Infof("Keep Alive listener port closing for port %s", ln.port)
			return nil, keepAliveListenerError{fmt.Errorf("Port closed for port %s", ln.port)}
		default:
		}

		conn, err := ln.AcceptTCP()
		if err != nil {
			switch err := err.(type) {
			case net.Error:
				if err.Timeout() {
					continue
				}
			}
			return nil, err
		}
		conn.SetKeepAlive(true)
		conn.SetKeepAlivePeriod(3 * time.Minute)

		// and return the connection.
		return conn, nil
	}
}

type keepAliveListenerError struct {
	error
}

// For HTTPS connections, we need to inject a header for downstream servers.
func (sc *ServiceConfig) createPortHttpServer(node service.ServicePublicEndpointNode,
	listener net.Listener, tlsConfig *tls.Config, stopChan chan bool) error {
	port := node.Name
	proto := node.Protocol
	portClosed := false

	// Setup a handler for the port http(s) endpoint.  This differs from the
	// handler for vhosts.
	httphandler := func(w http.ResponseWriter, r *http.Request) {
		// Notify any active connections that the endpoint is not available if they refresh the browser.
		if portClosed {
			// Listener.Close() stops listening but does not close active connections.
			// https://github.com/golang/go/blob/b6b4004d5a5bf7099ac9ab76777797236da7fe63/src/net/tcpsock.go#L229-230
			// Sending a response code doesn't close the connection; see the next comment.
			//http.Error(w, fmt.Sprintf("public endpoint %s not available", port), http.StatusServiceUnavailable)

			// We have to close this connection.  The browser will reuse the active connection, so if a
			// user connects, then the endpoint is stopped - that connection will get a port closed notice. Even
			// if the endpoint is restarted, the browser will reuse the connection to the closed listener.  This
			// ensures that they reconnect on the new connection each time they refresh the browser.
			w.Header().Set("Connection", "close")
			return
		}
		glog.V(2).Infof("httphandler (port) handling request: %+v", r)

		pepKey := registry.GetPublicEndpointKey(node.Name, node.Type)
		pepEP, err := sc.getPublicEndpoint(string(pepKey))
		if err != nil {
			http.Error(w, err.Error(), http.StatusNotFound)
			return
		}

		rp := sc.getReverseProxy(pepEP.hostIP, sc.muxPort, pepEP.privateIP, pepEP.epPort, sc.muxTLS && (sc.muxPort > 0))
		glog.V(1).Infof("Time to set up %s public endpoint proxy for %v", pepKey, r.URL)

		// Set up the X-Forwarded-Proto header so that downstream servers know
		// the request originated as HTTPS.
		if _, found := r.Header["X-Forwarded-Proto"]; !found {
			r.Header.Set("X-Forwarded-Proto", proto)
		}

		rp.ServeHTTP(w, r)
		return
	}

	// Create a new port server with a default handler.
	glog.V(2).Infof("Creating server mux for %s", port)
	portServer := http.NewServeMux()
	portServer.HandleFunc("/", httphandler)

	// HTTPS requires configuring the certificates for TLS.
	glog.V(0).Infof("Starting port endpoint %s server for port: %s", proto, port)
	go func() {
		glog.V(2).Infof("Creating %s server for port: %s", proto, port)
		server := &http.Server{Addr: port, Handler: portServer}

		// Create a keep alive listener with a stopChan
		keepAliveListener := newKeepAliveListener(listener.(*net.TCPListener), stopChan, port)

		// If we're using TLS we need to wrap the connection.
		if tlsConfig != nil {
			glog.V(2).Infof("Configuring port %s for TLS", port)
			listener = tls.NewListener(keepAliveListener, tlsConfig)
		}

		// The server.Serve() method will block.
		go func() {
			glog.V(2).Infof("Calling server.Serve(listener): %s port %s", proto, port)
			server.Serve(listener)
		}()

		glog.V(2).Infof("Waiting for stopChan: %s port %s", proto, port)
		<-stopChan
		glog.V(1).Infof("Closing the listener: %s port %s", proto, port)
		// Close the listener.
		listener.Close()
		portClosed = true
		glog.V(0).Infof("Closed %s port endpoint server for port: %s", proto, port)
	}()

	return nil
}

func (sc *ServiceConfig) createPublicPortServer(node service.ServicePublicEndpointNode, stopChan chan bool, shutdown <-chan (interface{})) error {
	port := node.Name
	useTLS := node.UseTLS
	proto := node.Protocol

	// Declare our listener..
	var listener net.Listener
	var err error
	var tlsConfig *tls.Config

	glog.V(1).Infof("About to listen on port %s; UseTLS=%t", port, useTLS)

	if useTLS {
		// Gather our certs files and handle the error.
		certFile, keyFile, err := sc.getCertFiles()
		if err != nil {
			glog.Errorf("Error getting certificates for TLS port %s: %s", port, err)
			disablePort(node)
			return err
		}

		// Create our certificate from the cert files (strings).
		glog.V(2).Infof("Loading certs from %s, %s", certFile, keyFile)
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			glog.Errorf("Could not set up tls certificate for public endpoint on port %s for %s: %s", port, node.ServiceID, err)
			disablePort(node)
			return err
		}

		// The list of certs to use for our secure listener on this port.
		certs := []tls.Certificate{cert}

		// This cipher suites and tls min version change may not be needed with golang 1.5
		// https://github.com/golang/go/issues/10094
		// https://github.com/golang/go/issues/9364
		tlsConfig = &tls.Config{
			MinVersion:               utils.MinTLS(),
			PreferServerCipherSuites: true,
			CipherSuites:             utils.CipherSuites(),
			Certificates:             certs,
		}
	}

	// Start listening on the port with a non-tls connection.
	listener, err = net.Listen("tcp", port)
	if err != nil {
		glog.Errorf("Could not setup TCP listener for port %s for public endpoint %s: %s", port, node.ServiceID, err)
		disablePort(node)
		return err
	}

	// If we're using http/https we setup a handler.
	if proto == "http" || proto == "https" {
		glog.V(0).Infof("Creating port %s server for port %s", proto, port)
		return sc.createPortHttpServer(node, listener, tlsConfig, stopChan)
	}

	glog.Infof("Listening on port %s; UseTLS=%t", port, useTLS)

	go func() {
		for {
			// accept connection on public port
			localConn, err := listener.Accept()
			if err != nil {
				glog.V(1).Infof("Stopping accept on port %s", port)
				return
			}

			// If we're using TLS we need to wrap the connection.
			if tlsConfig != nil {
				localConn = tls.Server(localConn, tlsConfig)
			}

			// lookup remote endpoint for this public port
			pepEPInfo, err := sc.getPublicEndpoint(fmt.Sprintf("%s-%d", node.Name, int(node.Type)))
			if err != nil {
				// This happens if an endpoint is accessed and the containers have died or not come up yet.
				glog.Errorf("Error retrieving public endpoint %s-%d: %s", node.Name, int(node.Type), err)
				// close the accepted connection and continue waiting for connections.
				if err := localConn.Close(); err != nil {
					glog.Errorf("Error closing client connection: %s", err)
				}
				continue
			}

			// setup remote connection
			var remoteAddr string
			_, isLocalContainer := sc.localAddrs[pepEPInfo.hostIP]
			if isLocalContainer {
				remoteAddr = fmt.Sprintf("%s:%d", pepEPInfo.privateIP, pepEPInfo.epPort)
			} else {
				remoteAddr = fmt.Sprintf("%s:%d", pepEPInfo.hostIP, sc.muxPort)
			}
			remoteConn, err := sc.getRemoteConnection(remoteAddr, isLocalContainer, sc.muxPort, pepEPInfo.privateIP, pepEPInfo.epPort, sc.muxTLS && (sc.muxPort > 0))
			if err != nil {
				glog.Errorf("Error getting remote connection for public endpoint %s-%d: %v", node.Name, int(node.Type), err)
				continue
			}

			glog.V(2).Infof("Established remote connection to %s", remoteConn.RemoteAddr())

			// Serve proxied requests/responses.  We pass our own port stop channel so that
			// all proxy loops end when our port is shutdown.
			go proxy.ProxyLoop(localConn, remoteConn, stopChan)
		}
	}()

	go func() {
		// Wait for shutdown, then kill all your connections
		select {
		case <-shutdown:
			// Received an application shutdown. Close the port channel to halt all proxy loops.
			glog.Infof("Shutting down port %s", port)
			close(stopChan)
		case <-stopChan:
		}

		listener.Close()
		glog.Infof("Closed port %s", port)
		return
	}()

	return nil
}

func (sc *ServiceConfig) syncAllPublicPorts(shutdown <-chan interface{}) error {
	rootConn, err := zzk.GetLocalConnection("/")
	if err != nil {
		glog.Errorf("syncAllPublicPorts - Error getting root zk connection: %v", err)
		return err
	}

	cancelChan := make(chan interface{})
	zkServicePEPService := service.ZKServicePublicEndpoints

	syncPorts := func(conn client.Connection, parentPath string, childIDs ...string) {
		allportsLock.Lock()
		defer allportsLock.Unlock()

		glog.V(1).Infof("syncPorts STARTING for parentPath:%s childIDs:%v", parentPath, childIDs)

		// start all servers that have been not started and enabled
		newPorts := make(map[string]chan bool)
		for _, pepID := range childIDs {

			// The pepID is the ZK child key. Get the node so we have all of the node data.
			glog.V(1).Infof("zkServicePEPService: %s, pepID: %s", zkServicePEPService, pepID)
			nodePath := fmt.Sprintf("%s/%s", zkServicePEPService, pepID)
			var node service.ServicePublicEndpointNode
			err := rootConn.Get(nodePath, &node)
			if err != nil {
				glog.Errorf("Unable to get the ZK Node from PepID")
				continue
			}

			if node.Type == registry.EPTypePort && node.Enabled {
				port := node.Name
				stopChan, running := allports[port]

				if !running {
					// recently enabled port - port should be opened
					stopChan = make(chan bool)
					if err := sc.createPublicPortServer(node, stopChan, shutdown); err != nil {
						continue
					}
				}

				newPorts[port] = stopChan
			}
		}

		// stop all servers that have been deleted or disabled
		for port, stopChan := range allports {
			_, found := newPorts[port]
			if !found {
				glog.V(2).Infof("Stopping port server for port %s", port)
				close(stopChan)
				glog.Infof("Port server shut down for port %s", port)
			}
		}

		allports = newPorts
		glog.V(2).Infof("Portserver allports: %+v", allports)
	}

	for {
		glog.V(1).Infof("Running registry.WatchChildren for zookeeper path: %s", zkServicePEPService)
		err := registry.WatchChildren(rootConn, zkServicePEPService, cancelChan, syncPorts, pepWatchError)
		if err != nil {
			glog.V(1).Infof("Will retry in 10 seconds to WatchChildren(%s) due to error: %v", zkServicePEPService, err)
			<-time.After(time.Second * 10)
			continue
		}
		select {
		case <-shutdown:
			close(cancelChan)
			return nil
		default:
		}
	}
}
