// Copyright 2016 The Serviced Authors.
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

package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

const DefaultProcMount = "/proc/mounts"

var (
	ErrParseMount = errors.New("could not parse mount data")
)

type ExecError struct {
	Command []string
	Output  []byte
	Err     error
}

func (err ExecError) Error() string {
	return fmt.Sprintf("error running command `%s` (%s): %s", strings.Join(err.Command, " "), string(err.Output), err.Err)
}

// MountInfo is the object describing the mount point
type MountInfo struct {
	Device     string
	MountPoint string
	FSType     string
	Options    []string
	Dump       int
	Fsck       int
}

func (mount MountInfo) String() string {
	return fmt.Sprintf("%s %s %s %s %d %d", mount.Device, mount.MountPoint, mount.FSType, strings.Join(mount.Options, ","), mount.Dump, mount.Fsck)
}

// MountProc manages mount info
type MountProc interface {
	ListAll() ([]MountInfo, error)
	IsMounted(string) (bool, error)
	Unmount(string) error
}

// GetDefaultMountProc returns the default mount process manager
func GetDefaultMountProc() MountProc {
	return &LinuxMount{DefaultProcMount}
}

// LinuxMount loads mount data from a file
type LinuxMount struct {
	ProcMount string
}

// ListAll returns info on all mount points
func (m *LinuxMount) ListAll() (mounts []MountInfo, err error) {
	fh, err := os.Open(m.ProcMount)
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	reader := bufio.NewReader(fh)
	for {
		line, err := reader.ReadString('\n')
		if line = strings.TrimSpace(line); line != "" {
			var fields []string
			if fields = strings.Fields(line); len(fields) != 6 {
				return nil, ErrParseMount
			}
			mount := MountInfo{}
			mount.Device = fields[0]
			mount.MountPoint = fields[1]
			mount.FSType = fields[2]
			mount.Options = strings.Split(fields[3], ",")
			if mount.Dump, err = strconv.Atoi(fields[4]); err != nil {
				return nil, ErrParseMount
			}
			if mount.Fsck, err = strconv.Atoi(fields[5]); err != nil {
				return nil, ErrParseMount
			}
			mounts = append(mounts, mount)
		}
		if err == io.EOF {
			return mounts, nil
		} else if err != nil {
			return nil, err
		}
	}
}

// IsMounted returns true if path is a mountpoint or device
func (m *LinuxMount) IsMounted(path string) (bool, error) {
	mounts, err := m.ListAll()
	if err != nil {
		return false, err
	}
	for _, mount := range mounts {
		if mount.Device == path || mount.MountPoint == path {
			return true, nil
		}
	}
	return false, nil
}

// Unmount unmounts a device or mountpoint
func (m *LinuxMount) Unmount(path string) error {
	if mounted, err := m.IsMounted(path); err != nil {
		return err
	} else if mounted {
		cmd := exec.Command("umount", "-f", path)
		if out, err := cmd.CombinedOutput(); err != nil {
			return ExecError{
				Command: cmd.Args,
				Output:  out,
				Err:     err,
			}
		}
	}
	return nil
}
