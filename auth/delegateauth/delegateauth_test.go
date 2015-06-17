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
package delegateauth

import (
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"strings"
	"testing"

	"github.com/control-center/serviced/auth/jwt"
	jwttest "github.com/control-center/serviced/auth/jwt/test"

	. "gopkg.in/check.v1"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

type delegateAuthTest struct {
	//  A mock implementation of JWT
	mockJWT    *jwttest.MockJWT
	authorizer *delegateAuthorizer
}

var _ = Suite(&delegateAuthTest{})

func (t *delegateAuthTest) SetUpTest(c *C) {
	t.mockJWT = &jwttest.MockJWT{}
	t.authorizer = &delegateAuthorizer{
		jwt: t.mockJWT,
	}
}

func (t *delegateAuthTest) TearDownTest(c *C) {
	// don't allow per-test-case values to be reused across test cases
	t.mockJWT = nil
	t.authorizer = nil
}

func (t *delegateAuthTest) TestNewDelegateAuthorizer(c *C) {
	da, err := NewDelegateAuthorizer(10)

	c.Assert(err, IsNil)
	c.Assert(da, NotNil)
	c.Assert(da.(*delegateAuthorizer).jwtTTL, Equals, 10)
}

func (t *delegateAuthTest) TestAddTokenWithBody(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	uriPrefix := "somePrefix"
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	emptyToken := t.getToken()
	encodedToken := "mockEncodedTokenString"

	t.mockJWT.
		On("NewToken", method, url, uriPrefix, []byte(expectedBody)).
		Return(emptyToken, nil)
	t.mockJWT.
		On("EncodeAndSignToken", emptyToken).
		Return(encodedToken, nil)
	request, _ := http.NewRequest(method, url, strings.NewReader(expectedBody))

	err := t.authorizer.AddToken("somePoolId", request, uriPrefix)

	c.Assert(err, IsNil)

	authHeader := request.Header.Get("Authorization")
	c.Assert(authHeader, NotNil)
	c.Assert(authHeader, Equals, fmt.Sprintf("JWT %s", encodedToken))
}

func (t *delegateAuthTest) TestAddTokenWithoutBody(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	uriPrefix := "somePrefix"
	emptyToken := t.getToken()
	var nilBody []byte
	encodedToken := "mockEncodedTokenString"

	t.mockJWT.
		On("NewToken", method, url, uriPrefix, nilBody).
		Return(emptyToken, nil)
	t.mockJWT.
		On("EncodeAndSignToken", emptyToken).
		Return(encodedToken, nil)
	request, _ := http.NewRequest(method, url, nil)

	err := t.authorizer.AddToken("somePoolId", request, uriPrefix)

	c.Assert(err, IsNil)

	authHeader := request.Header.Get("Authorization")
	c.Assert(authHeader, NotNil)
	c.Assert(authHeader, Equals, fmt.Sprintf("JWT %s", encodedToken))
}

func (t *delegateAuthTest) TestAddTokenFailsToReadBody(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	uriPrefix := "somePrefix"
	request, _ := http.NewRequest(method, url, &failedReadCloser{failRead: true})

	err := t.authorizer.AddToken("somePoolId", request, uriPrefix)

	c.Assert(err.Error(), Equals, "failed to read request body: read failed")
}

func (t *delegateAuthTest) TestAddTokenFailsToBuildToken(c *C) {
	tokenFailure := fmt.Errorf("NewToken failed")
	method := "GET"
	url := "http://control-center/dummy"
	uriPrefix := "somePrefix"
	var nilBody []byte
	t.mockJWT.
		On("NewToken", method, url, uriPrefix, nilBody).
		Return(nil, tokenFailure)
	request, _ := http.NewRequest(method, url, nil)

	err := t.authorizer.AddToken("somePoolId", request, uriPrefix)

	c.Assert(err.Error(), Equals, "failed to build token: NewToken failed")
}

func (t *delegateAuthTest) TestAddTokenFailsToEncodeToken(c *C) {
	tokenFailure := fmt.Errorf("Encoding failed")
	method := "GET"
	url := "http://control-center/dummy"
	uriPrefix := "somePrefix"
	var nilBody []byte
	emptyToken := t.getToken()
	t.mockJWT.
		On("NewToken", method, url, uriPrefix, nilBody).
		Return(emptyToken, nil)
	t.mockJWT.
		On("EncodeAndSignToken", emptyToken).
		Return(nil, tokenFailure)
	request, _ := http.NewRequest(method, url, nil)

	err := t.authorizer.AddToken("somePoolId", request, uriPrefix)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "failed to encode token: Encoding failed")
}

func (t *delegateAuthTest) TestValidateToken(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	request, _ := http.NewRequest(method, url, nil)
	request.Header.Add("Authorization", "JWT bogusValue")
	emptyToken := t.getToken()
	t.mockJWT.
		On("DecodeToken", "bogusValue").
		Return(emptyToken, nil)
	var nilBody []byte
	t.mockJWT.
		On("ValidateToken", emptyToken, method, url, nilBody, 0).
		Return(nil)

	err := t.authorizer.ValidateToken(request)

	c.Assert(err, IsNil)
}

func (t *delegateAuthTest) TestValidateTokenFailsForMissingHeader(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	request, _ := http.NewRequest(method, url, strings.NewReader(expectedBody))

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, "Request is missing Authorization header")
}

func (t *delegateAuthTest) TestValidateTokenFailsForInvalidHeader(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	request, _ := http.NewRequest(method, url, strings.NewReader(expectedBody))
	request.Header.Add("Authorization", "invalid syntax")

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, "Could not parse Authorization header \"invalid syntax\": input does not match format")
}

