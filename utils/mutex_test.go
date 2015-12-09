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
//
// +build unit

package utils_test

import (
	"math/rand"
	"testing"

	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func TestMutex(t *testing.T) { TestingT(t) }

type MutexSuite struct{}

var (
	_ = Suite(&MutexSuite{})
)

func randomKey(n int) string {
	b := make([]byte, n)
	for i := range b {
		b[i] = letterBytes[rand.Intn(len(letterBytes))]
	}
	return string(b)
}

func (s *MutexSuite) TestMutexMap(c *C) {
	m := utils.NewMutexMap()
	ch := make(chan bool)
	for i := 0; i < 10; i++ {
		key := randomKey(10)
		for j := 0; j < 10; j++ {
			go func() {
				for k := 0; k < 1000; k++ {
					m.LockKey(key)
					m.UnlockKey(key)
				}
				ch <- true
			}()
		}
	}
	for i := 0; i < 100; i++ {
		<-ch
	}
}
