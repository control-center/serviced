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

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// Path to meminfo file. Placed here so getMemorySize() is testable.
var meminfoFile = "/proc/meminfo"

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