func (t *delegateAuthTest) TestValidateTokenFailsForInvalidHeader2(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	request, _ := http.NewRequest(method, url, strings.NewReader(expectedBody))
	request.Header.Add("Authorization", "JWT")

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, "Could not parse Authorization header \"JWT\": EOF")
}

func (t *delegateAuthTest) TestValidateTokenFailsDecoding(c *C) {
	tokenFailure := fmt.Errorf("Decoding failed")
	method := "GET"
	url := "http://control-center/dummy"
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	request, _ := http.NewRequest(method, url, strings.NewReader(expectedBody))
	request.Header.Add("Authorization", "JWT bogusValue")
	t.mockJWT.
		On("DecodeToken", "bogusValue").
		Return(nil, tokenFailure)

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, "Could not parse JWT \"bogusValue\": Decoding failed")
}

func (t *delegateAuthTest) TestValidateTokenFailsToReadBody(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	request, _ := http.NewRequest(method, url, &failedReadCloser{failRead: true})
	request.Header.Add("Authorization", "JWT bogusValue")
	emptyToken := t.getToken()
	t.mockJWT.
		On("DecodeToken", "bogusValue").
		Return(emptyToken, nil)

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, "failed to read request body: read failed")
}

func (t *delegateAuthTest) TestValidateTokenFails(c *C) {
	method := "GET"
	url := "http://control-center/dummy"
	request, _ := http.NewRequest(method, url, nil)
	request.Header.Add("Authorization", "JWT bogusValue")
	emptyToken := t.getToken()
	t.mockJWT.
		On("DecodeToken", "bogusValue").
		Return(emptyToken, nil)
	tokenFailure := fmt.Errorf("Validation failed")
	var nilBody []byte
	t.mockJWT.
		On("ValidateToken", emptyToken, method, url, nilBody, 0).
		Return(tokenFailure)

	err := t.authorizer.ValidateToken(request)

	c.Assert(err.Error(), Equals, tokenFailure.Error())
}

func (t *delegateAuthTest) TestKeyLookup(c *C) {
	keyLookup := getKeyLookup()
	claims := make(map[string]interface{})
	claims["iss"] = DelegateIssuerID
	claims["zav"] = DelegateAuthV1
	claims["sub"] = "somePoolid"

	key, err := keyLookup(claims)

	c.Assert(key, Equals, "someSecret")
	c.Assert(err, IsNil)
}

func (t *delegateAuthTest) TestKeyLookupFailsForInvalidISS(c *C) {
	keyLookup := getKeyLookup()
	claims := make(map[string]interface{})
	claims["iss"] = "not a valid issuer"
	claims["zav"] = DelegateAuthV1
	claims["sub"] = "somePoolid"

	key, err := keyLookup(claims)

	c.Assert(err.Error(), Equals, "Claims['iss']=\"not a valid issuer\" is invalid")
	c.Assert(key, IsNil)
}

func (t *delegateAuthTest) TestKeyLookupFailsForInvalidZAV(c *C) {
	keyLookup := getKeyLookup()
	claims := make(map[string]interface{})
	claims["iss"] = DelegateIssuerID
	claims["zav"] = "not a valid auth version"
	claims["sub"] = "somePoolid"

	key, err := keyLookup(claims)

	c.Assert(err.Error(), Equals, "Claims['zav']=\"not a valid auth version\" is invalid")
	c.Assert(key, IsNil)
}

func (t *delegateAuthTest) TestReadBody(c *C) {
	expectedBody := "{\"param1\": \"value1\", \"param2\": 5}"
	request, _ := http.NewRequest("GET", "http://control-center/dummy", strings.NewReader(expectedBody))

	body, err := t.authorizer.readBody(request)

	c.Assert(err, IsNil)
	c.Assert(body, NotNil)
	c.Assert(string(body), Equals, expectedBody)

	// Make sure that request.Body provides the same data as before
	c.Assert(request.Body, NotNil)
	body2, err2 := ioutil.ReadAll(request.Body)
	c.Assert(err2, IsNil)
	c.Assert(body2, NotNil)
	c.Assert(string(body2), Equals, expectedBody)
}

func (t *delegateAuthTest) TestReadBodyWithNil(c *C) {
	request, _ := http.NewRequest("GET", "http://control-center/dummy", nil)

	body, err := t.authorizer.readBody(request)

	c.Assert(body, IsNil)
	c.Assert(request.Body, IsNil)
	c.Assert(err, IsNil)
}

func (t *delegateAuthTest) TestReadBodyFailsOnRead(c *C) {
	request, _ := http.NewRequest("GET", "http://control-center/dummy", &failedReadCloser{failRead: true})

	body, err := t.authorizer.readBody(request)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "failed to read request body: read failed")
	c.Assert(body, IsNil)
}

func (t *delegateAuthTest) TestReadBodyFailsOnClose(c *C) {
	request, _ := http.NewRequest("GET", "http://control-center/dummy", nil)
	request.Body = &failedReadCloser{failRead: false}

	body, err := t.authorizer.readBody(request)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "failed to close request body: close failed")
	c.Assert(body, IsNil)
}

func (t *delegateAuthTest) getToken() *jwt.Token {
	return &jwt.Token{
		Header: make(map[string]interface{}),
		Claims: make(map[string]interface{}),
	}
}

type failedReadCloser struct {
	failRead bool
}

func (fr *failedReadCloser) Read(p []byte) (n int, err error) {
	if fr.failRead {
		return 0, fmt.Errorf("read failed")
	}
	return 0, io.EOF
}

func (fr *failedReadCloser) Close() error {
	return fmt.Errorf("close failed")
}
