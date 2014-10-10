// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

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
		return nil, fmt.Errorf("error parsing %f", fileName)
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
