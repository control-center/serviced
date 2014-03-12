package cgroup

type CpuacctStat struct {
	User   int64
	System int64
}

func ReadCpuacctStat() CpuacctStat {
	stat := CpuacctStat{}
	kv, _ := parseSSKVint64("/sys/fs/cgroup/cpuacct/cpuacct.stat")
	for k, v := range kv {
		switch k {
		case "user":
			stat.User = v
		case "system":
			stat.System = v
		}
	}
	return stat
}
