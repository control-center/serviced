package cgroup

type BlkioIoServiced struct {
	Total int64
}

func ReadBlkioIoServiced() BlkioIoServiced {
	stat := BlkioIoServiced{}
	kv, _ := parseSSKVint64("/sys/fs/cgroup/blkio/blkio.io_serviced")
	for k, v := range kv {
		switch k {
		case "Total":
			stat.Total = v
		}
	}
	return stat
}
