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
	"path/filepath"
	"sync"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
	"github.com/fsnotify/fsnotify"
)

const (
	DelegateKeyFileName = "delegate.keys"
	MasterKeyFileName   = ".keys/master.keys"
)

var (
	delegateKeys HostKeys
	masterKeys   MasterKeys
	mKeyLock     sync.RWMutex
	dKeyCond     = utils.NewChannelCond()
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
	dKeyCond.RLock()
	defer dKeyCond.RUnlock()
	if delegateKeys.localPrivate == nil {
		return nil, ErrNoPrivateKey
	}
	return delegateKeys.Sign(message)
}

// SignAsMaster signs the given message with the master's private key
// will return an error if the delegate running this process is not the master
func SignAsMaster(message []byte) ([]byte, error) {
	mKeyLock.RLock()
	defer mKeyLock.RUnlock()
	if masterKeys.private == nil {
		return nil, ErrNoPrivateKey
	}
	return masterKeys.Sign(message)
}

// VerifyMasterSignature verifies that a given message was signed by the master
// whose public key we have.
func VerifyMasterSignature(message, signature []byte) error {
	if err := verifyMasterSignatureAsMaster(message, signature); err != ErrNoPublicKey {
		return err
	}

	dKeyCond.RLock()
	defer dKeyCond.RUnlock()
	if delegateKeys.masterPublic == nil {
		return ErrNoPublicKey
	}
	return delegateKeys.Verify(message, signature)
}

func verifyMasterSignatureAsMaster(message, signature []byte) error {
	mKeyLock.RLock()
	defer mKeyLock.RUnlock()
	if masterKeys.public == nil {
		return ErrNoPublicKey
	}
	return masterKeys.Verify(message, signature)

}

// GetMasterPublicKey() returns the public key of the master
//  It first checks to see if we have a set of master keys (i.e. we are the master)
//  It then checks the delegate keys.  If neither exists, returns ErrNoPublicKey
func GetMasterPublicKey() (crypto.PublicKey, error) {
	// check the master keys first
	if key, err := getMasterPublicKeyAsMaster(); err == nil {
		return key, err
	}

	dKeyCond.RLock()
	defer dKeyCond.RUnlock()
	if delegateKeys.masterPublic == nil {
		return nil, ErrNoPublicKey
	}
	return delegateKeys.masterPublic, nil
}

func getMasterPublicKeyAsMaster() (crypto.PublicKey, error) {
	mKeyLock.RLock()
	defer mKeyLock.RUnlock()
	if masterKeys.public == nil {
		return nil, ErrNoPublicKey
	}
	return masterKeys.public, nil
}

func getMasterPrivateKey() (crypto.PrivateKey, error) {
	mKeyLock.RLock()
	defer mKeyLock.RUnlock()
	if masterKeys.private == nil {
		return nil, ErrNoPrivateKey
	}
	return masterKeys.private, nil
}

// LoadKeysFromFile loads keys from a file on disk.
func LoadDelegateKeysFromFile(filename string) error {
	pub, priv, err := LoadKeyPairFromFile(filename)
	if err != nil {
		return err
	}
	updateDelegateKeys(pub, priv)
	return nil
}

// LoadMasterKeys sets the current master key pair to the one specified
func LoadMasterKeys(public crypto.PublicKey, private crypto.PrivateKey) {
	mKeyLock.Lock()
	defer mKeyLock.Unlock()
	masterKeys = MasterKeys{public, private}
}

// CreateOrLoadMasterKeys will load the master keys from disk
//  If the file does not exist, it will generate new keys and
//  write them to disk.
func CreateOrLoadMasterKeys(filename string) error {
	if _, err := os.Stat(filename); os.IsNotExist(err) {
		if err = os.MkdirAll(path.Dir(filename), os.ModeDir|0700); err != nil {
			return err
		}

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

	return LoadMasterKeyFile(filename)
}

// LoadMasterKeyFile will load the master keys from disk if
//  the file exists.  If the file does not exist, it will
//  return an error
func LoadMasterKeyFile(filename string) error {
	publicKey, privateKey, err := LoadKeyPairFromFile(filename)
	if err != nil {
		return err
	}

	LoadMasterKeys(publicKey, privateKey)

	return nil
}

// DumpPEMKeyPairToFile dumps PEM-encoded public and private keys to a single file
func DumpPEMKeyPairToFile(filename string, public, private []byte) error {
	data, err := DumpRSAPEMKeyPair(public, private)
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
	return LoadRSAKeyPairPackage(data)
}

// LoadDelegateKeysFromPEM loads the local delegate keys (master public, delegate private)
//  from PEM data passed in directly
//  Useful mostly for writing tests
func LoadDelegateKeysFromPEM(public, private []byte) error {
	pub, priv, err := LoadRSAKeyPair(public, private)
	if err != nil {
		return err
	}
	updateDelegateKeys(pub, priv)
	return nil
}

// LoadMasterKeysFromPEM loads the local master keys from PEM data passed in directly
//  Useful mostly for writing tests
func LoadMasterKeysFromPEM(public, private []byte) error {
	pub, priv, err := LoadRSAKeyPair(public, private)
	if err != nil {
		return err
	}

	mKeyLock.Lock()
	defer mKeyLock.Unlock()
	masterKeys = MasterKeys{pub, priv}
	return nil
}

// ClearKeys wipes the current state
func ClearKeys() {
	mKeyLock.Lock()
	masterKeys = MasterKeys{}
	mKeyLock.Unlock()
	updateDelegateKeys(nil, nil)
}

// WaitForDelegateKeys blocks until delegate keys are defined.
func WaitForDelegateKeys(cancel <-chan interface{}) <-chan struct{} {
	ch := make(chan struct{})
	go func() {
		for delegateKeys.localPrivate == nil || delegateKeys.masterPublic == nil {
			select {
			case <-dKeyCond.Wait():
			case <-cancel: // Receive from nil channel never returns, so this is fine
			}
		}
		close(ch)
	}()
	return ch
}

func NotifyOnKeyChange() <-chan struct{} {
	return dKeyCond.Wait()
}

// WatchDelegateKeyFile watches the delegate key file on the filesystem and
// updates the internal delegate keys when changes are detected.
func WatchDelegateKeyFile(filename string, cancel <-chan interface{}) error {
	filename = filepath.Clean(filename)

	log := log.WithFields(logrus.Fields{
		"keyfile": filename,
	})

	loadKeys := func() {
		if err := LoadDelegateKeysFromFile(filename); err != nil {
			log.WithError(err).Warn("Unable to load delegate keys from file. Continuing to watch for changes")
		} else {
			log.Info("Loaded delegate keys from file")
		}
	}

	// Try an initial load without any file changes
	loadKeys()

	filechanges, err := NotifyOnChange(filename, fsnotify.Write|fsnotify.Create, cancel)
	if err != nil {
		return err
	}
	for _ = range filechanges {
		loadKeys()
	}
	return nil
}

func updateDelegateKeys(pub crypto.PublicKey, priv crypto.PrivateKey) {
	dKeyCond.Lock()
	delegateKeys = HostKeys{pub, priv}
	dKeyCond.Unlock()
	dKeyCond.Broadcast()
}
