// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package utils

import (
	"fmt"
	"os"
)

var urandomFilename = "/dev/urandom"

// NewUUID generate a new UUID
func NewUUID() (string, error) {
	f, err := os.Open(urandomFilename)
	if err != nil {
		return "", err
	}
	b := make([]byte, 16)
	defer f.Close()
	f.Read(b)
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}


