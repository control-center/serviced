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
)

var cipherLookup map[string]uint16

var tlsCiphers []uint16

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

	for key, _ := range cipherLookup {
		defaultCiphers = append(defaultCiphers, key)
	}
	SetCiphers(defaultCiphers)
}

func SetCiphers(ciphers []string) error {
	newCiphers := make([]uint16, 0, len(ciphers))
	for _, cipherName := range ciphers {
		if cipher, ok := cipherLookup[cipherName]; !ok {
			return fmt.Errorf("unknown cipher %s", cipher)
		} else {
			newCiphers = append(newCiphers, cipher)
		}
	}
	tlsCiphers = newCiphers
	return nil
}

// GetDefaultCiphers returns the default tls ciphers
func GetDefaultCiphers() []string {
	return defaultCiphers
}

func MinTLS() uint16 {
	return tls.VersionTLS10
}

func CipherSuites() []uint16 {
	return tlsCiphers
}
