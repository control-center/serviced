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
	"os"

	"github.com/control-center/serviced/auth"
	. "gopkg.in/check.v1"
)

func (s *TestAuthSuite) TestDumpLoadKeys(c *C) {
	// We'll need a temp dir:
	tmpDir := c.MkDir()
	hostKeyFile := fmt.Sprintf("%s/host", tmpDir)
	masterKeyFile := fmt.Sprintf("%s/master", tmpDir)

	// Create some keys
	delegate_pub, delegate_priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	master_pub, master_priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	// Dump the keys:
	err = auth.DumpPEMKeyPairToFile(hostKeyFile, master_pub, delegate_priv)
	c.Assert(err, IsNil)
	err = auth.DumpPEMKeyPairToFile(masterKeyFile, master_pub, master_priv)
	c.Assert(err, IsNil)

	// Load the keys
	err = auth.LoadDelegateKeysFromFile(hostKeyFile)
	c.Assert(err, IsNil)
	err = auth.CreateOrLoadMasterKeys(masterKeyFile)
	c.Assert(err, IsNil)

	// Test signing and verifying
	//
	msg_delegate_signed := []byte("This was signed by the delegate")
	msg_master_signed := []byte("This was signed by the master")

	// Test VerifyMasterSignature
	// Sign something as master
	master_sig, err := auth.SignAsMaster(msg_master_signed)
	c.Assert(err, IsNil)

	// verify the signature
	err = auth.VerifyMasterSignature(msg_master_signed, master_sig)
	c.Assert(err, IsNil)

	// Test SignAsDelegate
	//  Sign something as delegate
	delegate_sig, err := auth.SignAsDelegate(msg_delegate_signed)
	c.Assert(err, IsNil)

	verifier, err := auth.RSAVerifierFromPEM(delegate_pub)
	c.Assert(err, IsNil)

	// Verify the signature
	err = verifier.Verify(msg_delegate_signed, delegate_sig)
	c.Assert(err, IsNil)
}

func (s *TestAuthSuite) TestCreateMasterKeys(c *C) {
	auth.ClearKeys()
	// We'll need a temp dir:
	tmpDir := c.MkDir()
	masterKeyFile := fmt.Sprintf("%s/.keys/master", tmpDir)
	// hostKeyFile := fmt.Sprintf("%s/delegate", tmpDir)

	// Master key file should not exist
	_, err := os.Stat(masterKeyFile)
	c.Assert(os.IsNotExist(err), Equals, true)

	// Generate new keys and write them to file
	err = auth.CreateOrLoadMasterKeys(masterKeyFile)
	c.Assert(err, IsNil)

	// File should now exist
	_, err = os.Stat(masterKeyFile)
	c.Assert(err, IsNil)

	// Sign something as master
	msg_master_signed := []byte("This was signed by the master")
	master_sig, err := auth.SignAsMaster(msg_master_signed)
	c.Assert(err, IsNil)

	// verify the signature
	err = auth.VerifyMasterSignature(msg_master_signed, master_sig)
	c.Assert(err, IsNil)

	// Load them again
	err = auth.CreateOrLoadMasterKeys(masterKeyFile)
	c.Assert(err, IsNil)

	// keys should not have changed
	err = auth.VerifyMasterSignature(msg_master_signed, master_sig)
	c.Assert(err, IsNil)
}

func (s *TestAuthSuite) TestGetMasterPublicKey(c *C) {
	auth.ClearKeys()

	// Try to get master public key without loading anything
	mpk, err := auth.GetMasterPublicKey()
	c.Assert(err, Equals, auth.ErrNoPublicKey)
	c.Assert(mpk, IsNil)

	// Create some keys and load them
	pub, priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	err = auth.LoadDelegateKeysFromPEM(pub, priv)
	c.Assert(err, IsNil)

	// Call GetMasterPublicKey() and make sure it matches
	mpk, err = auth.GetMasterPublicKey()
	c.Assert(err, IsNil)
	mpkPEM, err := auth.PEMFromRSAPublicKey(mpk, nil)
	c.Assert(mpkPEM, DeepEquals, pub)

	// Test that it falls back to master keys if delegate keys don't exist
	auth.ClearKeys()

	// Create some keys and load them
	mpub, mpriv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	err = auth.LoadMasterKeysFromPEM(mpub, mpriv)
	c.Assert(err, IsNil)

	// Call GetMasterPublicKey() and make sure it matches
	mpk, err = auth.GetMasterPublicKey()
	c.Assert(err, IsNil)
	mpkPEM, err = auth.PEMFromRSAPublicKey(mpk, nil)
	c.Assert(mpkPEM, DeepEquals, mpub)
	c.Assert(mpkPEM, Not(DeepEquals), pub)
}

func (s *TestAuthSuite) TestLoadKeysFromPEM(c *C) {
	auth.ClearKeys()
	// Create some keys
	delegate_pub, delegate_priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	master_pub, master_priv, err := auth.GenerateRSAKeyPairPEM(nil)
	c.Assert(err, IsNil)

	// Pub for priv should fail
	err = auth.LoadDelegateKeysFromPEM(master_pub, master_pub)
	c.Assert(err, Equals, auth.ErrNotRSAPrivateKey)

	// Priv for pub should fail
	err = auth.LoadDelegateKeysFromPEM(delegate_priv, delegate_priv)
	c.Assert(err, Equals, auth.ErrNotRSAPublicKey)

	// Passing them correctly should work
	err = auth.LoadDelegateKeysFromPEM(master_pub, delegate_priv)
	c.Assert(err, IsNil)

	// Repeat for master keys
	// Pub for priv should fail
	err = auth.LoadMasterKeysFromPEM(master_pub, master_pub)
	c.Assert(err, Equals, auth.ErrNotRSAPrivateKey)

	// Priv for pub should fail
	err = auth.LoadMasterKeysFromPEM(master_priv, master_priv)
	c.Assert(err, Equals, auth.ErrNotRSAPublicKey)

	// Passing them both should work
	err = auth.LoadMasterKeysFromPEM(master_pub, master_priv)
	c.Assert(err, IsNil)

	// Test Signing and Verifying
	//
	msg_delegate_signed := []byte("This was signed by the delegate")
	msg_master_signed := []byte("This was signed by the master")

	// Test VerifyMasterSignature
	// Sign something as master
	master_sig, err := auth.SignAsMaster(msg_master_signed)
	c.Assert(err, IsNil)

	// verify the signature
	err = auth.VerifyMasterSignature(msg_master_signed, master_sig)
	c.Assert(err, IsNil)

	// Test SignAsDelegate
	//  Sign something as delegate
	delegate_sig, err := auth.SignAsDelegate(msg_delegate_signed)
	c.Assert(err, IsNil)

	verifier, err := auth.RSAVerifierFromPEM(delegate_pub)
	c.Assert(err, IsNil)

	// Verify the signature
	err = verifier.Verify(msg_delegate_signed, delegate_sig)
	c.Assert(err, IsNil)

}
