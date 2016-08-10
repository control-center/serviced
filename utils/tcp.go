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

package utils

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"net"
	"strconv"
	"strings"
)

var (
	endian = binary.BigEndian
	// ErrInvalidTCPAddress is thrown when a specified address can't be packed
	ErrInvalidTCPAddress = errors.New("Invalid TCP address")
)

// PackTCPAddress packs a TCP address (IP and port) to 6 bytes
func PackTCPAddress(ip string, port uint16) ([]byte, error) {
	var result bytes.Buffer

	// Pack the port to 2 bytes
	portBuf := make([]byte, 2)
	endian.PutUint16(portBuf, port)
	result.Write(portBuf)

	// Pack the ip address to 4 bytes
	ipaddr := net.ParseIP(ip)
	if ipaddr == nil {
		return nil, ErrInvalidTCPAddress
	}
	ipbytes := ipaddr.To4()
	result.Write(ipbytes)

	return result.Bytes(), nil
}

// PackTCPAddressString packs a TCP address represented as a string ("IP:port")
// to 6 bytes
func PackTCPAddressString(address string) ([]byte, error) {
	parts := strings.Split(address, ":")
	if len(parts) != 2 {
		return nil, ErrInvalidTCPAddress
	}
	ip := parts[0]
	intport, err := strconv.Atoi(parts[1])
	if err != nil || uint16(intport) == 0 {
		return nil, ErrInvalidTCPAddress
	}
	port := uint16(intport)
	return PackTCPAddress(ip, port)
}

// UnpackTCPAddress unpacks a 6-byte representation of a TCP address produced
// by PackTCPAddress into an IP and port
func UnpackTCPAddress(packed []byte) (ip string, port uint16) {

	// Read off the port
	buf := bytes.NewBuffer(packed)
	binary.Read(buf, endian, &port)

	// Read off the IP
	ipbytes := make([]byte, 4)
	buf.Read(ipbytes)
	ip = net.IP(ipbytes).String()

	return
}

// UnpackTCPAddressToString unpacks a 6-byte representation of a TCP address
// produced by PackTCPAddress into a string of the format "IP:port"
func UnpackTCPAddressToString(packed []byte) string {
	ip, port := UnpackTCPAddress(packed)
	return fmt.Sprintf("%s:%d", ip, port)
}
