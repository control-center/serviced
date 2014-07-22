// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cmd

import (
	"bytes"
	"fmt"
	"io"
	"os"

	"github.com/control-center/serviced/cli/api"
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

var DefaultAPITest = APITest{}

type APITest struct {
	api.API
}

func InitAPITest(args ...string) {
	New(DefaultAPITest).Run(args)
}

func (t APITest) StartServer() error {
	fmt.Println("starting server")
	return nil
}

func ExampleServicedCLI_CmdInit_logging() {
	InitAPITest("serviced", "--logtostderr", "--alsologtostderr", "--master")
	InitAPITest("serviced", "--logstashurl", "127.0.0.1", "-v", "4", "--agent")
	InitAPITest("serviced", "--stderrthreshold", "2", "--vmodule", "a=1,b=2,c=3", "--master", "--agent")
	InitAPITest("serviced", "--log_backtrace_at", "file.go:123", "--master", "--agent")

	// Output:
	// This master has been configured to be in pool: default
	// starting server
	// starting server
	// This master has been configured to be in pool: default
	// starting server
	// This master has been configured to be in pool: default
	// starting server
}

func ExampleServicedCLI_CmdInit_logerr() {
	InitAPITest("serviced", "--master", "--stderrthreshold", "abc")
	InitAPITest("serviced", "--agent", "--vmodule", "abc")
	InitAPITest("serviced", "--master", "--log_backtrace_at", "abc")

	// Output:
	// strconv.ParseInt: parsing "abc": invalid syntax
	// This master has been configured to be in pool: default
	// starting server
	// syntax error: expect comma-separated list of filename=N
	// starting server
	// syntax error: expect file.go:234
	// This master has been configured to be in pool: default
	// starting server
}
