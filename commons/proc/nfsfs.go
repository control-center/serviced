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
	"bufio"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"

	"github.com/zenoss/glog"
)

var ErrMountPointNotFound = errors.New("mount point not found")

var procNFSFSServersFile = "servers"
var procNFSFSVolumesFile = "volumes"
var procFindmntCommand = "/bin/findmnt --raw --noheading -o MAJ:MIN,FSTYPE,SOURCE,TARGET,OPTIONS %s"

// NFSVolumeInfo is merged from mountinfo and volumes
type NFSMountInfo struct {
	// from findmnt
	MountInfo

	// from /proc/fs/nfsfs/volumes
	Version  string // nfsversion: v4, v3, ...
	ServerID string // id of server: 0a57d1a8
	Port     string // port on server: 801
	FSCache  string // whether fscache is used (yes/no)
}

// MountInfo is retrieved from mountinfo
type MountInfo struct {
	DeviceID   string // device id: 0:137
	FSType     string // filesystem type: btrfs, nfs4, ext4
	RemotePath string // path to the server: 10.87.209.168:/serviced_var
	LocalPath  string // path on the client: /tmp/serviced/var
	ServerIP   string // server ip address: 10.87.209.168
}

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

// GetMountInfo gets the mount info of the mountpoint
func GetMountInfo(mountpoint string) (*MountInfo, error) {
	command := []string{"bash", "-c", fmt.Sprintf(procFindmntCommand, mountpoint)}

	thecmd := exec.Command(command[0], command[1:]...)
	glog.V(1).Infof("command: %+v", command)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		glog.Warningf("could not find mountpoint:%s with command:%+v  output:%s (%s)", mountpoint, command, string(output), err)
		return nil, ErrMountPointNotFound
	}

	// [root@jrivera-tb1 ~]# /bin/findmnt --raw --noheading -o MAJ:MIN,FSTYPE,SOURCE,TARGET,OPTIONS /tmp/serviced/var
	// 0:329 nfs4 10.87.209.168:/serviced_var /tmp/serviced/var rw,relatime,vers=4.0,rsize=1048576,wsize=1048576,namlen=255,hard,proto=tcp,port=0,timeo=600,retrans=2,sec=sys,clientaddr=10.87.209.168,local_lock=none,addr=10.87.209.168
	line := strings.TrimSpace(string(output))

	glog.V(4).Infof("line: %s", line)

	fields := strings.Fields(line)
	switch len(fields) {
	case 5:
		break
	case 0:
		return nil, ErrMountPointNotFound
	default:
		glog.Infof("command: %+v", command)
		return nil, fmt.Errorf("expected 5 fields, got %d: %s", len(fields), line)
	}

	// parse options
	options := map[string]string{}
	optionParts := strings.Split(fields[4], ",")
	for _, option := range optionParts {
		pairs := strings.Split(option, "=")
		if len(pairs) == 2 {
			glog.V(4).Infof("option: %s  k:%s  v:%s", option, pairs[0], pairs[1])
			options[pairs[0]] = pairs[1]
		}
	}

	// return mount info
	info := MountInfo{
		DeviceID:   fields[0],
		FSType:     fields[1],
		RemotePath: fields[2],
		LocalPath:  fields[3],
		ServerIP:   options["addr"],
	}
	glog.V(4).Infof("mount info: %+v", info)
	return &info, nil
}

// GetNFSVolumeInfo gets the NFSMountInfo of the mountpoint
func GetNFSVolumeInfo(mountpoint string) (*NFSMountInfo, error) {
	minfo, err := GetMountInfo(mountpoint)
	if err != nil {
		return nil, err
	}

	if !strings.HasPrefix(minfo.FSType, "nfs") {
		return nil, fmt.Errorf("%s is not nfs; uses %s", minfo.LocalPath, minfo.FSType)
	}

	info := NFSMountInfo{
		MountInfo: *minfo,
	}

	return &info, nil
}
