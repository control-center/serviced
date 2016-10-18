// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//	   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/utils"
)

func pipe(f func(...string), args ...string) []byte {
	r, w, _ := os.Pipe()
	stdout := os.Stdout
	os.Stdout = w

	f(args...)

	output := make(chan []byte)
	go func() {
		var buffer bytes.Buffer
		io.Copy(&buffer, r)
		output <- buffer.Bytes()
	}()

	w.Close()
	os.Stdout = stdout
	return <-output
}

func pipeAPI(f func(api.API, ...string), test api.API, args ...string) []byte {
	p := func(args ...string) {
		f(test, args...)
	}
	return pipe(p, args...)
}

func pipeStderr(f func(...string), args ...string) {
	r, w, _ := os.Pipe()
	stderr := os.Stderr
	os.Stderr = w

	f(args...)

	output := make(chan []byte)
	go func() {
		var buffer bytes.Buffer
		io.Copy(&buffer, r)
		output <- buffer.Bytes()
	}()
	w.Close()
	os.Stderr = stderr
	fmt.Printf("%s", <-output)
}

func pipeAPIStderr(f func(api.API, ...string), test api.API, args ...string) {
	p := func(args ...string) {
		f(test, args...)
	}
	pipeStderr(p, args...)
}

// Trims leading and trailing whitespace from each line of a multi-line string
func TrimLines(valueToTrim string) string {
	re := regexp.MustCompile("\\s*\\n\\s*")
	trimmedOutput := re.ReplaceAllString(valueToTrim, "\n")
	return strings.TrimSpace(trimmedOutput)
}

var DefaultAPITest = APITest{}

type APITest struct {
	api.API
}

func InitAPITest(args ...string) {
	New(DefaultAPITest, utils.TestConfigReader(map[string]string{}), MockLogControl{}).Run(args)
}

func (t APITest) StartServer() error {
	fmt.Println("starting server")
	return nil
}
