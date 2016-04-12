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

package volume

import (
	"errors"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"sort"
	"strings"
	"syscall"

	"github.com/zenoss/glog"
)

var (
	ErrNotADirectory = errors.New("not a directory")
	ErrBtrfsCommand  = errors.New("error running btrfs command")
)

const FlagFileName = ".initialized"

// IsDir() checks if the given dir is a directory. If any error is encoutered
// it is returned and directory is set to false.
func IsDir(dirName string) (dir bool, err error) {
	if lstat, err := os.Lstat(dirName); err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	} else {
		if !lstat.IsDir() {
			return false, ErrNotADirectory
		}
	}
	return true, nil
}

// FileInfoSlice is a os.FileInfo array sortable by modification time
type FileInfoSlice []os.FileInfo

func (p FileInfoSlice) Len() int {
	return len(p)
}

func (p FileInfoSlice) Less(i, j int) bool {
	return p[i].ModTime().Before(p[j].ModTime())
}

func (p FileInfoSlice) Swap(i, j int) {
	p[i], p[j] = p[j], p[i]
}

// Labels will return the names of the files in the slice, sorted by modification time
func (p FileInfoSlice) Labels() []string {
	// This would probably be very slightly more efficient with a heap, but the
	// API would be more complicated
	sort.Sort(p)
	labels := make([]string, p.Len())
	for i, label := range p {
		labels[i] = label.Name()
	}
	return labels
}

func IsRoot() bool {
	user, err := user.Current()
	if err != nil {
		glog.Errorf("Unable to determine current user: %s", err)
		return false
	}
	return user.Uid == "0"
}

func IsSudoer() bool {
	if !IsRoot() {
		err := exec.Command("sudo", "-n", "true").Run()
		return err == nil
	}
	return false
}

// RunBtrFSCmd runs a btrfs command, optionally using sudo
func RunBtrFSCmd(sudoer bool, args ...string) ([]byte, error) {
	cmd := append([]string{"btrfs"}, args...)
	if sudoer {
		cmd = append([]string{"sudo", "-n"}, cmd...)
	}
	glog.V(4).Infof("Executing: %v", cmd)
	output, err := exec.Command(cmd[0], cmd[1:]...).CombinedOutput()
	if err != nil {
		glog.V(1).Infof("unable to run cmd:%s  output:%s  error:%s", cmd, string(output), err)
		return output, ErrBtrfsCommand
	}
	return output, err
}

// IsBtrfsFilesystem determines whether the path is a btrfs filesystem
func IsBtrfsFilesystem(path string) bool {
	_, err := RunBtrFSCmd(false, "filesystem", "df", path)
	return err == nil
}

func FlagFilePath(root string) string {
	return filepath.Join(root, FlagFileName)
}

func TouchFlagFile(root string) error {
	// Touch the file indicating that this dir has been initialized
	initfile := FlagFilePath(root)
	if _, err := os.Stat(initfile); os.IsNotExist(err) {
		return ioutil.WriteFile(initfile, []byte{}, 0754)
	}
	return nil
}

func FilesystemBytesSize(path string) int64 {
	s := syscall.Statfs_t{}
	syscall.Statfs(path, &s)
	return int64(s.Bsize) * int64(s.Blocks)
}

func DefaultSnapshotLabel(tenant, label string) string {
	prefix := tenant + "_"
	if !strings.HasPrefix(label, prefix) {
		label = prefix + label
	}
	return label
}
