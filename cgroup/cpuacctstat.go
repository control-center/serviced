// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

// CpuacctStat stores data from /sys/fs/cgroup/cpuacct/cpuacct.stat.
type CpuacctStat struct {
	User   int64
	System int64
}

// CpuacctStatFileName can be altered to use a different source for CpuacctStat.
var CpuacctStatFileName = "/sys/fs/cgroup/cpuacct/cpuacct.stat"

// ReadCpuacctStat fills out and returns a CpuacctStat struct from CpuacctStatFileName.
func ReadCpuacctStat() CpuacctStat {
	stat := CpuacctStat{}
	kv, _ := parseSSKVint64(CpuacctStatFileName)
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
