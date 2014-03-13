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
	kv, err := parseSSKV(filename)
	if err != nil {
		return nil, err
	}
	mapping := make(map[string]int64)
	for k, v := range kv {
		n, err := strconv.ParseInt(v, 0, 64)
		if err != nil {
			return nil, err
		}
		mapping[k] = n
	}
	return mapping, nil
}

// Parses a space-separated key-value pair file and returns a
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
		parts := strings.Split(line, " ")
		mapping[parts[0]] = parts[1]
	}
	return mapping, nil
}
