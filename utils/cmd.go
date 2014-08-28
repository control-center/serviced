// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package utils

import (
	"os/exec"
	"syscall"
)

// GetExitStatus tries to extract the exit code from an error
func GetExitStatus(err error) (int, bool) {
	if err != nil {
		if e, ok := err.(*exec.ExitError); ok {
			if status, ok := e.Sys().(syscall.WaitStatus); ok {
				return status.ExitStatus(), true
			}
		}
		return 0, false
	}
	return 0, true
}