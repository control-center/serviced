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

// ConvertUp & NewUUID62 taken from https://raw.githubusercontent.com/xhroot/Koderank/48db8afb0759a354bcb16759e2769d3b7621769e/uuid/uuid.go
// under MIT License
/*
Copyright (c) 2012, Antonio Rodriguez <dev@xhroot.com>

Permission is hereby granted, free of charge, to any person obtaining a copy
of this software and associated documentation files (the "Software"), to deal
in the Software without restriction, including without limitation the rights
to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
copies of the Software, and to permit persons to whom the Software is
furnished to do so, subject to the following conditions:

The above copyright notice and this permission notice shall be included in
all copies or substantial portions of the Software.

THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN
THE SOFTWARE.
*/

package utils

import (
	"crypto/rand"
	"fmt"
	"io"
	"math"
	"math/big"
)

const base62alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const base36alphabet = "0123456789abcdefghijklmnopqrstuvwxyz"

var randSource io.Reader = randReadT{}

// randReadT is a struct that implements the Reader interface and return random bytes
type randReadT struct{}

func (r randReadT) Read(p []byte) (n int, err error) {
	return rand.Read(p)
}

// NewUUID generate a new UUID
func NewUUID() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:]), nil
}

// NewUUID62 returns a base-36 UUID with no dashes.
func NewUUID62() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return ConvertUp(s, base62alphabet), nil
}

// NewUUID36 returns a base-36 UUID with no dashes.
func NewUUID36() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return ConvertUp(s, base36alphabet), nil
}

// intLength returns the length of a integer represented by the given
// bits in the given base.
func intLength(bits, base int) int {
	return int(math.Ceil(float64(bits) * math.Log10(2.0) / math.Log10(float64(base))))
}

// ConvertUp converts a hexadecimal UUID string to a base alphabet greater than 16.
func ConvertUp(oldNumber string, baseAlphabet string) string {
	n := big.NewInt(0)
	n.SetString(oldNumber, 16)

	base := big.NewInt(int64(len(baseAlphabet)))

	newNumber := make([]byte, intLength(16*8, len(baseAlphabet)))
	i := len(newNumber)

	for n.Int64() != 0 {
		i--
		_, r := n.DivMod(n, base, big.NewInt(0))
		newNumber[i] = baseAlphabet[r.Int64()]
	}
	return string(newNumber[i:])
}
