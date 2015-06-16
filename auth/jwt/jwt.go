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
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	// "net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
)

// A package-private facade that implements the JWT interface while hiding
// implementation details specific to the dgrijalva/jwt-go library
type jwtFacade struct {
	keyLookup KeyLookupFunc
	signer    Signer
}

// assert the interface
var _ JWT = &jwtFacade{}

func (facade *jwtFacade) RegisterSigner(signer Signer) {
	facade.signer = signer
}

func (facade *jwtFacade) RegisterKeyLookup(keyLookup KeyLookupFunc) {
	facade.keyLookup = keyLookup
}

// Allocates a new token with a complete Header and partial Claims.
func (facade *jwtFacade) NewToken(method, urlString, uriPrefix string, body []byte) (*Token, error) {
	token := &Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": DEFAULT_ALGORITHM,
		},
		Claims: map[string]interface{}{
			"iat": float64(time.Now().Unix()),		// use float64 to match default behavior of json.Unmarshall()
		},
	}

	canonicalURL, err := CanonicalURL(method, urlString, uriPrefix, body)
	if err != nil {
		return nil, fmt.Errorf("Failed to build canonicalURL: %v", err)
	}

	requestHash := sha256.Sum256(canonicalURL)
	token.Claims["req"] = hex.EncodeToString(requestHash[:])
	return token, nil
}

// Returns a signed token in the standard JWT format :
// <base64-encoded-header>.<base64-encoded-claims>.<base64-encoded-signature>
// The value of token.Signature will also be populated with the return value.
func (facade *jwtFacade) EncodeAndSignToken(token *Token) (string, error) {
	if err := validateMinimumRequirements(token); err != nil {
		return "", fmt.Errorf("invalid token: %v", err)
	}

	key, err := facade.keyLookup(token.Claims)
	if err != nil {
		return "", fmt.Errorf("Key lookup failed: %v", err)
	}

	signerFacade, _ := facade.signer.(*signerFacade)
	jwtgoToken := jwtgo.New(signerFacade.signingMethod)
	jwtgoToken.Header = token.Header
	jwtgoToken.Claims = token.Claims

	if encodedToken, err := jwtgoToken.SignedString(key); err == nil {
		token.Signature = strings.Split(encodedToken, ".")[2]
		return encodedToken, nil
	} else {
		return "", fmt.Errorf("Error encoding token: %v", err)
	}
}

// DecodeToken encodedToken into a Token. The value of encodedToken should have
// the standard JWT format:
// <base64-encoded-header>.<base64-encoded-claims>.<base64-encoded-signature>
//
// NOTE: the jwtgo.Parse() method will validate the values for the Claims
//       atributes "exp" and "nbf" (expires and not-before respectively) If
//       the Claims contains those values.  At the time this package was
//       created, Zenoss was not planning on using those tokens, so it shouldn't
//       be a problem.
func (facade *jwtFacade) DecodeToken(encodedToken string) (*Token, error) {
	jwtgoToken, err := jwtgo.Parse(encodedToken, func(token *jwtgo.Token) (interface{}, error) {
		return facade.keyLookup(token.Claims)
	})
	if err != nil || !jwtgoToken.Valid  {
		return nil, fmt.Errorf("Failed to decode token: %v", err)
	}

	token := &Token{
		Header:    jwtgoToken.Header,
		Claims:    jwtgoToken.Claims,
		Signature: strings.Split(encodedToken, ".")[2],
	}
	return token, nil
}

// Validate the Token for compliance with the JWT standards and Zenoss-specific requirements
func (facade *jwtFacade) ValidateToken(token *Token, method, urlString string, body []byte, expirationLimit time.Duration) error {
	if err := validateMinimumRequirements(token); err != nil {
		return fmt.Errorf("invalid token: %v", err)
	} else if token.Signature == "" {
		return fmt.Errorf("token is missing Signature")
	}

	issuedAtTime, err := getIssuedAtTime(token)
	if err != nil {
		return fmt.Errorf("Claims['iat'] is not valid: %s", err)
	} else if expirationLimit.Seconds() > 1.0 {
		var expirationTime float64
		expirationTime = issuedAtTime + float64(expirationLimit.Seconds())
		if expirationTime < float64(time.Now().Unix()) {
			return fmt.Errorf("token has expired")
		}
	}

	canonicalURL, err := CanonicalURL(method, urlString, "", body)
	if err != nil {
		return fmt.Errorf("Can not normalize request %s %s: %s", method, urlString, err)
	}
	requestHash := sha256.Sum256(canonicalURL)
	encodedHash := hex.EncodeToString(requestHash[:])
	if encodedHash != token.Claims["req"] {
		return fmt.Errorf("The request signature does not match request %s %s", method, urlString)
	}

	return nil
}

// Validate the minimum set of attributes required for the Header and Claims
// per a combination of JWT and Zenoss-specific conventions.
func validateMinimumRequirements(token *Token) error {
	if token == nil {
		return fmt.Errorf("nil token is invalid")
	}

	headerRequirements := map[string]interface{}{
		"typ": "JWT",
		"alg": DEFAULT_ALGORITHM,
	}
	if err := validateMap(token.Header, headerRequirements); err != nil {
		return fmt.Errorf("token.Header invalid: %v", err)
	}

	claimsRequirements := map[string]interface{}{
		"iat": nil, // JWT-standard, issued-at-time
		"iss": nil, // JWT-standard, issuer id
		"sub": nil, // JWT-standard, subject
		"zav": nil, // Zenoss-standard, Zenoss Authentication Version
		"req": nil, // Zenoss-standard, request signature, a base64-encoded value of a
					// SHA256 of the canonical request parameters
	}
	if err := validateMap(token.Claims, claimsRequirements); err != nil {
		return fmt.Errorf("token.Claims invalid: %s", err)
	}

	return nil
}

// Validate that the source map contains the minimum required attributes.
func validateMap(source, minimumRequirements map[string]interface{}) error {
	if source == nil {
		return fmt.Errorf("can not be nil")
	} else if len(source) == 0 {
		return fmt.Errorf("can not be empty")
	}

	for key, requiredValue := range minimumRequirements {
		if value, ok := source[key]; !ok {
			return fmt.Errorf("missing %q", key)
		} else if requiredValue != nil && !strings.EqualFold(value.(string), requiredValue.(string)) {
			return fmt.Errorf("[%q] should be %q, not %q", key, requiredValue, value)
		}
	}
	return nil
}

func getIssuedAtTime(token *Token) (float64, error) {
	rawIatValue := token.Claims["iat"]
	var err error
	var issuedAtTime float64

	switch rawIatValue := rawIatValue.(type) {
	case float32:
		issuedAtTime = float64(rawIatValue)
	case float64:
		issuedAtTime = rawIatValue
	case int:
		issuedAtTime = float64(rawIatValue)
	case int32:
		issuedAtTime = float64(rawIatValue)
	case int64:
		issuedAtTime = float64(rawIatValue)
	case uint:
		issuedAtTime = float64(rawIatValue)
	case uint32:
		issuedAtTime = float64(rawIatValue)
	case uint64:
		issuedAtTime = float64(rawIatValue)
	case string:
		issuedAtTime, err = strconv.ParseFloat(rawIatValue, 64)
	default:
		err = fmt.Errorf("Type %q is not valid", reflect.TypeOf(rawIatValue))
	}

	return issuedAtTime, err
}
