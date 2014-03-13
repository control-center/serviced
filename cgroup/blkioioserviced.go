// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// BlkioIoServiced stores data from /sys/fs/cgroup/blkio/blkio.io_serviced.
type BlkioIoServiced struct {
	Total int64
}

// BlkioIoServicedFileName can be altered to use a different source for BlkioIoServiced.
var BlkioIoServicedFileName = "/sys/fs/cgroup/blkio/blkio.io_serviced"

// ReadBlkioIoServiced fills out and returns a BlkioIoServiced struct from BlkioIoServicedFileName.
func ReadBlkioIoServiced() BlkioIoServiced {
	stat := BlkioIoServiced{}
	kv, _ := parseSSKVint64(BlkioIoServicedFileName)
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
