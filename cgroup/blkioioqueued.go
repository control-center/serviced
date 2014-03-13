// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// BlkioIoQueued stores data from /sys/fs/cgroup/blkio/blkio.io_queued.
type BlkioIoQueued struct {
	Total int64
}

// BlkioIoQueuedFileName can be altered to use a different source for BlkioIoQueued.
var BlkioIoQueuedFileName = "/sys/fs/cgroup/blkio/blkio.io_queued"

// ReadBlkioIoQueued fills out and returns a BlkioIoQueued struct from BlkioIoQueuedFileName.
func ReadBlkioIoQueued() BlkioIoQueued {
	stat := BlkioIoQueued{}
	kv, _ := parseSSKVint64(BlkioIoQueuedFileName)
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
