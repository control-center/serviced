// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// BlkioIoServiceBytes stores data from /sys/fs/cgroup/blkio/blkio.io_service_bytes.
type BlkioIoServiceBytes struct {
	Total int64
}

// ReadBlkioIoServiceBytes fills out and returns a BlkioIoServiceBytes struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/blkio/blkio.io_service_bytes is used.
func ReadBlkioIoServiceBytes(fileName string) BlkioIoServiceBytes {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/blkio/blkio.io_service_bytes"
	}
	stat := BlkioIoServiceBytes{}
	kv, _ := parseSSKVint64(fileName)
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
