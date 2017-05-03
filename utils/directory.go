// Copyright 2014 The Serviced Authors.
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

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package utils

import (
	"fmt"
	"os"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"github.com/control-center/serviced/config"
	"github.com/zenoss/glog"
	"runtime"
)

//ServiceDHome gets the home location of serviced by looking at the environment
func ServiceDHome() string {
	homeDir := config.GetOptions().HomePath
	if len(homeDir) == 0 {
		// This fallback is used in unit-tests, but in actual practice,
		// we should not hit this case, because there is a default
		// value defined via code in cli/api/options.go.  But just in
		// case that somehow get's undone in the future, the log message
		// will let us know that we used a fallback.
		_, filename, _, _ := runtime.Caller(1)
		homeDir = strings.Replace(path.Dir(filename), "utils", "", 1)
		plog.Warnf("SERVICED_HOME not set; defaulting to %s", homeDir)
	}
	return homeDir
}

//LocalDir gets the absolute path to a particular directory under ServiceDHome
// if SERVICED_HOME is not defined then we use the location of the caller
func LocalDir(p string) string {
	homeDir := ServiceDHome()
	return path.Join(homeDir, p)
}

// ResourcesDir points to internal services resources directory
func ResourcesDir() string {
	return LocalDir("isvcs/resources")
}

// BackupDir gets the directory where backup files are stored
func BackupDir(basepath string) string {
	if backupDir := strings.TrimSpace(basepath); backupDir == "" {
		return TempDir("backups")
	} else {
		return filepath.Join(filepath.Clean(backupDir), "backups")
	}
}

// TempDir gets the temp serviced directory
func TempDir(p string) string {
	var tmp string

	if user, err := user.Current(); err == nil {
		tmp = path.Join(os.TempDir(), fmt.Sprintf("serviced-%s", user.Username), p)
	} else {
		tmp = path.Join(os.TempDir(), fmt.Sprintf("serviced"), p)
		glog.Warningf("Defaulting home to %s", tmp)
	}

	return tmp
}

// ServicedLogDir gets the serviced log directory
func ServicedLogDir() string {
        if config.GetOptions().LogPath != "" {
		return config.GetOptions().LogPath
	} else{
		return os.Getenv("SERVICED_LOG_PATH")
	}
}
