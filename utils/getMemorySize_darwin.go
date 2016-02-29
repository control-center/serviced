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

	if rc, errno := C.sysctl(&mib[0], 2, unsafe.Pointer(&value), &length, unused, 0); rc == -1 {
		return 0, fmt.Errorf("unable to get memory size: errno=%d", errno)
	}

	return uint64(value), nil
}
