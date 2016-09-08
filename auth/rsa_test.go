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
	"bytes"
	"crypto/rsa"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

func (s *TestAuthSuite) TestDevKeys(c *C) {
	signer := auth.DevRSASigner()
	verifier := auth.DevRSAVerifier()

	message := []byte("this is a message that I plan to test")

	sig, err := signer.Sign(message)
	c.Assert(err, IsNil)

	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)
}

func (s *TestAuthSuite) TestRSAPrivateKeyFromPEM(c *C) {
	pem := auth.DevPrivKeyPEM

	var sample *rsa.PrivateKey

	key, err := auth.RSAPrivateKeyFromPEM(pem)
	c.Assert(err, IsNil)
	c.Assert(key, FitsTypeOf, sample)

	// Test a public key
	key, err = auth.RSAPrivateKeyFromPEM(auth.DevPubKeyPEM)
	c.Assert(err, Equals, auth.ErrNotRSAPrivateKey)
	c.Assert(key, IsNil)

	// Test bad PEM
	var badpem bytes.Buffer
	badpem.Write(pem[:100])
	badpem.Write([]byte("nopenotvalidpem"))
	badpem.Write(pem[100:len(pem)])
	key, err = auth.RSAPrivateKeyFromPEM(badpem.Bytes())
	c.Assert(err, Equals, auth.ErrNotPEMEncoded)
	c.Assert(key, IsNil)
}

func (s *TestAuthSuite) TestRSAPublicKeyFromPEM(c *C) {
	pem := auth.DevPubKeyPEM

	var sample *rsa.PublicKey

	key, err := auth.RSAPublicKeyFromPEM(pem)
	c.Assert(err, IsNil)
	c.Assert(key, FitsTypeOf, sample)

	// Test a public key
	key, err = auth.RSAPublicKeyFromPEM(auth.DevPrivKeyPEM)
	c.Assert(err, Equals, auth.ErrNotRSAPublicKey)
	c.Assert(key, IsNil)

	// Test bad PEM
	var badpem bytes.Buffer
	badpem.Write(pem[:100])
	badpem.Write([]byte("nopenotvalidpem"))
	badpem.Write(pem[100:len(pem)])
	key, err = auth.RSAPublicKeyFromPEM(badpem.Bytes())
	c.Assert(err, Equals, auth.ErrNotPEMEncoded)
	c.Assert(key, IsNil)
}

func (s *TestAuthSuite) TestRSASignAndVerify(c *C) {
	//priv := auth.DevPrivKeyPEM
	//pub := auth.DevPubKeyPEM

	// Try a verifier from non-PEM
	verifier, err := auth.RSAVerifierFromPEM([]byte("not pem"))
	c.Assert(verifier, IsNil)
	c.Assert(err, Equals, auth.ErrNotPEMEncoded)

	// Try a signer from non-PEM
	signer, err := auth.RSASignerFromPEM([]byte("not pem"))
	c.Assert(signer, IsNil)
	c.Assert(err, Equals, auth.ErrNotPEMEncoded)

	// Try a verifier from a private key
	verifier, err = auth.RSAVerifierFromPEM(auth.DevPrivKeyPEM)
	c.Assert(verifier, IsNil)
	c.Assert(err, Equals, auth.ErrNotRSAPublicKey)

	// Try a signer from a public key
	signer, err = auth.RSASignerFromPEM(auth.DevPubKeyPEM)
	c.Assert(signer, IsNil)
	c.Assert(err, Equals, auth.ErrNotRSAPrivateKey)

	// Get valid verifier
	verifier, err = auth.RSAVerifierFromPEM(auth.DevPubKeyPEM)
	c.Assert(err, IsNil)

	// Get valid signer
	signer, err = auth.RSASignerFromPEM(auth.DevPrivKeyPEM)
	c.Assert(err, IsNil)

	message := []byte("Four score and seven years ago our fathers brought forth on this continent a new nation")

	// Sign it
	sig, err := signer.Sign(message)
	c.Assert(err, IsNil)

	// Verify the signature
	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)
}
