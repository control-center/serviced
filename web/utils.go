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
	"time"

	"github.com/control-center/serviced/auth"
	"github.com/control-center/serviced/logging"
	"github.com/control-center/serviced/proxy"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/zzk/registry"
)

// initialize the logger
var plog = logging.PackageLogger()

// ipmap keeps track of all ipv4 addresses on this host
var ipmap = make(map[string]struct{})

func init() {
	// set up the ipmap
	ips, err := utils.GetIPv4Addresses()
	if err != nil {
		plog.WithError(err).Fatal("Could not get interface ip addresses")
	}
	for _, ip := range ips {
		ipmap[ip] = struct{}{}
	}
}

// IsLocalAddress returns true if the ip address is available on this host
func IsLocalAddress(ip string) bool {
	_, ok := ipmap[ip]
	return ok
}

// GetCertFiles returns the cert and key file if none is specified
func GetCertFiles(certFile, keyFile string) (string, string) {

	// set the cert file if it doesn't exist
	if certFile == "" {
		var err error
		certFile, err = proxy.TempCertFile()
		if err != nil {
			plog.WithError(err).Fatal("Could not create temp cert file")
		}
	}

	// set the key file if it doesn't exist
	if keyFile == "" {
		var err error
		keyFile, err = proxy.TempKeyFile()
		if err != nil {
			plog.WithError(err).Fatal("Could not create temp key file")
		}
	}

	return certFile, keyFile
}

// Dialer interface to make getRemoteConnection testable.
type dialerInterface interface {
	Dial(string, string) (net.Conn, error)
}

type netDialer struct{}

func (d *netDialer) Dial(network, address string) (net.Conn, error) {
	return net.Dial(network, address)
}

func newNetDialer() dialerInterface {
	return &netDialer{}
}

type tlsDialer struct {
	config *tls.Config
}

func (d *tlsDialer) Dial(network, address string) (net.Conn, error) {
	return tls.Dial(network, address, d.config)
}

func newTlsDialer(config *tls.Config) dialerInterface {
	return &tlsDialer{config: config}
}

// GetRemoteConnection returns a connection to a remote address
func GetRemoteConnection(useTLS bool, export *registry.ExportDetails) (remote net.Conn, err error) {
	var dialer dialerInterface
	if useTLS {
		config := tls.Config{InsecureSkipVerify: true}
		dialer = newTlsDialer(&config)
	} else {
		dialer = newNetDialer()
	}
	return getRemoteConnection(export, dialer)
}

func getRemoteConnection(export *registry.ExportDetails, dialer dialerInterface) (net.Conn, error) {
	// If the exported endpoint is on this Host, we don't go through the mux.
	if IsLocalAddress(export.HostIP) {
		// if the address is local return a connection directly to the container
		address := fmt.Sprintf("%s:%d", export.PrivateIP, export.PortNumber)
		return dialer.Dial("tcp4", address)
	}

	// Set up the remote address for the mux
	remoteAddress := fmt.Sprintf("%s:%d", export.HostIP, export.MuxPort)
	remote, err := dialer.Dial("tcp4", remoteAddress)

	// Prevent a panic if we couldn't connect to the mux.
	if err != nil {
		return nil, err
	}

	// Set the muxHeader on the remote connection so it knows what service to
	// proxy the connection to.
	muxHeader, err := utils.PackTCPAddress(export.PrivateIP, export.PortNumber)
	if err != nil {
		return nil, err
	}

	var (
		token        string
		tokenTimeout = 30 * time.Second
	)

	select {
	case token = <-auth.AuthToken(nil):
	case <-time.After(tokenTimeout):
		plog.WithField("timeout", "30s").Error("Unable to retrieve authentication token within the timeout")
		return nil, err
	}

	muxHeader, err = auth.BuildAuthMuxHeader(muxHeader, token)
	if err != nil {
		plog.WithError(err).Error("Error building authenticated mux header.")
		return nil, err
	}

	// Check for errors writing the mux header.
	if _, err = remote.Write(muxHeader); err != nil {
		return nil, err
	}

	return remote, nil
}

// TCPKeepAliveListener keeps a listener connection alive for the duration
// of a cancellable
type TCPKeepAliveListener struct {
	*net.TCPListener
	cancel <-chan struct{}
}

// Accept returns the connection to the listener
func (ln TCPKeepAliveListener) Accept() (c net.Conn, err error) {
	for {
		ln.SetDeadline(time.Now().Add(time.Second))
		select {
		case <-ln.cancel:
			return nil, errors.New("port closed")
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
