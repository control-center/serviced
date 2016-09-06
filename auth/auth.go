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

	// Devs: Feel free to add more, or replace those above, but define errors in a nice well-known place
	// TODO: Remove this comment
)

// KeyStore represents storage of public/private key pairs, as well as access
// to the CC master's public key.
type KeyStore interface {
	PublicKey() crypto.PublicKey
	MasterPublicKey() crypto.PublicKey
	PrivateKey() crypto.PrivateKey
}

// Identity represents the identity of a host. The most-used implementation
// will involve serializing this to a token.
type Identity interface {
	Verify() bool
	Expired() bool
	HostID() string
	PoolID() string
	HasAdminAccess() bool
	HasDFSAccess() bool
	PublicKey() crypto.PublicKey
}
