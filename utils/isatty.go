package utils

import (
	"os"
	"syscall"
	"unsafe"
)

// Isatty returns true if f is a TTY, false otherwise.
func Isatty(f *os.File) bool {
	var t syscall.Termios
	_, _, errno := syscall.Syscall6(syscall.SYS_IOCTL,
		f.Fd(), ioctlTermioFlag,
		uintptr(unsafe.Pointer(&t)), 0, 0, 0)
	return errno == 0
}
