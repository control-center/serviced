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
	"time"
)

const (
	// DefaultSigningAlgorithm defines the default the signing algorithm.
	// The JWT specification defines the a of valid algorithms.
	// HS256 represents the HMAC SHA-256 algorithm.
	DefaultSigningAlgorithm = "HS256"
)

// Token defines the standard JWT structure
type Token struct {
	Header    map[string]interface{} // The first segment of the token
	Claims    map[string]interface{} // The second segment of the token
	Signature string                 // The third segment of the token.  Populated when you decode a token
}

// KeyLookupFunc is the function that uses the contents of Claims to lookup
// the encryption key used to sign the token.
type KeyLookupFunc func(claims map[string]interface{}) (interface{}, error)

// Signer is the interface for signing and verifying message signatures.
type Signer interface {
	Sign(msg string, key interface{}) (string, error)
	Verify(msg, signature string, key interface{}) error
	Algorithm() string
}

// JWT is the interface for managing Zenoss-specific JWT facilities for a given
// JWT authentication scheme. Application-level code in serviced should NOT use
// this interface directly. Instead, application-level code in other parts of
// serviced should use an interface specific to a particular authentication
// scheme such as DelegateAuthorizer.
type JWT interface {

	// Register the signer used for signing and verifying the JWT token.
	RegisterSigner(signer Signer)

	// Register the key lookup function. The key lookup function uses the data
	// from the Claims field to retrieve the key used to sign and verify the token.
	RegisterKeyLookup(keyLookup KeyLookupFunc)

	// Decodes encodedToken into a Token. The value of encodedToken should have
	// the standard JWT format:
	// <base64-encoded-header>.<base64-encoded-claims>.<base64-encoded-signature>
	DecodeToken(encodedToken string) (*Token, error)

	// Returns an encoded and signed JWT in the format
	// "<base64-encoded-header>.<base64-encoded-claims>.<base64-encoded-signature>"
	// Upon success, the value of token.Signature will be populated with the return value.
	EncodeAndSignToken(token *Token) (string, error)

	// Allocates a new token with a complete Header and partial Claims. The Claims
	// is initialized with JWT-standard 'iat' attribute, and the Zenoss-specific
	// attribte 'req'
	NewToken(method, urlString, uriPrefix string, body []byte) (*Token, error)

	// Validate the Token for compliance with the JWT standards and Zenoss-specific requirements
	ValidateToken(token *Token, method, urlString string, body []byte, jwtTTL time.Duration) error
}
