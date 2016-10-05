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

// +build unit

package auth_test

import (
	"fmt"
	"net/http"
	"time"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

type restTestConfig struct {
	hostID string
	poolID string
	admin  bool
	dfs    bool
	exp    time.Duration
	method string
	uri    string
}

var (
	originalTokenGetter         = auth.AuthTokenGetter
	originalRestTokenExpiration = auth.RestTokenExpiration
)

func restTestCleanup() {
	auth.AuthTokenGetter = originalTokenGetter
	auth.RestTokenExpiration = originalRestTokenExpiration
}

func newTestConfig() restTestConfig {
	return restTestConfig{"mockHost", "mockPool", true, true, time.Minute, "GET", "/super/fake/request"}
}

func (s *TestAuthSuite) TestBuildAndExtractRestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	expectedReqHash := auth.GetRequestHash(req)
	// Create Rest Token
	restToken, err := auth.BuildRestToken(req)
	c.Assert(err, IsNil)
	c.Assert(restToken, NotNil)

	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	h := req.Header.Get("Authorization")
	c.Assert(h, DeepEquals, fmt.Sprintf("Bearer %s", restToken))

	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	c.Assert(extractedToken, DeepEquals, restToken)

	// Parse token
	parsedToken, err := auth.ParseRestToken(extractedToken)
	c.Assert(err, IsNil)
	c.Assert(parsedToken, NotNil)
	c.Assert(parsedToken.RestToken(), DeepEquals, restToken)
	c.Assert(parsedToken.AuthToken(), DeepEquals, authToken)
	c.Assert(parsedToken.HasAdminAccess(), Equals, cfg.admin)
	c.Assert(parsedToken.Expired(), Equals, false)
	c.Assert(parsedToken.Valid(), IsNil)
	c.Assert(parsedToken.RequestHash(), DeepEquals, expectedReqHash)

	restTestCleanup()
}

func (s *TestAuthSuite) TestExpiredRestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	auth.RestTokenExpiration = -1 * time.Hour
	// Create Rest Token
	restToken, err := auth.BuildRestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrRestTokenExpired)

	restTestCleanup()
}

func (s *TestAuthSuite) TestTamperedRestToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := auth.BuildRestToken(req)
	c.Assert(err, IsNil)
	// modify token
	l := len(restToken)
	restToken = restToken[:l-4] + "HOLA"
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	c.Assert(extractedToken, DeepEquals, restToken)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrRestTokenBadSig)

	restTestCleanup()
}

func (s *TestAuthSuite) TestInvalidRestToken(c *C) {
	cfg := newTestConfig()
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	// Empty token
	auth.AddRestTokenToRequest(req, "")
	_, err := auth.ExtractRestToken(req)
	c.Assert(err, Equals, auth.ErrBadRestToken)

	// Invalid token
	req, _ = http.NewRequest(cfg.method, cfg.uri, nil)
	invalidToken := "THIS ISNT A REST TOKEN"
	auth.AddRestTokenToRequest(req, invalidToken)
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	token, err := auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrBadRestToken)
	c.Assert(token, IsNil)

	restTestCleanup()
}

func (s *TestAuthSuite) TestExpiredAuthToken(c *C) {
	cfg := newTestConfig()
	cfg.exp = -1 * time.Hour
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := auth.BuildRestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrIdentityTokenExpired)

	restTestCleanup()
}

func (s *TestAuthSuite) TestTamperedAuthToken(c *C) {
	cfg := newTestConfig()
	// Create auth token
	authToken, _, err := auth.CreateJWTIdentity(cfg.hostID, cfg.poolID, cfg.admin, cfg.dfs, s.delegatePubPEM, cfg.exp)
	authToken = authToken[:40] + "HOLA" + authToken[44:]
	// Create requests
	req, _ := http.NewRequest(cfg.method, cfg.uri, nil)
	auth.AuthTokenGetter = func() (string, error) {
		return authToken, nil
	}
	// Create Rest Token
	restToken, err := auth.BuildRestToken(req)
	c.Assert(err, IsNil)
	// Add rest token to request header
	auth.AddRestTokenToRequest(req, restToken)
	// Extract rest token from request
	extractedToken, err := auth.ExtractRestToken(req)
	c.Assert(err, IsNil)
	// Parse token
	_, err = auth.ParseRestToken(extractedToken)
	c.Assert(err, Equals, auth.ErrIdentityTokenBadSig)

	restTestCleanup()
}
