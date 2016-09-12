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

package auth

import (
	"crypto"
	"io/ioutil"
	"os"
	"path"
)

const (
	DelegateKeyFileName = ".keys/delegate.keys"
	MasterKeyFileName   = ".keys/master.keys"
)

var (
	delegateKeys HostKeys
	masterKeys   MasterKeys
)

type HostKeys struct {
	masterPublic crypto.PublicKey
	localPrivate crypto.PrivateKey
}

type MasterKeys struct {
	public  crypto.PublicKey
	private crypto.PrivateKey
}

func (h *HostKeys) Sign(message []byte) ([]byte, error) {
	signer, err := RSASigner(h.localPrivate)
	if err != nil {
		return nil, err
	}
	return signer.Sign(message)
}

func (h *HostKeys) Verify(message, signature []byte) error {
	verifier, err := RSAVerifier(h.masterPublic)
	if err != nil {
		return err
	}
	return verifier.Verify(message, signature)
}

func (m *MasterKeys) Sign(message []byte) ([]byte, error) {
	signer, err := RSASigner(m.private)
	if err != nil {
		return nil, err
	}
	return signer.Sign(message)
}

func (m *MasterKeys) Verify(message, signature []byte) error {
	verifier, err := RSAVerifier(m.public)
	if err != nil {
		return err
	}
	return verifier.Verify(message, signature)
}

// SignAsDelegate signs the given message with the private key local
// to the delegate running this process.
func SignAsDelegate(message []byte) ([]byte, error) {
	if delegateKeys.localPrivate == nil {
		return nil, ErrNotRSAPrivateKey
	}
	return delegateKeys.Sign(message)
}

// SignAsMaster signs the given message with the master's private key
// will return an error if the delegate running this process is not the master
func SignAsMaster(message []byte) ([]byte, error) {
	if masterKeys.private == nil {
		return nil, ErrNotRSAPrivateKey
	}
	return masterKeys.Sign(message)
}

// VerifyMasterSignature verifies that a given message was signed by the master
// whose public key we have.
func VerifyMasterSignature(message, signature []byte) error {
	if delegateKeys.masterPublic == nil {
		if masterKeys.public == nil {
			return ErrNoPublicKey
		}
		return masterKeys.Verify(message, signature)
	}
	return delegateKeys.Verify(message, signature)
}

// GetMasterPublicKey() returns the public key of the master
//  If the host keys have not been loaded yet, it checks to see if
//  master keys have been loaded.  If neither exists, returns ErrNoPublicKey
func GetMasterPublicKey() (crypto.PublicKey, error) {
	if delegateKeys.masterPublic == nil {
		if masterKeys.public == nil {
			return nil, ErrNoPublicKey
		}
		return masterKeys.public, nil
	}
	return delegateKeys.masterPublic, nil
}

// LoadKeysFromFile loads keys from a file on disk.
func LoadDelegateKeysFromFile(filename string) error {
	pub, priv, err := LoadKeyPairFromFile(filename)
	if err != nil {
		return err
	}

	delegateKeys = HostKeys{pub, priv}
	return nil
}

// CreateOrLoadMasterKeys will load the master keys from disk
//  If the file does not exist, it will generate new keys and
//  write them to disk.
func CreateOrLoadMasterKeys(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		os.MkdirAll(path.Dir(filename), os.ModeDir|0700)
		pub, priv, err := GenerateRSAKeyPairPEM(nil)
		if err != nil {
			return err
		}
		err = DumpPEMKeyPairToFile(filename, pub, priv)
		if err != nil {
			return err
		}

	} else if err != nil {
		return err
	}

	publicKey, privateKey, err := LoadKeyPairFromFile(filename)
	if err != nil {
		return err
	}

	masterKeys = MasterKeys{publicKey, privateKey}

	return nil
}

// DumpPEMKeyPairToFile dumps PEM-encoded public and private keys to a single file
func DumpPEMKeyPairToFile(filename string, public, private []byte) error {
	data, err := DumpPEMKeyPair(public, private)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(filename, data, 0666)
}

// LoadPEMKeyPair loads a private/public key pair from a reader over PEM-encoded data.
//  The private key is first, the public key is second.
func LoadKeyPairFromFile(filename string) (public crypto.PublicKey, private crypto.PrivateKey, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, nil, err
	}
	return LoadKeyPair(data)
}

// ClearKeys wipes the current state
func ClearKeys() {
	delegateKeys, masterKeys = HostKeys{}, MasterKeys{}
}
