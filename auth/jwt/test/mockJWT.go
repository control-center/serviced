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

package test

import (
	"time"

	"github.com/control-center/serviced/auth/jwt"

	"github.com/stretchr/testify/mock"
)

// assert the interface
var _ jwt.JWT = &MockJWT{}

type MockJWT struct {
	mock.Mock
}

func (mjwt *MockJWT) RegisterSigner(signer jwt.Signer) {
	mjwt.Mock.Called(signer)
}

func (mjwt *MockJWT) RegisterKeyLookup(keyLookup jwt.KeyLookupFunc) {
	mjwt.Mock.Called(keyLookup)
}

func (mjwt *MockJWT) NewToken(method, urlString, uriPrefix string, body []byte) (*jwt.Token, error) {
	args := mjwt.Mock.Called(method, urlString, uriPrefix, body)

	var token *jwt.Token
	if arg0 := args.Get(0); arg0 != nil {
		token = arg0.(*jwt.Token)
	}
	return token, args.Error(1)
}

func (mjwt *MockJWT) EncodeAndSignToken(token *jwt.Token) (string, error) {
	args := mjwt.Mock.Called(token)

	var encodedToken string
	if arg0 := args.Get(0); arg0 != nil {
		encodedToken = arg0.(string)
	}

	return encodedToken, args.Error(1)
}

func (mjwt *MockJWT) DecodeToken(encodedToken string) (*jwt.Token, error) {
	args := mjwt.Mock.Called(encodedToken)

	var token *jwt.Token
	if arg0 := args.Get(0); arg0 != nil {
		token = arg0.(*jwt.Token)
	}
	return token, args.Error(1)
}

func (mjwt *MockJWT) ValidateToken(token *jwt.Token, method, urlString string, body []byte, jwtTTL time.Duration) error {
	args := mjwt.Mock.Called(token, method, urlString, body, jwtTTL)
	return args.Error(0)
}
