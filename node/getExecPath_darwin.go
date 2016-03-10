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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package node

// #import <mach-o/dyld.h>
import "C"

import (
	"fmt"
)

func getExecPath() (string, error) {
	var bufferSize C.uint32_t = 1024
	buffer := make([]C.char, bufferSize)

	result := C._NSGetExecutablePath(&buffer[0], &bufferSize)
	if result == -1 {
		buffer := make([]C.char, bufferSize)
		result = C._NSGetExecutablePath(&buffer[0], &bufferSize)
		if result == -1 {
			return "", fmt.Errorf("Unable to determine path")
		}
	}
	return C.GoStringN(&buffer[0], C.int(bufferSize)), nil
}