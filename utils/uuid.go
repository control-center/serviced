// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// base62 uuid taken from https://raw.githubusercontent.com/xhroot/Koderank/48db8afb0759a354bcb16759e2769d3b7621769e/uuid/uuid.go
// under MIT License

package utils

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
)

const base62alphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

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

// NewUUID62 createa a base-62 UUID.
func NewUUID62() (string, error) {
	b := make([]byte, 16)
	_, err := randSource.Read(b)
	if err != nil {
		return "", err
	}
	s := fmt.Sprintf("%x", b)
	return ConvertUp(s, base62alphabet), nil
}

// ConvertUp converts a hexadecimal UUID string to a base alphabet greater than
// 16. It is used here to compress a 32 character UUID down to 23 URL friendly
// characters.
func ConvertUp(oldNumber string, baseAlphabet string) string {
	n := big.NewInt(0)
	n.SetString(oldNumber, 16)

	base := big.NewInt(int64(len(baseAlphabet)))

	newNumber := make([]byte, 23) //converted size of max base-62 uuid
	i := len(newNumber)

	for n.Int64() != 0 {
		i--
		_, r := n.DivMod(n, base, big.NewInt(0))
		newNumber[i] = baseAlphabet[r.Int64()]
	}
	return string(newNumber[i:])
}
