// +build linux

package devicemapper

import (
	"fmt"
	"os"
	"syscall"
)

func major(device uint64) uint64 {
	return (device >> 8) & 0xfff
}

func minor(device uint64) uint64 {
	return (device & 0xff) | ((device >> 12) & 0xfff00)
}

// Calculate the same device prefix as Docker, using mostly their code
func GetDevicePrefix(root string) (string, error) {
	st, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("Error looking up dir %s: %s", root, err)
	}
	sysSt := st.Sys().(*syscall.Stat_t)
	// "reg-" stands for "regular file".
	// In the future we might use "dev-" for "device file", etc.
	// docker-maj,min[-inode] stands for:
	//	- Managed by docker
	//	- The target of this device is at major <maj> and minor <min>
	//	- If <inode> is defined, use that file inside the device as a loopback image. Otherwise use the device itself.
	return fmt.Sprintf("docker-%d:%d-%d", major(sysSt.Dev), minor(sysSt.Dev), sysSt.Ino), nil
}
