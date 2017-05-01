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

package web

import (
	"crypto/tls"
	"net"
	"net/http"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/config"
	"github.com/control-center/serviced/proxy"
)

// ServeTCP sets up a tcp based server connection given a set of exports.
func ServeTCP(cancel <-chan struct{}, listener net.Listener, tlsConfig *tls.Config, exports Exports) {
	stopChan := make(chan bool)
	wg := &sync.WaitGroup{}

	go func() {
		for {
			local, err := listener.Accept()
			if err != nil {
				plog.WithError(err).Debug("Stopping accept on host:port")
				return
			}

			if tlsConfig != nil {
				local = tls.Server(local, tlsConfig)
			}

			export := exports.Next()
			if export == nil {
				// This happens if the endpoint is accessed and the containers
				// have died or not come up yet.
				plog.Warn("Could not retrieve endpoint")

				// close the accepted connection and continue waiting for
				// connections.
				if err := local.Close(); err != nil {
					plog.WithError(err).Error("Could not close client connection")
				}
				continue
			}

			logger := plog.WithFields(log.Fields{
				"application": export.Application,
				"hostip":      export.HostIP,
				"privateip":   export.PrivateIP,
			})

			// setup remote connection
			remote, err := GetRemoteConnection(config.MuxTLSIsEnabled(), export)
			if err != nil {
				logger.WithError(err).Error("Could not get remote connection for endpoint")
				continue
			}

			logger.WithField("remoteaddress", remote.RemoteAddr()).Debug("Established remote connection")

			wg.Add(1)
			go func() {
				proxy.ProxyLoop(local, remote, stopChan)
				wg.Done()
			}()
		}
	}()

	<-cancel
	listener.Close()
	close(stopChan)
	wg.Wait()
}

// ServeHTTP sets up an http server for handling a collection of endpoints
func ServeHTTP(cancel <-chan struct{}, address, protocol string, listener net.Listener, tlsConfig *tls.Config, exports Exports) {
	logger := plog.WithFields(log.Fields{
		"portaddress": address,
		"protocol":    protocol,
		"usetls":      tlsConfig != nil,
	})

	portClosed := make(chan struct{})

	// Setup a handler for the port http(s) endpoint.  This differs from the
	// handler for vhosts.
	httphandler := func(w http.ResponseWriter, r *http.Request) {
		// Notify any active connections that the endpoint is not available if
		// they refresh the browser.
		select {
		case <-portClosed:
			// Listener.Close() stops listening but does not close active
			// connections.
			// https://github.com/golang/go/blob/b6b4004d5a5bf7099ac9ab76777797236da7fe63/src/net/tcpsock.go#L229-230
			// Sending a response code doesn't close the connection; see the
			// next comment.
			// http.Error(w, fmt.Sprintf("public endpoint %s not available", port), http.StatusServiceUnavailable)

			// We have to close this connection.  The browser will reuse the
			// active connection, so if a user connects, then the endpoint is
			// stopped - that connection will get a port closed notice. Even if
			// the endpoint is restarted, the browser will reuse the connection
			// to the closed listener.  This ensures that they reconnect on the
			// new connection each time they refresh the browser.
			w.Header().Set("Connection", "close")
			return
		default:
		}

		logger.WithField("handlerrequest", r).Debug("Handler handling (port) request")

		export := exports.Next()
		if export == nil {
			http.Error(w, "endpoint not available", http.StatusNotFound)
			return
		}

		rp := GetReverseProxy(config.MuxTLSIsEnabled(), export)

		logger.WithFields(log.Fields{
			"application": export.Application,
			"hostip":      export.HostIP,
			"privateip":   export.PrivateIP,
			"url":         r.URL,
		}).Debug("Set up public endpoint proxy")

		// Set up the X-Forwarded-Proto header so that downstream servers know
		// the request originated as HTTPS.
		if _, found := r.Header["X-Forwarded-Proto"]; !found {
			r.Header.Set("X-Forwarded-Proto", protocol)
		}
		
		if tlsConfig != nil {
			w.Header().Add("Strict-Transport-Security","max-age=31536000")
		}
		rp.ServeHTTP(w, r)

		return
	}

	// Create a new port server with a default handler.
	portServer := http.NewServeMux()
	portServer.HandleFunc("/", httphandler)

	// HTTPS requires configuring the certificates for TLS.
	server := &http.Server{Addr: address, Handler: portServer}

	if tlsConfig != nil {
		keepAliveListener := &TCPKeepAliveListener{
			TCPListener: listener.(*net.TCPListener),
			cancel:      cancel,
		}
		listener = tls.NewListener(keepAliveListener, tlsConfig)
	}

	wg := &sync.WaitGroup{}
	wg.Add(1)
	go func() {
		server.Serve(listener)
		wg.Done()
	}()

	<-cancel
	listener.Close()
	close(portClosed)
	wg.Wait()
}
