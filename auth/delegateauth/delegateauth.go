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

// Package delegateauth implements JWT authentication for serviced delegates.
package delegateauth

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"time"

	"github.com/control-center/serviced/auth/jwt"
)

const (
	// DelegateIssuerID - The issuer ('iss') value for delegate authentication
	DelegateIssuerID = "serviced_delegate"

	// DelegateAuthV1 - The Zenoss Authentication Version ('zav') for version 1
	// of the serviced delegate authentication scheme
	DelegateAuthV1 = "cc_delegate_auth_v1"
)

// A package-private facade that implements the JWT interface while hiding
// implementation details specific to the dgrijalva/jwt-go library
type delegateAuthorizer struct {
	jwt    jwt.JWT
	jwtTTL int
}

// assert the interface
var _ DelegateAuthorizer = &delegateAuthorizer{}

// NewDelegateAuthorizer creates a new authorizer instance
func NewDelegateAuthorizer(jwtTTL int) (DelegateAuthorizer, error) {
	Jwt, err := jwt.NewInstance(jwt.DefaultSigningAlgorithm, getKeyLookup())
	if err != nil {
		return nil, err
	}
	authorizer := &delegateAuthorizer{
		jwt:    Jwt,
		jwtTTL: jwtTTL,
	}
	return authorizer, nil
}

// Build a JWT and add it to the request header
func (authorizer *delegateAuthorizer) AddToken(poolID string, request *http.Request, uriPrefix string) error {
	body, err := authorizer.readBody(request)
	if err != nil {
		return err
	}

	token, err := authorizer.getToken(poolID, request.Method, request.URL.String(), uriPrefix, body)
	if err != nil {
		return err
	}

	encodedToken, err := authorizer.jwt.EncodeAndSignToken(token)
	if err != nil {
		return fmt.Errorf("failed to encode token: %s", err)
	}

	request.Header.Add("Authorization", fmt.Sprintf("JWT %s", encodedToken))
	return nil
}

// Validate the JWT token in the request header
func (authorizer *delegateAuthorizer) ValidateToken(request *http.Request) error {
	authorizationHeader := request.Header.Get("Authorization")
	if len(authorizationHeader) == 0 {
		return fmt.Errorf("Request is missing Authorization header")
	}

	var encodedToken string
	_, err := fmt.Sscanf(authorizationHeader, "JWT %s", &encodedToken)
	if err != nil {
		return fmt.Errorf("Could not parse Authorization header %q: %s", authorizationHeader, err)
	}

	token, err := authorizer.jwt.DecodeToken(encodedToken)
	if err != nil {
		return fmt.Errorf("Could not parse JWT %q: %s", encodedToken, err)
	}

	body, err := authorizer.readBody(request)
	if err != nil {
		return err
	}

	tokenTTL := time.Duration(authorizer.jwtTTL) * time.Second
	return authorizer.jwt.ValidateToken(token, request.Method, request.URL.String(), body, tokenTTL)
}

func (authorizer *delegateAuthorizer) getToken(poolID, method, urlString, uriPrefix string, body []byte) (*jwt.Token, error) {
	token, err := authorizer.jwt.NewToken(method, urlString, uriPrefix, body)
	if err != nil {
		return nil, fmt.Errorf("failed to build token: %s", err)
	}

	token.Claims["iss"] = DelegateIssuerID
	token.Claims["sub"] = poolID
	token.Claims["zav"] = DelegateAuthV1
	return token, nil
}

func (authorizer *delegateAuthorizer) readBody(request *http.Request) ([]byte, error) {
	if request.Body == nil {
		return nil, nil
	}

	body, err := ioutil.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read request body: %s", err)
	}

	if err := request.Body.Close(); err != nil {
		return nil, fmt.Errorf("failed to close request body: %s", err)
	}

	// Replace request.Body with a new reader that has a copy of the original body
	buffer := bytes.NewBuffer(body)
	request.Body = ioutil.NopCloser(bytes.NewReader(buffer.Bytes()))
	return body, nil
}

func getKeyLookup() jwt.KeyLookupFunc {
	return func(claims map[string]interface{}) (interface{}, error) {
		if claims["iss"] != DelegateIssuerID {
			return nil, fmt.Errorf("Claims['iss']=%q is invalid", claims["iss"])
		} else if claims["zav"] != DelegateAuthV1 {
			return nil, fmt.Errorf("Claims['zav']=%q is invalid", claims["zav"])
		}

		// FIXME: lookup poolID and return secret
		return "someSecret", nil
	}
}
