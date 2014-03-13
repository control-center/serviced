// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

import "fmt"

// CpuacctStat stores data from /sys/fs/cgroup/cpuacct/cpuacct.stat.
type CpuacctStat struct {
	User   int64
	System int64
}

// ReadCpuacctStat fills out and returns a CpuacctStat struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/cpuacct/cpuacct.stat is used.
func ReadCpuacctStat(fileName string) (*CpuacctStat, error) {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/cpuacct/cpuacct.stat"
	}
	stat := CpuacctStat{}
	kv, err := parseSSKVint64(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing /sys/fs/cgroup/cpuacct/cpuacct.stat")
	}
	for k, v := range kv {
		switch k {
		case "user":
			stat.User = v
		case "system":
			stat.System = v
		}
	}
	return &stat, nil
}
