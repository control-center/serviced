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

package proc

import (
	"github.com/zenoss/glog"

	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"reflect"
	"strings"
	"time"
)

var procNFSDExportsFile = "fs/nfsd/exports"

func GetProcNFSDExportsFilePath() string {
	return fmt.Sprintf("%s%s", procDir, procNFSDExportsFile)
}

var ErrMountPointNotExported = fmt.Errorf("mount point not exported")

// ProcNFSDExports is a parsed representation of /proc/fs/nfsd/exports information.
type ProcNFSDExports struct {
	MountPoint    string                       // exported path on the server: /exports/serviced_var
	ClientOptions map[string]NFSDExportOptions // keys are clients: 'Machine Name Formats' of exports manpage
}

// Equals checks the equality of two ProcNFSDExports
func (a *ProcNFSDExports) Equals(b *ProcNFSDExports) bool {
	if a.MountPoint != b.MountPoint {
		return false
	}
	if !reflect.DeepEqual(a.ClientOptions, b.ClientOptions) {
		return false
	}
	return true
}

// NFSDExportOptions are options specified in 'General Options' of exports manpage
type NFSDExportOptions map[string]string

// GetProcNFSDExport gets the export info to the mountpoint entry in /proc/fs/nfsd/exports
func GetProcNFSDExport(mountpoint string) (*ProcNFSDExports, error) {
	exports, err := GetProcNFSDExports()
	if err != nil {
		return nil, err
	}

	export, ok := exports[mountpoint]
	if !ok {
		return nil, ErrMountPointNotExported
	}

	return &export, nil
}

// GetProcNFSDExports gets a map to the /proc/fs/nfsd/exports
func GetProcNFSDExports() (map[string]ProcNFSDExports, error) {
	// read in the file
	data, err := ioutil.ReadFile(GetProcNFSDExportsFilePath())
	if err != nil {
		return nil, err
	}

	exports := make(map[string]ProcNFSDExports)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	linenum := 0
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		linenum++
		glog.V(4).Infof("%d: %s", linenum, line)
		if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		switch len(fields) {
		case 0:
			continue
		case 1:
			glog.Errorf("expected at least 2 fields, got %d: %s", len(fields), line)
			continue
		}

		svr := ProcNFSDExports{
			MountPoint:    fields[0],
			ClientOptions: make(map[string]NFSDExportOptions),
		}
		for _, clientSpec := range fields[1:] {
			parts := strings.Split(clientSpec, "(")
			switch len(parts) {
			case 0:
				continue
			case 1:
				glog.Errorf("expected at least 2 parts, got %d: %s", len(parts), clientSpec)
				continue
			}

			svr.ClientOptions[parts[0]] = parseOptions(strings.TrimSuffix(parts[1], ")"))
		}

		exports[svr.MountPoint] = svr
	}

	glog.V(4).Infof("nfsd exports: %+v", exports)
	return exports, nil
}

// parseOptions generates a map from a line specifying options
func parseOptions(line string) map[string]string {
	options := map[string]string{}
	optionParts := strings.Split(line, ",")
	for _, option := range optionParts {
		pairs := strings.SplitN(option, "=", 2)
		switch len(pairs) {
		case 1:
			glog.V(4).Infof("option: %s", pairs[0])
			options[pairs[0]] = ""
			continue
		case 2:
			glog.V(4).Infof("option: %s  k:%s  v:%s", option, pairs[0], pairs[1])
			options[pairs[0]] = pairs[1]
			continue
		}
	}

	return options
}

// MonitorExportedVolume monitors the exported volume and logs on failure
func MonitorExportedVolume(mountpoint string, monitorInterval time.Duration, shutdown <-chan interface{}) {
	glog.Infof("monitoring exported volume %s at polling interval: %s", mountpoint, monitorInterval)

	var modtime time.Time
	for {
		glog.V(0).Infof("determining export status for DFS NFS volume %s", mountpoint)

		changed, err := hasFileChanged(GetProcNFSDExportsFilePath(), &modtime)
		if err != nil {
			glog.Warningf("unable to determine whether mountpoint %s is exported: %s", mountpoint, err)
		} else if changed {
			glog.V(0).Infof("exported info has changed for mountpoint %s", mountpoint)
			mountinfo, err := GetProcNFSDExport(mountpoint)
			if err != nil {
				if err == ErrMountPointNotExported {
					glog.Warningf("volume %s is not exported - further action may be required", mountpoint)

					// TODO: take action: possibly reload nfs, restart nfs
				} else {
					glog.Warningf("unable to retrieve volume export info for %s: %s", mountpoint, err)
				}
			} else {
				glog.V(0).Infof("DFS NFS volume %s (uuid:%+v) is exported", mountpoint, mountinfo.ClientOptions["*"]["uuid"])
			}
		}

		select {
		case <-time.After(monitorInterval):

		case <-shutdown:
			glog.V(0).Infof("no longer monitoring export status for DFS NFS volume %s", mountpoint)
			return
		}
	}
}

// hasFileChanged determines whether file has been modified and updates modtime
func hasFileChanged(filename string, modtime *time.Time) (bool, error) {
	fileinfo, err := os.Stat(filename)
	if err != nil {
		glog.V(4).Infof("unable to stat file: %s %s", filename, err)
		return false, err
	}

	prevtime := *modtime
	*modtime = fileinfo.ModTime()
	if fileinfo.ModTime().After(prevtime) {
		duration := fileinfo.ModTime().Sub(prevtime)
		if prevtime.IsZero() {
			glog.V(4).Infof("first time file %s is checked", filename)
			return true, nil
		}
		glog.V(4).Infof("file %s has been modified since %s", filename, duration)
		return true, nil
	}

	glog.V(4).Infof("file %s has not been modified", filename)
	return false, nil
}
