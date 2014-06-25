package utils

import (
	"os/exec"
	"syscall"
)

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