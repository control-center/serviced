// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
	"time"

	"github.com/zenoss/glog"
)

func commandAsRoot(name string, arg ...string) (*exec.Cmd, error) {
	user, e := user.Current()
	if e != nil {
		return nil, e
	}
	if user.Uid == "0" {
		return exec.Command(name, arg...), nil
	}
	cmd := exec.Command("sudo", "-n", "echo")
	if output, err := cmd.CombinedOutput(); err != nil {
		glog.Errorf("Unable to run as root cmd:%+v error:%v output:%s", cmd, err, string(output))
		return nil, err
	}
	return exec.Command("sudo", append([]string{"-n", name}, arg...)...), nil //Go, you make me sad.
}

type tarinfo struct {
	Permission string // TODO: May want to change this later when we care
	Owner      string
	Group      string
	Size       int64
	Timestamp  time.Time
	Filename   string
}

type tarfile []tarinfo

func readTarFile(filename string) (*tarfile, error) {
	cmd, err := commandAsRoot("tar", "-tzvf", filename)
	if err != nil {
		return nil, err
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		err = fmt.Errorf("%s: %s", string(output), err)
		glog.Errorf("Could not load contents of %s: %s", filename, err)
		return nil, err
	}

	return new(tarfile).init(output)
}

func (t *tarfile) init(output []byte) (*tarfile, error) {
	rawData := strings.TrimSpace(string(output))
	rows, err := strings.Split(rawData, "\n"), fmt.Errorf("bad format")
	contents := make([]tarinfo, len(rows))
	for i, row := range rows {
		fields := strings.Fields(row)
		if len(fields) != 6 {
			return nil, fmt.Errorf("bad format")
		}
		var data tarinfo
		data.Permission = fields[0]
		owngrp := strings.SplitN(fields[1], "/", 2)
		if len(owngrp) != 2 {
			return nil, err
		}
		data.Owner = owngrp[0]
		data.Group = owngrp[1]
		data.Size, err = strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, err
		}
		data.Timestamp, err = time.Parse("2006-01-02 15:04", fields[3]+" "+fields[4])
		if err != nil {
			return nil, err
		}
		data.Filename = fields[5]
		contents[i] = data
	}
	*t = tarfile(contents)
	return t, nil
}

func (t *tarfile) ExpandedSize(blocksize int64) (int64, error) {
	if blocksize <= 0 {
		return 0, fmt.Errorf("blocksize <= 0")
	}

	var total int64
	for _, data := range []tarinfo(*t) {
		total += data.Size
	}
	return total / blocksize, nil
}

type diskinfo struct {
	FileSystem string
	Blocks     int64
	Used       int64
	Available  int64
	UsePercent int
	MountedOn  string
}

func parseDisks(output []byte) ([]diskinfo, error) {
	rawData := strings.TrimSpace(string(output))
	rows, err := strings.Split(rawData, "\n"), fmt.Errorf("bad format")
	if len(rows) < 2 {
		return nil, err
	}
	disks := make([]diskinfo, len(rows)-1)
	for i, row := range rows[1:] {
		fields := strings.Fields(row)
		if len(fields) != 6 {
			return nil, fmt.Errorf("bad format")
		}
		var disk diskinfo
		disk.FileSystem = fields[0]
		disk.Blocks, err = strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			return nil, err
		}
		disk.Used, err = strconv.ParseInt(fields[2], 10, 64)
		if err != nil {
			return nil, err
		}
		disk.Available, err = strconv.ParseInt(fields[3], 10, 64)
		if err != nil {
			return nil, err
		}
		usePercent, err := strconv.ParseInt(strings.TrimRight(fields[4], "%"), 10, 32)
		disk.UsePercent = int(usePercent)
		if err != nil {
			return nil, err
		}
		disk.MountedOn = fields[5]
		disks[i] = disk
	}
	return disks, nil
}

func checkDisk(dest string, blockSize int64) (*diskinfo, error) {
	if blockSize <= 0 {
		return nil, fmt.Errorf("blockSize <= 0")
	}
	if cmd, err := commandAsRoot("df", "--block-size", strconv.FormatInt(blockSize, 10), dest); err != nil {
		return nil, err
	} else if output, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("%s: %s", string(output), err)
		glog.Errorf("Could not read available disk %+v: %s", cmd, err)
		return nil, err
	} else if disks, err := parseDisks(output); err != nil {
		glog.Errorf("Could not parse output %s: %s", string(output), err)
		return nil, err
	} else if len(disks) != 1 {
		glog.Errorf("Unexpected result %s: %+v", string(output), disks)
		return nil, fmt.Errorf("unexpected error")
	} else {
		return &disks[0], nil
	}
}

func mkdir(path string) error {
	return os.MkdirAll(path, os.ModeDir|0755)
}

func ls(path string) ([]string, error) {
	files, err := ioutil.ReadDir(path)
	if err != nil {
		return nil, err
	}

	result := make([]string, len(files))
	for i, file := range files {
		result[i] = file.Name()
	}

	return result, nil
}

func exportTGZ(src, filename string) error {
	//FIXME: Tar file should put all contents below a sub-directory (rather than directly in current directory).
	cmd, e := commandAsRoot("tar", "-czf", filename, "-C", src, ".")
	if e != nil {
		return e
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		glog.Errorf("Unable to writeDirectoryToTgz cmd:%+v error:%v output:%s", cmd, err, string(output))
		return err
	}
	return nil
}

func importTGZ(dest, filename string) error {
	if _, err := os.Stat(dest); err != nil {
		if !os.IsNotExist(err) {
			glog.Errorf("Could not stat %s: %s", dest, err)
			return err
		}
	}
	if err := mkdir(dest); err != nil {
		glog.Errorf("Could not create %s: %s", dest, err)
		return err
	}

	// verify there is enough space to expand the tgz
	tf, err := readTarFile(filename)
	if err != nil {
		glog.Errorf("Could not read tar file %s: %s", filename, err)
		return err
	}

	expandedSize, err := tf.ExpandedSize(1024)
	if err != nil {
		glog.Errorf("Could not compute expanded size of tar file %s: %s", filename, err)
		return err
	}

	disk, err := checkDisk(dest, 1024)
	if err != nil {
		glog.Errorf("Could not acquire disk information for %s: %s", dest, err)
		return err
	}

	if disk.Available < 2*expandedSize {
		glog.Errorf("Not enough space on disk to restore from backup (uncompressed: %dK) (available: %dK)", expandedSize, disk.Available)
		return fmt.Errorf("insufficient disk space")
	}

	cmd, err := commandAsRoot("tar", "-xpUf", filename, "-C", dest, "--numeric-owner")
	if err != nil {
		return err
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		err = fmt.Errorf("%s: %s", string(output), err)
		glog.Errorf("Could not import tgz [%v]: %s", err)
		return err
	}
	return nil
}
