// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
