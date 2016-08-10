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

package utils_test

import (
	"testing"

	. "github.com/control-center/serviced/utils"
)

func TestPackTCPAddresses(t *testing.T) {
	var (
		ip   string = "172.12.0.1"
		port uint16 = 11211
		addr string = "172.12.0.1:11211"
	)
	packed, err := PackTCPAddress(ip, port)
	if len(packed) != 6 {
		t.Fail()
	}
	if err != nil {
		t.Fail()
	}
	ip2, port2 := UnpackTCPAddress(packed)
	if ip != ip2 {
		t.Fail()
	}
	if port != port2 {
		t.Fail()
	}
	spacked, err := PackTCPAddressString(addr)
	if err != nil {
		t.Fail()
	}
	if string(packed) != string(spacked) {
		t.Fail()
	}
	addr2 := UnpackTCPAddressToString(spacked)
	if addr2 != addr {
		t.Fail()
	}

	for _, invalidaddr := range []string{
		"1.2.3.4:abc",
		"1.2.3.4:65536",
		"not an address",
		"666.666.666.666:123",
	} {
		if _, err := PackTCPAddressString(invalidaddr); err != ErrInvalidTCPAddress {
			t.Logf("Invalid address didn't produce an error: %s", invalidaddr)
			t.Fail()
		}
	}
}
