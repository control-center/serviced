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
	"unsafe"
)

// getMemorySize attempts to get the size of the installed RAM.
func getMemorySize() (size uint64, err error) {
	var mib []C.int = []C.int{ C.CTL_HW, C.HW_MEMSIZE }
	var value C.int64_t = 0
	var length C.size_t = 8
	var unused unsafe.Pointer

	if -1 == C.sysctl(&mib[0], 2, unsafe.Pointer(&value), &length, unused, 0) {
		return 0, fmt.Errorf("oops")
	}

	return uint64(value), nil
}

//func getExecPath() (string, error) {
//	var bufferSize C.uint32_t = 1024
//	buffer := make([]C.char, bufferSize)
//
//	result := C._NSGetExecutablePath(&buffer[0], &bufferSize)
//	if result == -1 {
//		buffer := make([]C.char, bufferSize)
//		result = C._NSGetExecutablePath(&buffer[0], &bufferSize)
//		if result == -1 {
//			return "", fmt.Errorf("Unable to determine path")
//		}
//	}
//	return C.GoStringN(&buffer[0], C.int(bufferSize)), nil
//}