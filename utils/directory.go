// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package utils

import (
	"os"
	"path"
	"runtime"
	"strings"
)

//ServiceDHome gets the home location of serviced by looking at the enviornment
func ServiceDHome() string {
	return os.Getenv("SERVICED_HOME")
}

//LocalDir gets the absolute path to a particular directory under ServiceDHome
// if SERVICED_HOME is not defined then we use the location of the caller
func LocalDir(p string) string {
	homeDir := ServiceDHome()
	if len(homeDir) == 0 {
		_, filename, _, _ := runtime.Caller(1)
		homeDir = strings.Replace(path.Dir(filename), "utils", "", 1)
	}
	return path.Join(homeDir, p)
}

// ResourcesDir points to internal services resources directory
func ResourcesDir() string {
	homeDir := ServiceDHome()
	if len(homeDir) > 0 {
		return path.Join(homeDir, "resources")
	}
	return LocalDir("isvcs/resources")
}
