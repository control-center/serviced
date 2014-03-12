package cgroup

import (
	"bufio"
	"io/ioutil"
	"strconv"
	"strings"
)

// Parses a space-separated key-value pair file and returns a
// key(string):value(int64) mapping.
func parseSSKVint64(filename string) (map[string]int64, error) {
	if kv, err := parseSSKV(filename); err != nil {
		return nil, err
	} else {
		mapping := make(map[string]int64)
		for k, v := range kv {
			if n, err := strconv.ParseInt(v, 0, 64); err != nil {
				return nil, err
			} else {
				mapping[k] = n
			}
		}
		return mapping, nil
	}
	return nil, nil
}

// Parses a space-separated key-value pair file and returns a
// key(string):value(string) mapping.
func parseSSKV(filename string) (map[string]string, error) {
	if stats, err := ioutil.ReadFile(filename); err != nil {
		return nil, err
	} else {
		mapping := make(map[string]string)
		scanner := bufio.NewScanner(strings.NewReader(string(stats)))
		for scanner.Scan() {
			line := scanner.Text()
			parts := strings.Split(line, " ")
			mapping[parts[0]] = parts[1]
		}
		return mapping, nil
	}
}
