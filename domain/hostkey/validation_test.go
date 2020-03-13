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

package hostkey

import (
	. "gopkg.in/check.v1"
	"strings"
)

type validationSuite struct{}

var _ = Suite(&validationSuite{})

var DefaultKeyText = `-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxeGhO/4jJ7fPwXHjtZx+
q/Ne+fhMEzGB41aD6QKij6u0LPBWynmXdJeLdIW1N8ZFF7PdpA4qAu6ouMRvOuSJ
1qPt1hToahBxxducEp64nQ/fWN0uANjPqjlKcjj/fiSZ2ewrXYAOmnbaIQgt3fjv
VYQgdGmHA5uyROclsutOF0shyprU2x/S8uXIK1fJM/yxukcDG6GvymW0b5mqLZZA
Zmpt11QJ8YV5yiBtziSyYfiXTFs5yoydvRqmTIRm1CBnV3JYXio9fXv4C1BVTk11
miqYybTUZga1O9mykjDbrwtaigb2rP1EjQzJoMLHW27edXBZUFQjedD0N20+WkUx
0wIDAQAB
-----END PUBLIC KEY-----`

func (s *validationSuite) TestRSAKey_Success(c *C) {
	key := RSAKey{PEM: DefaultKeyText}
	err := key.ValidEntity()
	c.Assert(err, IsNil)
}

func AssertContains(c *C, actual, expected string) {
	if !strings.Contains(actual, expected) {
		c.Errorf("String \"%s\" does not contain \"%s\"", actual, expected)
	}
}

func (s *validationSuite) TestRSAKey_Empty(c *C) {
	key := RSAKey{PEM: ""}
	err := key.ValidEntity()
	c.Assert(err, NotNil)
	AssertContains(c, err.Error(), "Invalid public key PEM block")
}

func (s *validationSuite) TestRSAKey_KeyType(c *C) {
	key := RSAKey{PEM: strings.Replace(DefaultKeyText, "PUBLIC KEY", "FOOBAR KEY", 2)}
	err := key.ValidEntity()
	c.Assert(err, NotNil)
	AssertContains(c, err.Error(), "Unexpected public key PEM type")
}

func (s *validationSuite) TestRSAKey_Trailing(c *C) {
	key := RSAKey{PEM: DefaultKeyText + "\n--- FOO ---"}
	err := key.ValidEntity()
	c.Assert(err, NotNil)
	AssertContains(c, err.Error(), "Unexpected characters following public key PEM block")
}
