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
	"encoding/pem"
	"strings"

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

	// Test a private key
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

func (s *TestAuthSuite) TestPEMFromRSAPPublicKey(c *C) {
	pem := auth.DevPubKeyPEM
	expected := strings.TrimSpace(string(pem[:]))
	pub, err := auth.RSAPublicKeyFromPEM(pem)

	// Get a PEM from the public key
	out, err := auth.PEMFromRSAPublicKey(pub, nil)
	c.Assert(err, IsNil)
	actual := strings.TrimSpace(string(out[:]))
	c.Assert(actual, Equals, expected)
}

func (s *TestAuthSuite) TestPEMFromRSAPrivateKey(c *C) {
	pem := auth.DevPrivKeyPEM
	expected := strings.TrimSpace(string(pem[:]))
	priv, err := auth.RSAPrivateKeyFromPEM(pem)

	// Get a PEM from the private key
	out, err := auth.PEMFromRSAPrivateKey(priv, nil)
	c.Assert(err, IsNil)
	actual := strings.TrimSpace(string(out[:]))
	c.Assert(actual, Equals, expected)
}

func (s *TestAuthSuite) TestGenerateRSAKeyPairPEM(c *C) {

	var samplePriv *rsa.PrivateKey
	var samplePub *rsa.PublicKey

	pub, priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	// Make sure pub is a public key and priv is a private key
	publicKey, err := auth.RSAPublicKeyFromPEM(pub)
	c.Assert(err, IsNil)
	c.Assert(publicKey, FitsTypeOf, samplePub)
	privateKey, err := auth.RSAPrivateKeyFromPEM(priv)
	c.Assert(err, IsNil)
	c.Assert(privateKey, FitsTypeOf, samplePriv)

	// Make sure the keys are a pair
	expectedPublicKey := privateKey.Public()
	expectedpublicPEM, err := auth.PEMFromRSAPublicKey(expectedPublicKey, nil)
	c.Assert(err, IsNil)
	c.Assert(pub, DeepEquals, expectedpublicPEM)

	// Headers are empty
	pubBlock, pubRest := pem.Decode(pub)
	c.Assert(len(pubBlock.Headers), Equals, 0)
	c.Assert(len(pubRest), Equals, 0)

	privBlock, privRest := pem.Decode(priv)
	c.Assert(len(privBlock.Headers), Equals, 0)
	c.Assert(len(privRest), Equals, 0)

	// Test signing and verifying
	// Get valid verifier
	verifier, err := auth.RSAVerifierFromPEM(pub)
	c.Assert(err, IsNil)

	// Get valid signer
	signer, err := auth.RSASignerFromPEM(priv)
	c.Assert(err, IsNil)

	message := []byte("Four score and seven years ago our fathers brought forth on this continent a new nation")

	// Sign it
	sig, err := signer.Sign(message)
	c.Assert(err, IsNil)

	// Verify the signature
	err = verifier.Verify(message, sig)
	c.Assert(err, IsNil)

	// Generate another pair, and make sure they are unique
	newpub, newpriv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)
	c.Assert(newpub, Not(DeepEquals), pub)
	c.Assert(newpriv, Not(DeepEquals), priv)

	// Pass in some headers and make sure they survive
	headers := make(map[string]string)
	headers["header1"] = "value1"
	headers["header2"] = "value2"
	pubwheader, privwheader, err := auth.GenerateRSAKeyPairPEM(headers)
	c.Assert(err, IsNil)
	pubBlock, _ = pem.Decode(pubwheader)
	privBlock, _ = pem.Decode(privwheader)
	c.Assert(pubBlock.Headers, DeepEquals, headers)
	c.Assert(privBlock.Headers, DeepEquals, headers)

}
