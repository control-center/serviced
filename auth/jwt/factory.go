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

// Package jwt implements Zenoss-specific JWT facilities
package jwt

import (
	"fmt"

	jwtgo "github.com/dgrijalva/jwt-go"
)

// NewInstance returns a new instance of JWT
func NewInstance(algorithm string, keyLookup KeyLookupFunc) (JWT, error) {
	// TODO: support other algorithms ... someday ... maybe
	if algorithm != DefaultSigningAlgorithm {
		return nil, fmt.Errorf("algorithm %q is invalid; must be one of [%q]", algorithm, DefaultSigningAlgorithm)
	} else if keyLookup == nil {
		return nil, fmt.Errorf("keyLookup can not be nil")
	}

	signerInstance := &signerFacade{
		signingMethod: jwtgo.GetSigningMethod(algorithm),
	}

	jwt := &jwtFacade{}
	jwt.RegisterSigner(signerInstance)
	jwt.RegisterKeyLookup(keyLookup)

	return jwt, nil
}
