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

var (
	pubkey, _  = auth.RSAPublicKeyFromPEM(auth.DevPubKeyPEM)
	privkey, _ = auth.RSAPrivateKeyFromPEM(auth.DevPrivKeyPEM)

	pubkey2, _ = auth.RSAPublicKeyFromPEM([]byte(`
-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvVqjjHX9S6Ir11Ql0yOL
6//CUDy5soSGofU+eDM/joELQq0d0upqLwiaOgS3rfnQakXhJsunA0ERcSO8R3AV
shmVqPRrvEnsfMZZR7N5xtNaCGrJfbhIB/4uPzxpTQ/fsgFQK6djOA8IXM0D5YWI
EYRJwagNkVK2/kP8xOkN+7K4JQ9y+eq3OxC0o0W9WTk686HW62/i8MMBfe74P9E+
Sm7P4fZKY1zLTzF3cMDkUrh1VxPHh2ZOmHT9WC3wtjYy0HIECirA2Mm65Jb4Ug4U
YMG/KaPg45TTAMcA7ZA+dbxNDvjemSnAvCUqVuqIEFJ0Mnh+fmnKe95mzMym0Y5U
aQIDAQAB
-----END PUBLIC KEY-----`))

	privkey2, _ = auth.RSAPrivateKeyFromPEM([]byte(`
-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAvVqjjHX9S6Ir11Ql0yOL6//CUDy5soSGofU+eDM/joELQq0d
0upqLwiaOgS3rfnQakXhJsunA0ERcSO8R3AVshmVqPRrvEnsfMZZR7N5xtNaCGrJ
fbhIB/4uPzxpTQ/fsgFQK6djOA8IXM0D5YWIEYRJwagNkVK2/kP8xOkN+7K4JQ9y
+eq3OxC0o0W9WTk686HW62/i8MMBfe74P9E+Sm7P4fZKY1zLTzF3cMDkUrh1VxPH
h2ZOmHT9WC3wtjYy0HIECirA2Mm65Jb4Ug4UYMG/KaPg45TTAMcA7ZA+dbxNDvje
mSnAvCUqVuqIEFJ0Mnh+fmnKe95mzMym0Y5UaQIDAQABAoIBAA/Ujw3EI3v6P94N
q+fd9emyBSW+Hew9xh+uKB3Wcv7P7QgS/wZOELiD6JjnIhAKbZEk7HDY38qW2wVx
bmEujrHID8oDPSqNp5a64mXrLEgiDUnc26GGEYeRiX5B56/InvP7xh8QLGxYXWOc
xDGhG0ITpDLrgM7gcmoJdw1jSob7Qi4FU93bSS4D6JiDOGDhtDveq6/V3Y/eJu+P
RuXk/SjzWSey0oxeYxbM9DaBXSUzDO/PLWZv1uPDg4Ws5qAnw+mGfKi59MRF8naA
978/hbOkBKBUphWIiUxO0CJyWAhd9y6VAYrKIl0GBQdqvgdirGAeBgFCGme3trU2
YlcALR0CgYEA5lAO5Pc13Tr25g8/zeaY9beksDe5Q/4aiJFMBBRjlBpxcWa90yiS
5PfjWiNHfTcGq4Dcr5ZTQihk/ZP6E9pv7H/FaGCbFNIX4rK3IYD0yqAgQFA+ppFK
T6llPzTzJMrKXIOLMVNYFod6nbE+YDmz17n3fUW6gFSp/ZLZGTRnYjsCgYEA0nke
joQT4t0lsWchUhO53qYOdbA64KdyMDxKb/rXQhbsB5KhWWkwQBa7EuiSCuVMfBhH
BD4EdJqP9l23qBrJZuowsBIZt8ozYBfsKZnzuXYqhismOYTFQ7XyHN4EYmIyNLxi
4LmpnYlD5Qc+GaDatZI66cyzasZXM098nXtktasCgYEAtRwfmk4MPXwwy5kSQ4gi
oJdZGnm3ZpBbrSkU7eBargxdSR/SBkrRuNx2HFvBy+WJiTQ8VpePwWaihAXpkdMk
UIXpZrsROL49qjd/awlNdkmVEv4HRlTaaup6g8nPqg8OMtH+kztG+fBvq7HFq0W0
9t92jzxV/LSXOKBRuFBNPCECgYEAt/jA2effDg0p9mBkAr9VV6WkvABX5qjWqgz5
L9p9r7ojhBcKTAIi99ImoUeC6F03trztzmp7MIUt0zZl413Or9OCzVR1AG6Q66zd
dBuqq3D7iJ1M4zgHycDPKaZzBKA6rFgCwdXnydkC7L2g7Xvp0I5KSrTwGyPVcvdG
wMzr4dMCgYEA0WrVMbJVnvJBpTjAa6jAxt3QpE5J7+8xft6H8grlTC7xyZeLr8qT
IoLCT5Kcc42xQkZSwT/8ajIjb/nxTaYh/PcRxV61cqVyaumkZ2kFOEEA3lXtafye
4bqWx3C5SKP5enhXAK8t/4dzY3pTs/GNZ1hJIDUX87NjidIUvxIfMaM=
-----END RSA PRIVATE KEY-----`))
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
	token, err := auth.CreateJWTIdentity("host", "pool", true, false, pubkey, time.Minute, privkey)

	c.Assert(err, IsNil)

	identity, err := auth.ParseJWTIdentity(token, pubkey)
	c.Assert(err, IsNil)

	c.Assert(identity.HostID(), Equals, "host")
	c.Assert(identity.PoolID(), Equals, "pool")
	c.Assert(identity.Expired(), Equals, false)
	c.Assert(identity.HasAdminAccess(), Equals, true)
	c.Assert(identity.HasDFSAccess(), Equals, false)

	signer, _ := auth.RSASigner(privkey)
	message := []byte("this is a message")
	sig, _ := signer.Sign(message)

	verifier, err := identity.Verifier()
	c.Assert(err, IsNil)

	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)
}

func (s *TestAuthSuite) TestExpiredToken(c *C) {
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, pubkey, time.Minute, privkey)

	fakenow := time.Now().UTC().Add(time.Hour)
	at(fakenow, func() {
		_, err := auth.ParseJWTIdentity(token, pubkey)
		c.Assert(err, Equals, auth.ErrIdentityTokenExpired)
	})
}

func (s *TestAuthSuite) TestEarlyToken(c *C) {
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, pubkey, time.Minute, privkey)

	fakenow := time.Unix(0, 0)
	at(fakenow, func() {
		_, err := auth.ParseJWTIdentity(token, pubkey)
		c.Assert(err, Equals, auth.ErrIdentityTokenNotValidYet)
	})
}

func (s *TestAuthSuite) TestBadSignature(c *C) {
	token, _ := auth.CreateJWTIdentity("host", "pool", true, false, pubkey, time.Minute, privkey2)

	_, err := auth.ParseJWTIdentity(token, pubkey)
	c.Assert(err, Equals, auth.ErrIdentityTokenBadSig)
}
