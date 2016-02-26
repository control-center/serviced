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
	"sort"
	"strings"
)

type ciphers []uint16

func (c ciphers) Len() int           { return len(c) }
func (c ciphers) Less(i, j int) bool { return c[i] < c[j] }
func (c ciphers) Swap(i, j int)      { c[i], c[j] = c[j], c[i] }

// DefaultTLSMinVersion minimum TLS version supported
const DefaultTLSMinVersion = "VersionTLS10"

var cipherLookup map[string]uint16

var tlsCiphers []uint16

var tlsVersion uint16

var defaultCiphers []string

func init() {
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
	}

	tlsCiphers = make([]uint16, 0)
	for key := range cipherLookup {
		defaultCiphers = append(defaultCiphers, key)
	}
	SetCiphers(defaultCiphers)
}

// GetDefaultCiphers returns the default tls ciphers
func GetDefaultCiphers() []string {
	return defaultCiphers
}

// SetCiphers that can be used
func SetCiphers(cipherNames []string) error {
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
	tlsCiphers = newCiphers
	// reverse sort the ciphers, since the uint value determines the order
	sort.Sort(sort.Reverse(ciphers(tlsCiphers)))
	return nil
}

// SetMinTLS the min tls that can be used
func SetMinTLS(version string) error {
	upperTLS := strings.ToUpper(strings.TrimSpace(version))
	switch upperTLS {
	case "VERSIONTLS10":
		tlsVersion = tls.VersionTLS10
	case "VERSIONTLS11":
		tlsVersion = tls.VersionTLS11
	case "VERSIONTLS12":
		tlsVersion = tls.VersionTLS12
	default:
		return fmt.Errorf("Invalid TLS version %s", version)

	}
	return nil
}

// MinTLS the min tls that can be used
func MinTLS() uint16 {
	return tlsVersion
}

// CipherSuites the ciphers that can be sued
func CipherSuites() []uint16 {
	return tlsCiphers
}
