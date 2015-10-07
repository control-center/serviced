// Copyright 2015 The Serviced Authors.
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

// +build root,integration

package web

import (
	"unsafe"
)

// #cgo LDFLAGS: -lcrypt
// #define _GNU_SOURCE
// #include <crypt.h>
// #include <stdlib.h>
import "C"

// Wrapper for C library crypt_r
// This function is here to support creating users with known passwords for integration tests.
func crypt(key, salt string) string {
	cdata := C.struct_crypt_data{}
	ckey := C.CString(key)
	csalt := C.CString(salt)
	result := C.GoString(C.crypt_r(ckey, csalt, &cdata))
	C.free(unsafe.Pointer(ckey))
	C.free(unsafe.Pointer(csalt))
	return result
}
