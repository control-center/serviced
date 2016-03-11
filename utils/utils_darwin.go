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

// #include <sys/types.h>
// #include <sys/sysctl.h>
import "C"

import (
	"fmt"
	"net"
	"unsafe"
	"syscall"
)

const (
	ioctlTermioFlag = syscall.TIOCGETA
)

func determinePlatform() int {
	return Darwin
}

// getHostID retrieves the system's unique id
func getHostID() (hostid string, err error) {
	var mib []C.int = []C.int{ C.CTL_KERN, C.KERN_HOSTID }
	var value C.int64_t = 0
	var length C.size_t = 8
	var unused unsafe.Pointer

	if rc, errno := C.sysctl(&mib[0], 2, unsafe.Pointer(&value), &length, unused, 0); rc == -1 {
		return "", fmt.Errorf("unable to get hostID: errno=%d", errno)
	}

	if uint64(value) > 0 {
		return fmt.Sprintf("%x", uint64(value)), nil
	}

	hostIP, err := getHostIPv4Addr()
	if err != nil {
		return "", err
	}

	var numericIP uint32
	numericIP |= uint32(hostIP[0])
	numericIP |= uint32(hostIP[1]) << 8
	numericIP |= uint32(hostIP[2]) << 16
	numericIP |= uint32(hostIP[3]) << 24

	hostID := uint32(numericIP<<16 | numericIP>>16)
	return fmt.Sprintf("%x", hostID), nil
}

func getHostIPv4Addr() (net.IP, error) {
	ips, _ := GetIPv4Addresses()
	if len(ips) == 0 {
		return nil, fmt.Errorf("unable to IP4 addrs for hostID")
	}
	// Use the first IP (if more than one)
	return net.ParseIP(ips[0]).To4(), nil
}

// getMemorySize attempts to get the size of the installed RAM.
func getMemorySize() (size uint64, err error) {
	var mib []C.int = []C.int{ C.CTL_HW, C.HW_MEMSIZE }
	var value C.int64_t = 0
	var length C.size_t = 8
	var unused unsafe.Pointer

	if rc, errno := C.sysctl(&mib[0], 2, unsafe.Pointer(&value), &length, unused, 0); rc == -1 {
		return 0, fmt.Errorf("unable to get memory size: errno=%d", errno)
	}

	return uint64(value), nil
}

// Returns a list of network routes
func getRoutes() (routes []RouteEntry, err error) {
	return []RouteEntry{}, nil
}