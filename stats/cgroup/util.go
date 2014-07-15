// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cgroup

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"strconv"
	"strings"
)

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
