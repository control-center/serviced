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
package jwt

import (
	"time"
)

// The default for the JWT-standard name of the signing algorithm.
const (
	DEFAULT_ALGORITHM = "HS256"
)

type Token struct {
	Header    map[string]interface{} // The first segment of the token
	Claims    map[string]interface{} // The second segment of the token
	Signature string                 // The third segment of the token.  Populated when you decode a token
}

type KeyLookupFunc func(claims map[string]interface{}) (interface{}, error)

type Signer interface {
	Sign(msg string, key interface{}) (string, error)
	Verify(msg, signature string, key interface{}) error
	Algorithm() string
}

type JWT interface {

	// Register the signer used for signing and verifying the JWT token
	RegisterSigner(signer Signer)

	// Register the key lookup function. The key lookup function uses the data
	// from the Claims field of the token to lookup the key passed to the
	// signing function.
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
	ValidateToken(token *Token, method, urlString string, body []byte, expirationLimit time.Duration) error
}
