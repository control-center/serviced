// Copyright 2015 The Serviced Authors.
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

package utils

import (
	"crypto/tls"
	"fmt"
	"strings"

	"github.com/zenoss/glog"
)

//---------------------------------------------------------------------------
// This file maintains a set of default TLS configuration options for each
// of the 3 different kinds of listening ports (HTTP, RPC and MUX).
//
// Default values may be changed by calling one of the Set methods in this
// file.
//---------------------------------------------------------------------------

// DefaultTLSMinVersion minimum TLS version supported
const DefaultTLSMinVersion = "VersionTLS10"

var cipherLookup map[string]uint16

// TODO: Add options to separate certs and keys by connection type?
type configInfo struct {
	name           string
	minTLSVersion  uint16
	cipherSuite    []uint16
	defaultCiphers []string
}

var configMap map[string]*configInfo

func init() {
	//
	// FYI - In our lab testing for CC-2512/CC-2514, these ciphers did not work at all
	// (e.g. tls handshake failures and/or Chrome connection failures), but they are left in the
	// list just in case; e.g. either a browser adds support for them, or a newer version of
	// GO TLS supports them (not sure which side caused the connection failure).
	//
	//       tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256
	//       tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA
	//       tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA
	//       tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA
	cipherLookup = map[string]uint16{
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256":   tls.TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256": tls.TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256,
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA":      tls.TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA":    tls.TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA,
		"TLS_RSA_WITH_AES_128_CBC_SHA":            tls.TLS_RSA_WITH_AES_128_CBC_SHA,
		"TLS_RSA_WITH_AES_256_CBC_SHA":            tls.TLS_RSA_WITH_AES_256_CBC_SHA,
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA":     tls.TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA":           tls.TLS_RSA_WITH_3DES_EDE_CBC_SHA,
		"TLS_RSA_WITH_RC4_128_SHA":                tls.TLS_RSA_WITH_RC4_128_SHA,
		"TLS_RSA_WITH_AES_128_GCM_SHA256":         tls.TLS_RSA_WITH_AES_128_GCM_SHA256,
		"TLS_RSA_WITH_AES_256_GCM_SHA384":         tls.TLS_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA":        tls.TLS_ECDHE_ECDSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA":          tls.TLS_ECDHE_RSA_WITH_RC4_128_SHA,
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384":   tls.TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384,
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384": tls.TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384,
	}

	// All supported ciphers in descending order with (roughly) newest first. Note that in GO 1.6 the HTTP2
	// server will fail to start if some of the ciphers lower in the list appear before several which
	// are higher in the list or if ECDHE_RSA_WITH_AES_128_GCM_SHA256 is NOT in the list.
	// See ConfigureServer() and isBadCipher() in https://github.com/golang/net/blob/master/http2/server.go
	httpDefaultCiphers := []string{
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_ECDSA_WITH_AES_128_GCM_SHA256",
		"TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_3DES_EDE_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_ECDSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
		"TLS_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_3DES_EDE_CBC_SHA",
		"TLS_RSA_WITH_RC4_128_SHA",
		"TLS_RSA_WITH_AES_128_GCM_SHA256",
		"TLS_RSA_WITH_AES_256_GCM_SHA384",
		"TLS_ECDHE_ECDSA_WITH_RC4_128_SHA",
		"TLS_ECDHE_RSA_WITH_RC4_128_SHA",
	}

	// For RPC/MUX communication, we want the fastest ciphers possible.
	//
	// CC-2512/CC-2514 - based on testing in our lab, some ciphers have terrible performance and some
	//                   do not work with HTTP, RPC or MUX communications. This list is ordered by most-performant
	//                   to least performant based on our tests.
	//
	rpcDefaultCiphers := []string{
		"TLS_ECDHE_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_128_CBC_SHA",
		"TLS_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_256_CBC_SHA",
		"TLS_ECDHE_RSA_WITH_AES_128_GCM_SHA256",
	}

	muxDefaultCiphers := rpcDefaultCiphers
	tlsVersion, _ := tlsVersionStringToUint(DefaultTLSMinVersion)

	configMap = make(map[string]*configInfo, 0)
	for _, connectionType := range []string{"http", "rpc", "mux"} {
		config := &configInfo{
			name:          connectionType,
			minTLSVersion: tlsVersion,
		}
		switch connectionType {
		case "http":
			config.defaultCiphers = httpDefaultCiphers
		case "rpc":
			config.defaultCiphers = rpcDefaultCiphers
		case "mux":
			config.defaultCiphers = muxDefaultCiphers
		}

		configMap[config.name] = config
		SetCiphers(config.name, config.defaultCiphers)
	}
}

// GetDefaultCiphers returns the default tls ciphers
func GetDefaultCiphers(connectionType string) []string {
	configInfo, ok := configMap[connectionType]
	if !ok {
		glog.Fatalf("connectionType %s is undefined", connectionType)
	}
	return configInfo.defaultCiphers
}

// SetCiphers that can be used
func SetCiphers(connectionType string, cipherNames []string) error {
	configInfo, ok := configMap[connectionType]
	if !ok {
		glog.Fatalf("connectionType %s is undefined", connectionType)
	}

	newCiphers := make([]uint16, 0, len(cipherNames))
	for _, cipherName := range cipherNames {
		upperCipher := strings.ToUpper(strings.TrimSpace(cipherName))
		var cipher uint16
		var ok bool
		if cipher, ok = cipherLookup[upperCipher]; !ok {
			return fmt.Errorf("unknown cipher %s", cipherName)
		}

		newCiphers = append(newCiphers, cipher)

	}
	configInfo.cipherSuite = newCiphers
	return nil
}

// SetMinTLS the min tls that can be used
func SetMinTLS(connectionType string, version string) error {
	configInfo, ok := configMap[connectionType]
	if !ok {
		glog.Fatalf("connectionType %s is undefined", connectionType)
	}

	tlsVersion, err := tlsVersionStringToUint(version)
	if err != nil {
		return fmt.Errorf("Invalid TLS version %s", version)
	}

	configInfo.minTLSVersion = tlsVersion
	return nil
}

func tlsVersionStringToUint(version string) (uint16, error) {
	upperTLS := strings.ToUpper(strings.TrimSpace(version))
	switch upperTLS {
	case "VERSIONTLS10":
		return tls.VersionTLS10, nil
	case "VERSIONTLS11":
		return tls.VersionTLS11, nil
	case "VERSIONTLS12":
		return tls.VersionTLS12, nil
	default:
		return 0, fmt.Errorf("Invalid TLS version %s", version)
	}
}

// MinTLS the min tls version that can be used for a given connection type
func MinTLS(connectionType string) uint16 {
	configInfo, ok := configMap[connectionType]
	if !ok {
		glog.Fatalf("connectionType %s is undefined", connectionType)
	}
	return configInfo.minTLSVersion
}

// CipherSuites the ciphers that can be sued
func CipherSuites(connectionType string) []uint16 {
	configInfo, ok := configMap[connectionType]
	if !ok {
		glog.Fatalf("connectionType %s is undefined", connectionType)
	}
	return configInfo.cipherSuite
}

func CipherSuitesByName(c *tls.Config) []string {
	suiteList := make([]string, 0)
	for _, cipher := range c.CipherSuites {
		suiteList = append(suiteList, fmt.Sprintf("%s (%d)", GetCipherName(cipher), cipher))
	}
	return suiteList
}

// Get the name of the cipher
func GetCipherName(cipher uint16) string {
	for key, value := range cipherLookup {
		if cipher == value {
			return key
		}
	}
	return "unsupported"
}
