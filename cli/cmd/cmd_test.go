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
	New(DefaultAPITest, utils.TestConfigReader(map[string]string{})).Run(args)
}

func (t APITest) StartServer() error {
	fmt.Println("starting server")
	return nil
}

func ExampleServicedCLI_CmdInit_logging() {
	InitAPITest("serviced", "--logtostderr", "--alsologtostderr", "--master", "--allow-loop-back=true", "server")
	InitAPITest("serviced", "--logstashurl", "127.0.0.1", "-v", "4", "--agent", "--endpoint", "1.2.3.4:4979", "server")
	InitAPITest("serviced", "--stderrthreshold", "2", "--vmodule", "a=1,b=2,c=3", "--master", "--agent", "--allow-loop-back=true", "server")
	InitAPITest("serviced", "--log_backtrace_at", "file.go:123", "--master", "--agent", "--allow-loop-back=true", "server")

	// Output:
	// starting server
	// starting server
	// starting server
	// starting server
}

func ExampleServicedCLI_CmdInit_logerr() {
	InitAPITest("serviced", "--master", "--stderrthreshold", "abc", "--allow-loop-back=true", "server")
	InitAPITest("serviced", "--agent", "--endpoint", "5.6.7.8:4979", "--vmodule", "abc", "server")
	InitAPITest("serviced", "--master", "--log_backtrace_at", "abc", "--allow-loop-back=true", "server")

	// Output:
	// Unable to set logging options: strconv.ParseInt: parsing "abc": invalid syntax
	// starting server
	// Unable to set logging options: syntax error: expect comma-separated list of filename=N
	// starting server
	// Unable to set logging options: syntax error: expect file.go:234
	// starting server
}
