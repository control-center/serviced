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

package auth

import (
	"crypto"
	"errors"
)

var (
	// ErrIdentityTokenExpired is thrown when an identity token is expired
	ErrIdentityTokenExpired = errors.New("Identity token expired")
	// ErrIdentityTokenBadSig is thrown when an identity token has a bad signature
	ErrIdentityTokenBadSig = errors.New("Identity token signature cannot be verified")
	// ErrNotRSAPublicKey is thrown when a key is not an RSA public key and needs to be
	ErrNotRSAPublicKey = errors.New("Not an RSA public key")
	// ErrNotRSAPrivateKey is thrown when a key is not an RSA private key and needs to be
	ErrNotRSAPrivateKey = errors.New("Not an RSA private key")
	// ErrNotPEMEncoded is thrown when bytes are not PEM encoded and need to be
	ErrNotPEMEncoded = errors.New("Not PEM encoded")

	// Devs: Feel free to add more, or replace those above, but define errors in a nice well-known place
	// TODO: Remove this comment
)

// Signer is used to sign a message
type Signer interface {
	Sign(message []byte) ([]byte, error)
}

// Verifier is used to verify a signed message
type Verifier interface {
	Verify(message []byte, signature []byte) error
}

// Identity represents the identity of a host. The most-used implementation
// will involve serializing this to a token.
type Identity interface {
	Valid() error
	Expired() bool
	HostID() string
	PoolID() string
	HasAdminAccess() bool
	HasDFSAccess() bool
	PublicKey() crypto.PublicKey
}

// TODO: Placeholder until we have the token available
func AuthToken() string {
    return "my super fake token"
}
