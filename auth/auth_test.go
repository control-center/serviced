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
	"testing"

	"github.com/control-center/serviced/auth"

	. "gopkg.in/check.v1"
)

type TestAuthSuite struct {
	masterPubPEM    []byte
	masterPrivPEM   []byte
	delegatePubPEM  []byte
	delegatePrivPEM []byte
}

var (
	mPub, mPriv, dPub, dPriv []byte
)

func init() {
	mPub, mPriv, _ = auth.GenerateRSAKeyPairPEM(nil)
	dPub, dPriv, _ = auth.GenerateRSAKeyPairPEM(nil)
}

var _ = Suite(&TestAuthSuite{})

func TestAuth(t *testing.T) { TestingT(t) }

func (s *TestAuthSuite) SetUpTest(c *C) {
	s.masterPubPEM, s.masterPrivPEM = mPub, mPriv
	s.delegatePubPEM, s.delegatePrivPEM = dPub, dPriv

	auth.LoadMasterKeysFromPEM(mPub, mPriv)
	auth.LoadDelegateKeysFromPEM(mPub, dPriv)
}

func (s *TestAuthSuite) TearDownTest(c *C) {
	auth.ClearKeys()
}
