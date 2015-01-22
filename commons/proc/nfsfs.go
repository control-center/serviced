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
	"strings"
)

var procNFSFSServersFile = "servers"
var procNFSFSVolumesFile = "volumes"

// ProcNFSFSServer is a parsed representation of /proc/fs/nfsfs/servers information.
type ProcNFSFSServer struct {
	Version  string // nfsversion: v4, v3, ...
	ServerID string // id of server: 0a57d1a8
	Port     string // port on server: 801
	Hostname string // hostname of server: 10.87.209.168
}

// ProcNFSFSVolume is a parsed representation of /proc/fs/nfsfs/volumes information.
type ProcNFSFSVolume struct {
	Version  string // nfsversion: v4, v3, ...
	ServerID string // id of server: 0a57d1a8
	Port     string // port on server: 801
	DeviceID string // device id: 0:137
	FSID     string // filesystem id: 45a148e989326106
	FSCache  string // whether fscache is used (yes/no)
}

// GetProcNFSFSServers gets a map to the /proc/fs/nfsfs/servers
func GetProcNFSFSServers() (map[string]ProcNFSFSServer, error) {
	// read in the file
	data, err := ioutil.ReadFile(fmt.Sprintf(procDir+"fs/nfsfs/%s", procNFSFSServersFile))
	if err != nil {
		return nil, err
	}

	servers := make(map[string]ProcNFSFSServer)
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	linenum := 0
	for scanner.Scan() {
		line := scanner.Text()

		linenum++
		glog.V(4).Infof("%d: %s", linenum, line)
		if linenum < 2 {
			continue
		} else if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		switch len(fields) {
		case 5:
			break
		case 0:
			continue
		default:
			return nil, fmt.Errorf("expected 5 fields, got %d: %s", len(fields), line)
		}
		svr := ProcNFSFSServer{
			Version:  fields[0],
			ServerID: fields[1],
			Port:     fields[2],
			Hostname: fields[4],
		}
		key := fmt.Sprintf("%s:%s:%s", svr.Version, svr.ServerID, svr.Port)
		servers[key] = svr
	}
	glog.V(4).Infof("nfsfs servers: %+v", servers)
	return servers, nil
}

// GetProcNFSFSVolumes gets a map to the /proc/fs/nfsfs/volumes
func GetProcNFSFSVolumes() ([]ProcNFSFSVolume, error) {
	// read in the file
	data, err := ioutil.ReadFile(fmt.Sprintf(procDir+"fs/nfsfs/%s", procNFSFSVolumesFile))
	if err != nil {
		return nil, err
	}

	var volumes []ProcNFSFSVolume
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	linenum := 0
	for scanner.Scan() {
		line := scanner.Text()

		linenum++
		glog.V(4).Infof("%d: %s", linenum, line)
		if linenum < 2 {
			continue
		} else if strings.HasPrefix(line, "#") {
			continue
		}

		fields := strings.Fields(line)
		switch len(fields) {
		case 6:
			break
		case 0:
			continue
		default:
			return nil, fmt.Errorf("expected 5 fields, got %d: %s", len(fields), line)
		}
		svr := ProcNFSFSVolume{
			Version:  fields[0],
			ServerID: fields[1],
			Port:     fields[2],
			DeviceID: fields[3],
			FSID:     fields[4],
			FSCache:  fields[5],
		}
		volumes = append(volumes, svr)
	}
	glog.V(4).Infof("nfsfs volumes: %+v", volumes)
	return volumes, nil
}
