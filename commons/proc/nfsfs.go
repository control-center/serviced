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
	"errors"

	"github.com/zenoss/glog"

	"bufio"
	"fmt"
	"io/ioutil"
	"os/exec"
	"strings"
)

var ErrMountPointNotFound = errors.New("mount point not found")

var procNFSFSServersFile = "servers"
var procNFSFSVolumesFile = "volumes"
var procFindmntCommand = "/bin/findmnt --raw --noheading -o MAJ:MIN,FSTYPE,SOURCE,TARGET,OPTIONS %s | awk '{n=split($NF,fields,\"=\"); print $1, $2, $3, $4, fields[n]}'"

// NFSVolumeInfo is merged from mountinfo and volumes
type NFSMountInfo struct {
	// from findmnt
	MountInfo

	// from /proc/fs/nfsfs/volumes
	Version  string // nfsversion: v4, v3, ...
	ServerID string // id of server: 0a57d1a8
	Port     string // port on server: 801
	FSID     string // filesystem id: 45a148e989326106
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
			return nil, fmt.Errorf("expected 6 fields, got %d: %s", len(fields), line)
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

// GetProcNFSFSVolume gets the ProcNFSFSVolume of a deviceid from /proc/fs/nfsfs/volumes
func GetProcNFSFSVolume(deviceid string) (*ProcNFSFSVolume, error) {
	volumes, err := GetProcNFSFSVolumes()
	if err != nil {
		return nil, err
	}

	for idx := range volumes {
		glog.V(4).Infof("volume: %+v", volumes[idx])
		if deviceid == volumes[idx].DeviceID {
			return &volumes[idx], nil
		}
	}

	return nil, fmt.Errorf("unable to find volume for deviceid %s", deviceid)
}

// GetMountInfo gets the mount info of the mountpoint
func GetMountInfo(mountpoint string) (*MountInfo, error) {
	command := []string{"bash", "-c", fmt.Sprintf(procFindmntCommand, mountpoint)}

	thecmd := exec.Command(command[0], command[1:]...)
	glog.V(1).Infof("command: %+v", command)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		return nil, err
	}

	// [root@jrivera-tb1 ~]# /bin/findmnt --noheading -o MAJ:MIN,SOURCE,TARGET,OPTIONS /tmp/serviced/var
	//   0:137 10.87.209.168:/serviced_var /tmp/serviced/var rw,relatime,vers=4.0,rsize=1048576,wsize=1048576,namlen=255,hard,proto=tcp,port=0,timeo=600,retrans=2,sec=sys,clientaddr=10.87.209.168,local_lock=none,addr=10.87.209.168
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
	info := MountInfo{
		DeviceID:   fields[0],
		FSType:     fields[1],
		RemotePath: fields[2],
		LocalPath:  fields[3],
		ServerIP:   fields[4],
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

	volume, err := GetProcNFSFSVolume(minfo.DeviceID)
	if err != nil {
		return nil, err
	}

	info := NFSMountInfo{
		MountInfo: *minfo,

		Version:  volume.Version,
		ServerID: volume.ServerID,
		Port:     volume.Port,
		FSID:     volume.FSID,
		FSCache:  volume.FSCache,
	}

	return &info, nil
}
