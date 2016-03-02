// Copyright 2016 The Serviced Authors.
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

package utils

//#include <stdlib.h>
//extern int fd_lock(int fd, char* filepath);
//extern int fd_unlock(int fd, char* filepath);
import "C"

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"strconv"
	"strings"
)

const (
	ioctlTermioFlag = syscall.TCGETS
	hostIDCmdString = "/usr/bin/hostid"
)

// Path to meminfo file. Placed here so getMemorySize() is testable.
var meminfoFile = "/proc/meminfo"

func determinePlatform() int {
	if _, err := os.Stat("/etc/redhat-release"); err == nil {
		return Rhel
	} else {
		return Debian
	}
}

// getHostID retrieves the system's unique id, on linux this maps
// to /usr/bin/hostid.
func getHostID() (hostid string, err error) {
	cmd := exec.Command(hostIDCmdString)
	stdout, err := cmd.Output()
	if err != nil {
		return hostid, err
	}
	return strings.TrimSpace(string(stdout)), err
}

// getMemorySize attempts to get the size of the installed RAM.
func getMemorySize() (size uint64, err error) {
	file, err := os.Open(meminfoFile)
	if err != nil {
		return 0, err
	}
	defer file.Close()
	reader := bufio.NewReader(file)
	line, err := reader.ReadString('\n')
	for err == nil {
		if strings.Contains(line, "MemTotal:") {
			parts := strings.Fields(line)
			if len(parts) < 3 {
				return 0, err
			}
			size, err := strconv.Atoi(parts[1])
			if err != nil {
				return 0, err
			}
			return uint64(size) * 1024, nil
		}
		line, err = reader.ReadString('\n')
	}
	return 0, err
}

// Returns a list of network routes
func getRoutes() (routes []RouteEntry, err error) {
	output, err := exec.Command("/sbin/route", "-A", "inet").Output()
	if err != nil {
		return routes, err
	}

	columnMap := make(map[string]int)
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	if len(lines) < 2 {
		return routes, fmt.Errorf("no routes found")
	}
	routes = make([]RouteEntry, len(lines)-2)
	for lineNum, line := range lines {
		line = strings.TrimSpace(line)
		switch {
		// skip first line
		case lineNum == 0:
			continue
		case lineNum == 1:
			for number, name := range strings.Fields(line) {
				columnMap[name] = number
			}
			continue
		default:
			fields := strings.Fields(line)
			metric, err := strconv.Atoi(fields[columnMap["Metric"]])
			if err != nil {
				return routes, err
			}
			ref, err := strconv.Atoi(fields[columnMap["Ref"]])
			if err != nil {
				return routes, err
			}
			use, err := strconv.Atoi(fields[columnMap["Use"]])
			if err != nil {
				return routes, err
			}
			routes[lineNum-2] = RouteEntry{
				Destination: fields[columnMap["Destination"]],
				Gateway:     fields[columnMap["Gateway"]],
				Genmask:     fields[columnMap["Genmask"]],
				Flags:       fields[columnMap["Flags"]],
				Metric:      metric,
				Ref:         ref,
				Use:         use,
				Iface:       fields[columnMap["Iface"]],
			}
		}
	}
	return routes, err
}
