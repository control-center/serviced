package dao

import "fmt"
import "os"

var urandomFilename = "/dev/urandom"

// Generate a new UUID
func newUuid() (string, error) {
	f, err := os.Open(urandomFilename)
	if err != nil {
		return "", err
	}
	b := make([]byte, 16)
	defer f.Close()
	f.Read(b)
	uuid := fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:])
	return uuid, err
}

