// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

import "fmt"

// BlkioIoServiced stores data from /sys/fs/cgroup/blkio/blkio.io_serviced.
type BlkioIoServiced struct {
	Total int64
}

// ReadBlkioIoServiced fills out and returns a BlkioIoServiced struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/blkio/blkio.io_serviced is used.
func ReadBlkioIoServiced(fileName string) (*BlkioIoServiced, error) {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/blkio/blkio.io_serviced"
	}
	stat := BlkioIoServiced{}
	kv, err := parseSSKVint64(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing /sys/fs/cgroup/blkio/blkio.io_serviced")
	}
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return &stat, nil
}
