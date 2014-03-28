// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package validation

import (
	"fmt"
	"net"
	"strings"
)

func NotEmpty(fieldName string, value string) Violation {
	if strings.TrimSpace(value) == "" {
		return NewViolation(fmt.Sprintf("empty string for %v", fieldName))
	}
	return nil
}

func IsIP(value string) Violation {
	if nil == net.ParseIP(value) {
		return NewViolation(fmt.Sprintf("invalid IP Address %s", value))
	}
	return nil
}
