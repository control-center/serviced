/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

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
	return LocalDir("isvcs/resources")
}
