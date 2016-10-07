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
	jwt "github.com/dgrijalva/jwt-go"
	. "gopkg.in/check.v1"
	"time"
)

func (s *TestAuthSuite) TestHostExpired(c *C) {
	hostid := "fakehost"
	expires := jwt.TimeFunc().Add(time.Minute).Unix()
	auth.SetHostExpiration(hostid, expires)

	hasExpired := auth.HostExpired(hostid)
	c.Assert(hasExpired, Equals, false)
}

func (s *TestAuthSuite) TestHostExpiredExpires(c *C) {
	hostid := "fakehost"
	expires := jwt.TimeFunc().Add(time.Second).Unix()
	auth.SetHostExpiration(hostid, expires)

	fakenow := jwt.TimeFunc().Add(time.Minute)
	auth.At(fakenow, func() {
		hasExpired := auth.HostExpired(hostid)
		c.Assert(hasExpired, Equals, true)
	})
}
