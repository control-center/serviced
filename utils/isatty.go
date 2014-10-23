package utils

import (
	"os"
	"syscall"
	"unsafe"
)

// Isatty returns true if f is a TTY, false otherwise.
func Isatty(f *os.File) bool {
	var t [2]byte
	_, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
		f.Fd(), syscall.TIOCGPGRP,
		uintptr(unsafe.Pointer(&t)))
	return errno == 0
}
