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
