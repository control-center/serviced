// Copyright 2015 The Serviced Authors.
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
package jwt

import (
	"reflect"

	. "gopkg.in/check.v1"
)

func (t *JwtSuite) TestNewInstance(c *C) {
	jwt, err := NewInstance(DefaultSigningAlgorithm, t.getDummyKeyLookup())
	c.Assert(jwt, NotNil)
	c.Assert(err, IsNil)

	jwtFacade, _ := jwt.(*jwtFacade)
	c.Assert(jwtFacade, NotNil)
	actualKLF := reflect.ValueOf(jwtFacade.keyLookup)
	expectedKLF := reflect.ValueOf(t.getDummyKeyLookup())
	c.Assert(actualKLF, Equals, expectedKLF)

	signerFacade, _ := jwtFacade.signer.(*signerFacade)
	c.Assert(signerFacade, NotNil)
	c.Assert(signerFacade.signingMethod.Alg(), Equals, DefaultSigningAlgorithm)

}

func (t *JwtSuite) TestNewInstanceWithInvalidAlgorithm(c *C) {
	jwt, err := NewInstance("invalidAlgorithm", t.getDummyKeyLookup())
	c.Assert(jwt, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "algorithm \"invalidAlgorithm\" is invalid; must be one of [\"HS256\"]")
}

func (t *JwtSuite) TestNewInstanceWithoutKeyLookup(c *C) {
	jwt, err := NewInstance(DefaultSigningAlgorithm, nil)
	c.Assert(jwt, IsNil)
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "keyLookup can not be nil")
}
