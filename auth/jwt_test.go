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
	"time"

	"github.com/control-center/serviced/auth"
	jwt "github.com/dgrijalva/jwt-go"
	. "gopkg.in/check.v1"
)

func at(t time.Time, f func()) {
	defer func() {
		jwt.TimeFunc = time.Now
	}()

	jwt.TimeFunc = func() time.Time {
		return t
	}

	f()
}

func (s *TestAuthSuite) TestIdentityHappyPath(c *C) {
	token, err := auth.CreateJWTIdentity("host", "pool", true, false, s.delegatePubPEM, time.Minute)

	c.Assert(err, IsNil)

	identity, err := auth.ParseJWTIdentity(token)
	c.Assert(err, IsNil)

	c.Assert(identity.HostID(), Equals, "host")
	c.Assert(identity.PoolID(), Equals, "pool")
	c.Assert(identity.Expired(), Equals, false)
	c.Assert(identity.HasAdminAccess(), Equals, true)
	c.Assert(identity.HasDFSAccess(), Equals, false)

	signer, _ := auth.RSASignerFromPEM(s.delegatePrivPEM)
	message := []byte("this is a message")
	sig, _ := signer.Sign(message)

	verifier, err := identity.Verifier()
	c.Assert(err, IsNil)

	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)
}

func (s *TestAuthSuite) TestExpiredToken(c *C) {
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, s.delegatePubPEM, time.Minute)

	fakenow := time.Now().UTC().Add(time.Hour)
	at(fakenow, func() {
		_, err := auth.ParseJWTIdentity(token)
		c.Assert(err, Equals, auth.ErrIdentityTokenExpired)
	})
}

func (s *TestAuthSuite) TestEarlyToken(c *C) {
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, s.delegatePubPEM, time.Minute)

	fakenow := time.Unix(0, 0)
	at(fakenow, func() {
		_, err := auth.ParseJWTIdentity(token)
		c.Assert(err, Equals, auth.ErrIdentityTokenNotValidYet)
	})
}

func (s *TestAuthSuite) TestBadSignature(c *C) {
	auth.LoadMasterKeysFromPEM(s.masterPubPEM, s.delegatePrivPEM)
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, s.delegatePubPEM, time.Minute)

	_, err := auth.ParseJWTIdentity(token)
	c.Assert(err, Equals, auth.ErrIdentityTokenBadSig)
}
