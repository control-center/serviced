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

// MemoryStat stores data from /sys/fs/cgroup/memory/memory.stat.
type MemoryStat struct {
	Cache                   int64
	Rss                     int64
	RssHuge                 int64
	MappedFile              int64
	Pgpgin                  int64
	Pgpgout                 int64
	Pgfault                 int64
	Pgmajfault              int64
	InactiveAnon            int64
	ActiveAnon              int64
	InactiveFile            int64
	ActiveFile              int64
	Unevictable             int64
	HierarchicalMemoryLimit int64
	TotalCache              int64
	TotalRss                int64
	TotalRssHuge            int64
	TotalMappedFile         int64
	TotalPgpgin             int64
	TotalPgpgout            int64
	TotalPgfault            int64
	TotalPgmajfault         int64
	TotalInactiveAnon       int64
	TotalActiveAnon         int64
	TotalInactiveFile       int64
	TotalActiveFile         int64
	TotalUnevictable        int64
}

// ReadMemoryStat fills out and returns a MemoryStat struct from the given file name.
// if fileName is "", the default path of /sys/fs/cgroup/memory/memory.stat is used.
func ReadMemoryStat(fileName string) (*MemoryStat, error) {
	if fileName == "" {
		fileName = "/sys/fs/cgroup/memory/memory.stat"
	}
	stat := MemoryStat{}
	kv, err := parseSSKVint64(fileName)
	if err != nil {
		return nil, fmt.Errorf("error parsing %s: %v", fileName, err)
	}
	for k, v := range kv {
		switch k {
		case "cache":
			stat.Cache = v
		case "rss":
			stat.Rss = v
		case "rss_huge":
			stat.RssHuge = v
		case "mapped_file":
			stat.MappedFile = v
		case "pgpgin":
			stat.Pgpgin = v
		case "pgpgout":
			stat.Pgpgout = v
		case "pgfault":
			stat.Pgfault = v
		case "pgmajfault":
			stat.Pgmajfault = v
		case "inactive_anon":
			stat.InactiveAnon = v
		case "active_anon":
			stat.ActiveAnon = v
		case "inactive_file":
			stat.InactiveFile = v
		case "active_file":
			stat.ActiveFile = v
		case "unevictable":
			stat.Unevictable = v
		case "hierarchical_memory_limit":
			stat.HierarchicalMemoryLimit = v
		case "total_cache":
			stat.TotalCache = v
		case "total_rss":
			stat.TotalRss = v
		case "total_rss_huge":
			stat.TotalRssHuge = v
		case "total_mapped_file":
			stat.TotalMappedFile = v
		case "total_pgpgin":
			stat.TotalPgpgin = v
		case "total_pgpgout":
			stat.TotalPgpgout = v
		case "total_pgfault":
			stat.TotalPgfault = v
		case "total_pgmajfault":
			stat.TotalPgmajfault = v
		case "total_inactive_anon":
			stat.TotalInactiveAnon = v
		case "total_active_anon":
			stat.TotalActiveAnon = v
		case "total_inactive_file":
			stat.TotalInactiveFile = v
		case "total_active_file":
			stat.TotalActiveFile = v
		case "total_unevictable":
			stat.TotalUnevictable = v
		}
	}
	return &stat, nil
}
