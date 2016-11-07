// Copyright 2014 The Serviced Authors.
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

package utils_test

import (
	"testing"
	"time"

	"github.com/control-center/serviced/utils"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type valueChangePublisherSuite struct {
}

var _ = Suite(&valueChangePublisherSuite{})

func (s *valueChangePublisherSuite) Test_Synchronous(c *C) {
	initial := "foo"
	p := utils.NewValueChangePublisher(initial)
	value, _ := p.Get()
	c.Assert(value, Equals, initial)

	expected := "bar"
	p.Set(expected)
	value, _ = p.Get()
	c.Assert(value, Equals, expected)
}

func (s *valueChangePublisherSuite) Test_Notify(c *C) {
	initial := "foo"
	p := utils.NewValueChangePublisher(initial)
	value, notify := p.Get()

	expected := "bar"
	ready, ok := make(chan struct{}), make(chan struct{})
	go func() {
		select {
		case <-ready:
			close(ok)
		case <-notify:
			v, _ := p.Get()
			c.Assert(v, Equals, expected)
		case <-time.After(time.Second):
			c.Fail()
		}
	}()

	close(ready)
	<-ok
	c.Assert(value, Equals, initial)
	p.Set(expected)

	// notify does not block; the value was already changed
	select {
	case <-notify:
		value, _ = p.Get()
		c.Assert(value, Equals, expected)
	case <-time.After(time.Second):
		c.Fail()
	}
}
