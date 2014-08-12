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
