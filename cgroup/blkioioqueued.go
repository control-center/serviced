package cgroup

type BlkioIoQueued struct {
	Total int64
}

func ReadBlkioIoQueued() BlkioIoQueued {
	stat := BlkioIoQueued{}
	kv, _ := parseSSKVint64("/sys/fs/cgroup/blkio/blkio.io_queued")
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
