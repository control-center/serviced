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

package auth

import (
	"errors"

	"github.com/control-center/serviced/logging"
)

const (
	TOKEN_LEN_BYTES = 4
	SIGNATURE_BYTES = 256
)

var (
	// ErrIdentityTokenInvalid is a generic invalid token error
	ErrIdentityTokenInvalid = errors.New("Identity token is invalid")
	// ErrIdentityTokenExpired is thrown when an identity token is expired
	ErrIdentityTokenExpired = errors.New("Identity token expired")
	// ErrIdentityTokenNotValidYet is thrown when an identity token is used before its issue time
	ErrIdentityTokenNotValidYet = errors.New("Identity token used before issue time")
	// ErrIdentityTokenBadSig is thrown when an identity token has a bad signature
	ErrIdentityTokenBadSig = errors.New("Identity token signature cannot be verified")
	// ErrNoPublicKey is thrown when no public key is available to verify a signature
	ErrNoPublicKey = errors.New("Cannot retrieve public key to verify signature")
	// ErrNoPrivateKey is thrown when no private key is available to sign a message
	ErrNoPrivateKey = errors.New("Cannot retrieve private key to sign message")
	// ErrInvalidSigningMethod is thrown when an identity token is not signed with the correct method
	ErrInvalidSigningMethod = errors.New("Identity token signing method was not RSAPSS")
	// ErrInvalidIdentityTokenClaims is thrown when an identity token does not have required claims
	ErrInvalidIdentityTokenClaims = errors.New("Identity token is missing required claims")
	// ErrNotRSAPublicKey is thrown when a key is not an RSA public key and needs to be
	ErrNotRSAPublicKey = errors.New("Not an RSA public key")
	// ErrNotRSAPrivateKey is thrown when a key is not an RSA private key and needs to be
	ErrNotRSAPrivateKey = errors.New("Not an RSA private key")
	// ErrNotPEMEncoded is thrown when bytes are not PEM encoded and need to be
	ErrNotPEMEncoded = errors.New("Not PEM encoded")
	// ErrBadKeysFile is thrown when the local keys file isn't parseable
	ErrBadKeysFile = errors.New("Unable to read security keys file")
	// ErrNotAuthenticated is thrown when there's no authentication token
	ErrNotAuthenticated = errors.New("No authentication token available")
	// ErrBadToken is thrown when there is a problem extracting a token from a data stream (e.g. mux, rpc, etc)
	ErrBadToken = errors.New("Could not extract token")
	// ErrRestTokenExpired is thrown when an rest token is expired
	ErrRestTokenExpired = errors.New("Rest token expired")
	// ErrBadRestToken is thrown when the rest token cant be extracted or parsed
	ErrBadRestToken = errors.New("Invalid rest token")
	// ErrRestTokenBadSig is thrown when a rest token has a bad signature
	ErrRestTokenBadSig = errors.New("Rest token signature cannot be verified")
	// ErrSSHFailed is thrown when we can't ssh to a remote host to register keys
	ErrSSHFailed = errors.New("Unable to make an ssh connection to host")
	// ErrAuth0TokenExpired is thrown when an auth0 token is expired
	ErrAuth0TokenExpired = errors.New("auth0 token expired")
	// ErrAuth0TokenBadIssuer is thrown when the issuer claim in an auth0 token does not match the value configured in the API
	ErrAuth0TokenBadIssuer = errors.New("auth0 token issuer does not match value configured in API")
	// ErrAuth0TokenBadAudience is thrown when the audience claim in an auth0 token does not match the value of the target API
	ErrAuth0TokenBadAudience = errors.New("auth0 token audience does not match the target API")

	log = logging.PackageLogger()
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
	Verifier() (Verifier, error)
}
