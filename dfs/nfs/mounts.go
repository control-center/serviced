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

package nfs

import (
	"bufio"
	"fmt"
	"io"
	"strconv"
	"strings"
)

type mountOptions map[string]string

// String prints options in manner that mimics fstab format; ie only append =val when
// val is not empty.
func (options mountOptions) String() string {
	vals := make([]string, len(options))
	i := 0
	for k, v := range options {
		if len(v) == 0 {
			vals[i] = k
		} else {
			vals[i] = k + "=" + v
		}
		i++
	}
	return strings.Join(vals, ",")
}

func parseMountOptions(options string) mountOptions {
	parsedOptions := make(mountOptions)
	for _, v := range strings.Split(options, ",") {
		parts := strings.Split(v, "=")
		if len(parts) > 1 {
			parsedOptions[parts[0]] = parts[1]
		} else {
			parsedOptions[parts[0]] = ""
		}
	}
	return parsedOptions
}

type mountInstance struct {
	Src       string
	Dst       string
	Type      string
	Options   mountOptions
	Dump      int
	FsckOrder int
}

func parseMounts(reader io.Reader) (mounts []mountInstance, err error) {
	scanner := bufio.NewScanner(reader)
	for scanner.Scan() {
		parts := strings.Fields(scanner.Text())
		if len(parts) < 6 {
			return mounts, fmt.Errorf("invalid mount spec")
		}
		dump, err := strconv.ParseInt(parts[4], 10, 16)
		if err != nil {
			return mounts, fmt.Errorf("error parsing dump value: %s", err)
		}
		fsckOrder, err := strconv.ParseInt(parts[5], 10, 16)
		if err != nil {
			return mounts, fmt.Errorf("error parsing fsck order: %s", err)
		}
		instance := mountInstance{
			Src:       parts[0],
			Dst:       parts[1],
			Type:      parts[2],
			Options:   parseMountOptions(parts[3]),
			Dump:      int(dump),
			FsckOrder: int(fsckOrder),
		}
		mounts = append(mounts, instance)
	}
	return mounts, nil
}

