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

// Package stats collects serviced metrics and posts them to the TSDB.

package stats

import (
	"path/filepath"
)

// GetOpenFileDescriptorCount returns the number of open file descriptors for the process id of the caller.
func GetOpenFileDescriptorCount() (int64, error) {
	files, err := filepath.Glob("/proc/self/fd/*")
	if err != nil {
		return 0, err
	}
	return int64(len(files)), nil
}
