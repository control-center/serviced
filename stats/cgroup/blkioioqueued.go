// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

import "fmt"

// BlkioIoQueued stores data from /sys/fs/cgroup/blkio/blkio.io_queued.
type BlkioIoQueued struct {
	Total int64
}

// ReadBlkioIoQueued fills out and returns a BlkioIoQueued struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/blkio/blkio.io_queued is used.
func ReadBlkioIoQueued(fileName string) (*BlkioIoQueued, error) {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/blkio/blkio.io_queued"
	}
	stat := BlkioIoQueued{}
	kv, err := parseSSKVint64(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing /sys/fs/cgroup/blkio/blkio.io_queued")
	}
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return &stat, nil
}
