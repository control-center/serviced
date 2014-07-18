// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

// Package cgroup provides access to /sys/fs/cgroup metrics.

package cgroup

import (
	"io/ioutil"
	"math/rand"
	"strconv"
	"testing"
	"time"
)

const chars = "1234567890!@#$%%^&*()QWERTYUIOPASDFGHJKL:\"{}|ZXCVBNM<>?qwertyuiop[]\\asdfghjkl;'zxcvbnm,./"

func randomChar() string {
	return string(chars[rand.Intn(len(chars))])
}

func randomString() string {
	str := randomChar()
	for rand.Float64() > 0.05 {
		str += randomChar()
	}
	return str
}

func randomMapStringInt64() map[string]int64 {
	data := make(map[string]int64)
	for rand.Float64() > 0.01 {
		data[randomString()] = int64(rand.Intn(9999999999))
	}
	return data
}

func randomMapStringString() map[string]string {
	data := make(map[string]string)
	for rand.Float64() > 0.01 {
		data[randomString()] = randomString()
	}
	return data
}

func writeSSKVint64(data map[string]int64, filename string) {
	filedata := ""
	for k, v := range data {
		filedata += k + " " + strconv.Itoa(int(v)) + "\n"
	}
	_ = ioutil.WriteFile(filename, []byte(filedata), 0x777)
}

func writeSSKVstring(data map[string]string, filename string) {
	filedata := ""
	for k, v := range data {
		filedata += k + " " + v + "\n"
	}
	_ = ioutil.WriteFile(filename, []byte(filedata), 0x777)
}

func tempFileName() string {
	f, _ := ioutil.TempFile("", "prefix")
	defer f.Close()
	return f.Name()
}

func TestParseSSKVint64(t *testing.T) {
	rand.Seed(int64(time.Now().Nanosecond()))
	data := randomMapStringInt64()
	filename := tempFileName()
	writeSSKVint64(data, filename)
	testdata, err := parseSSKVint64(filename)
	if err != nil {
		t.Error(err)
	}
	for k := range data {
		if data[k] != testdata[k] {
			t.Error(data[k], "!=", testdata[k])
		}
	}
	for k := range testdata {
		if data[k] != testdata[k] {
			t.Error(data[k], "!=", testdata[k])
		}
	}
}

func TestParseSSKV(t *testing.T) {
	rand.Seed(int64(time.Now().Nanosecond()))
	data := randomMapStringString()
	filename := tempFileName()
	writeSSKVstring(data, filename)
	testdata, err := parseSSKV(filename)
	if err != nil {
		t.Error(err)
	}
	for k := range data {
		if data[k] != testdata[k] {
			t.Error(data[k], "!=", testdata[k])
		}
	}
	for k := range testdata {
		if data[k] != testdata[k] {
			t.Error(data[k], "!=", testdata[k])
		}
	}
}
