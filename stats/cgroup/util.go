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

package cgroup

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

const (
	Cpuacct string = "cpuacct"
	Memory  string = "memory"
)

// Helper function that takes a docker ID and parameter (to specify cpu or memory)
// and returns a path to the correct stats file
func GetCgroupDockerStatsFilePath(dockerID string, stat string) string {
	// The location of this is dependent on the cgroup driver that docker is using, as
	// well as the version of Docker being used.  To that end, we'll look at all the known
	// locations, and pick the first one that's populated.
	locations := []string{
		// debian(all docker versions) + rhel(docker 1.10-rc1 [and presumably beyond])
		filepath.Join("/sys/fs/cgroup/", stat, "/docker/", dockerID, stat+".stat"),
		// rhel  (<=1.9.0)
		filepath.Join("/sys/fs/cgroup/", stat, "/system.slice/docker-"+dockerID+".scope/", stat+".stat"),
		// rhel (1.9.0 w/ '--exec-opt native.cgroupdriver=cgroupfs' flag, workaround for docker issue #17653
		filepath.Join("/sys/fs/cgroup/", stat, "/system.slice/docker/", dockerID, stat+".stat"),
		// This may be the actual path needed for docker issue #17653, leaving the above for now.
		filepath.Join("/sys/fs/cgroup/", stat, "/system.slice/docker.service/docker/", dockerID, stat+".stat"),
	}
	for _, loc := range locations {
		if _, err := os.Stat(loc); err == nil {
			return loc
		}
	}
	return ""
}

// parseSSKVint64 parses a space-separated key-value pair file and returns a
// key(string):value(int64) mapping.
func parseSSKVint64(filename string) (map[string]int64, error) {
	kv, err := parseSSKV(filename)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]int64)
	for k, v := range kv {
		n, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			uintVal, uintErr := strconv.ParseUint(v, 0, 64)
			if uintErr == nil {
				// the value is 64-bit unsigned, so we will cast it to a signed 64-bit integer.  This will result
				// in an incorrect value, but will allow us to report other metrics without breaking everything
				n = int64(uintVal)
			} else {
				return nil, err
			}
		}
		mapping[k] = n
	}
	return mapping, nil
}

// parseSSKV parses a space-separated key-value pair file and returns a
// key(string):value(string) mapping.
func parseSSKV(filename string) (map[string]string, error) {
	stats, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]string)
	scanner := bufio.NewScanner(strings.NewReader(string(stats)))
	for scanner.Scan() {
		line := scanner.Text()
		parts := strings.Fields(line)
		if len(parts) != 2 {
			return nil, fmt.Errorf("expected 2 parts, got %d: %s", len(parts), line)
		}
		mapping[parts[0]] = parts[1]
	}
	return mapping, nil
}
