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
	"net"
	"sync"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk/registry2"
)

var ErrPortServerRunning = errors.New("port server is already running")

// PublicPortManager manages all the port servers for a particular host id
type PublicPortManager struct {
	hostID    string
	certFile  string
	keyFile   string
	onFailure func(portNumber string, err error)
	mu        *sync.RWMutex
	ports     map[string]*PublicPortHandler
}

// NewPublicPortManager creates a new public port manager for a host id
func NewPublicPortManager(hostID, certFile, keyFile string, onFailure func(portAddr string, err error)) *PublicPortManager {
	return &PublicPortManager{
		hostID:    hostID,
		certFile:  certFile,
		keyFile:   keyFile,
		onFailure: onFailure,
		mu:        &sync.RWMutex{},
		ports:     make(map[string]*PublicPortHandler),
	}
}

// Enable implements starts the public port server at the port address
func (m *PublicPortManager) Enable(portAddr, protocol string, useTLS bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// get the port handler or create if it doesn't exist
	h, ok := m.ports[portAddr]
	if !ok {
		h = NewPublicPortHandler(portAddr)
		m.ports[portAddr] = h
	}

	// start the port server
	if err := h.Serve(protocol, useTLS, m.certFile, m.keyFile); err != nil {
		m.onFailure(portAddr, err)
	}
}

// Disable stops the public port server at the port address
func (m *PublicPortManager) Disable(portAddr string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// get the port handler and stop it if it exists
	h, ok := m.ports[portAddr]
	if ok {
		h.Stop()
	}
}

// Set updates the exports for a particular port handler
func (m *PublicPortManager) Set(portAddr string, data []registry.ExportDetails) {
	m.mu.Lock()
	defer m.mu.Unlock()

	h, ok := m.ports[portAddr]
	if ok {
		h.SetExports(data)
	} else {
		h = NewPublicPortHandler(portAddr, data...)
		m.ports[portAddr] = h
	}
}

// PublicPortHandler manages the port server at a specific port address
type PublicPortHandler struct {
	portAddr string
	exports  Exports
	cancel   chan struct{}
	wg       *sync.WaitGroup
}

// NewPublicPortHandler sets up a new public port at the given port address
func NewPublicPortHandler(portAddr string, data ...registry.ExportDetails) *PublicPortHandler {
	cancel := make(chan struct{})
	close(cancel)

	return &PublicPortHandler{
		portAddr: portAddr,
		exports:  NewRoundRobinExports(data), // round-robin is the default
		cancel:   cancel,
		wg:       &sync.WaitGroup{},
	}
}

// Serve starts the port server at address
func (h *PublicPortHandler) Serve(protocol string, useTLS bool, certFile, keyFile string) error {
	logger := log.WithFields(log.Fields{
		"portaddress": h.portAddr,
		"protocol":    protocol,
		"usetls":      useTLS,
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
	listener, err := net.Listen("tcp", h.portAddr)
	if err != nil {
		logger.WithError(err).Debug("Could not start TCP listener")
		return err
	}

	h.wg.Add(1)
	go func() {
		logger.Info("Starting port server")
		defer logger.Debug("Port server exited")

		if protocol == "http" || protocol == "https" {
			ServeHTTP(h.cancel, h.portAddr, protocol, listener, tlsConfig, h.exports)
		} else {
			ServeTCP(h.cancel, listener, tlsConfig, h.exports)
		}
		h.wg.Done()
	}()

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
