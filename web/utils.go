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

// GetRemoteConnection returns a connection to a remote address
func GetRemoteConnection(useTLS bool, export *registry.ExportDetails) (remote net.Conn, err error) {
	isLocalAddress := IsLocalAddress(export.HostIP)

	if isLocalAddress {
		// if the address is local return the connection
		address := fmt.Sprintf("%s:%d", export.PrivateIP, export.PortNumber)
		return net.Dial("tcp4", address)
	}

	// set up the remote address
	remoteAddress := fmt.Sprintf("%s:%d", export.HostIP, export.MuxPort)
	if useTLS {
		config := tls.Config{InsecureSkipVerify: true}
		remote, err = tls.Dial("tcp4", remoteAddress, &config)
	} else {
		remote, err = net.Dial("tcp4", remoteAddress)
	}

	// Prevent a panic if we couldn't connect to the mux.
	if err != nil {
		return nil, err
	}

	// set the muxHeader on the remote connection
	muxHeader, err := utils.PackTCPAddress(export.PrivateIP, export.PortNumber)
	if err != nil {
		return nil, err
	}
	remote.Write(muxHeader)

	return
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
