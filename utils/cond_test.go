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

package utils_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

func TestChannelCond(t *testing.T) { TestingT(t) }

type ChannelCondSuite struct{}

var (
	_ = Suite(&ChannelCondSuite{})
)

func (s *ChannelCondSuite) TestChannelCond(c *C) {
	cond := utils.NewChannelCond()
	x := make([]bool, 1)

	ch := cond.Wait()

	go func() {
		<-ch
		fmt.Println("HJIHIHI")
		x[0] = true
	}()

	time.Sleep(1 * time.Second)
	c.Assert(x[0], Equals, false)
	cond.Broadcast()
	time.Sleep(1 * time.Second)
	c.Assert(x[0], Equals, true)
}
