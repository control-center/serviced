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
	"errors"
	"fmt"
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk/registry2"
)

var ErrPortServerRunning = errors.New("port server is already running")

// PublicPortManager manages all the port servers for a particular ip address
type PublicPortManager struct {
	hostIP    string
	certFile  string
	keyFile   string
	onFailure func(portNumber string, err error)
	mu        *sync.Mutex
	ports     map[string]*PublicPortHandler
}

// NewPublicPortManager creates a new public port manager at a given ip address
func NewPublicPortManager(hostIP, certFile, keyFile string, onFailure func(portNumber string, err error)) *PublicPortManager {
	return &PublicPortManager{
		hostIP:    hostIP,
		certFile:  certFile,
		keyFile:   keyFile,
		onFailure: onFailure,
		mu:        &sync.Mutex{},
		ports:     make(map[string]*PublicPortHandler),
	}
}

// Enable implements starts the public port server at the address
func (m *PublicPortManager) Enable(portNumber, protocol string, useTLS bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get the port handler or create if it doesn't exist
	h, ok := m.ports[portNumber]
	if !ok {
		address := fmt.Sprintf("%s:%s", m.hostIP, portNumber)
		h = NewPublicPortHandler(address)
		m.ports[portNumber] = h
	}

	// start the port server
	if err := h.Serve(protocol, useTLS, m.certFile, m.keyFile); err != nil {
		m.onFailure(portNumber, err)
	}
}

// Disable stops the public port server at the address
func (m *PublicPortManager) Disable(portNumber string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get the port handler and stop it if it exists
	h, ok := m.ports[portNumber]
	if ok {
		h.Stop()
	}
}

// Set updates the exports for a particular port handler
func (m *PublicPortManager) Set(portNumber string, data []registry.ExportDetails) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.ports[portNumber]
	if ok {
		h.SetExports(data)
	} else {
		address := fmt.Sprintf("%s:%s", m.hostIP, portNumber)
		h = NewPublicPortHandler(address, data...)
		m.ports[portNumber] = h
	}
}

// PublicPortHandler manages the port server at a specific address
type PublicPortHandler struct {
	address string
	exports Exports
	cancel  chan struct{}
	wg      *sync.WaitGroup
}

// NewPublicPortHandler sets up a new public port at the given address
func NewPublicPortHandler(address string, data ...registry.ExportDetails) *PublicPortHandler {
	cancel := make(chan struct{})
	close(cancel)

	return &PublicPortHandler{
		address: address,
		exports: NewRoundRobinExports(data), // round-robin is the default
		cancel:  cancel,
		wg:      &sync.WaitGroup{},
	}
}

// Serve starts the port server at address
func (h *PublicPortHandler) Serve(protocol string, useTLS bool, certFile, keyFile string) error {
	logger := log.WithFields(log.Fields{
		"Address":  h.address,
		"Protocol": protocol,
		"UseTLS":   useTLS,
	})

	// don't start the server if it is already running
	select {
	case <-h.cancel:
		h.cancel = make(chan struct{})
	default:
		logger.Debug("Port server is already running")
		return ErrPortServerRunning
	}

	var tlsConfig *tls.Config
	if useTLS {

		// get the certificate
		certFile, keyFile = GetCertFiles(certFile, keyFile)
		cert, err := tls.LoadX509KeyPair(certFile, keyFile)
		if err != nil {
			logger.WithError(err).Debug("Could not set up certificate")
			return err
		}

		// cipher suites and tls min version change may not be needed with
		// golang 1.5:
		// https://github.com/golang/go/issues/10094
		// https://github.com/golang/go/issues/9364
		tlsConfig = &tls.Config{
			MinVersion:               utils.MinTLS("http"),
			PreferServerCipherSuites: true,
			CipherSuites:             utils.CipherSuites("http"),
			Certificates:             []tls.Certificate{cert},
		}

		logger.Debug("Set up tls certificate")
	}

	// start listening on the port with a non-tls connection
	listener, err := net.Listen("tcp", h.address)
	if err != nil {
		logger.WithError(err).Debug("Could not start TCP listener")
		return err
	}

	h.wg.Add(1)
	go func() {
		if protocol == "http" || protocol == "https" {
			ServeHTTP(h.cancel, h.address, protocol, listener, tlsConfig, h.exports)
		} else {
			ServeTCP(h.cancel, listener, tlsConfig, h.exports)
		}
		h.wg.Done()
	}()

	logger.Info("Started port server")
	return nil
}

// Stop shuts down the port server
func (h *PublicPortHandler) Stop() {
	select {
	case <-h.cancel:
	default:
		close(h.cancel)
	}
	h.wg.Wait()
}

// SetExports updates the export list for the port handler
func (h *PublicPortHandler) SetExports(data []registry.ExportDetails) {
	h.exports.Set(data)
}
