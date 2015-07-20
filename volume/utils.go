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
	"fmt"
	"os"
	"sort"
)

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
			return false, fmt.Errorf("%s is not a directory", dirName)
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
