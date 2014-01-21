package isvcs

import (
	"github.com/zenoss/serviced"

	"path"
	"runtime"
)

func localDir(p string) string {
	homeDir := serviced.ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = path.Dir(filename)
	}
	return path.Join(homeDir, p)
}

func imagesDir() string {
	return localDir("images")
}

func resourcesDir() string {
	return localDir("resources")
}
