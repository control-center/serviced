package cgroup

type BlkioIoServiceBytes struct {
	Total int64
}

func ReadBlkioIoServiceBytes() BlkioIoServiceBytes {
	stat := BlkioIoServiceBytes{}
	kv, _ := parseSSKVint64("/sys/fs/cgroup/blkio/blkio.io_service_bytes")
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
