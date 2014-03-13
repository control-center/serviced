// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// BlkioIoServiceBytes stores data from /sys/fs/cgroup/blkio/blkio.io_service_bytes.
type BlkioIoServiceBytes struct {
	Total int64
}

// BlkioIoServiceBytesFileName can be altered to use a different source for BlkioIoServiceBytes.
var BlkioIoServiceBytesFileName = "/sys/fs/cgroup/blkio/blkio.io_service_bytes"

// ReadBlkioIoServiceBytes fills out and returns a BlkioIoServiceBytes struct from BlkioIoServiceBytesFileName.
func ReadBlkioIoServiceBytes() BlkioIoServiceBytes {
	stat := BlkioIoServiceBytes{}
	kv, _ := parseSSKVint64(BlkioIoServiceBytesFileName)
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
