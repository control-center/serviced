package cmd

import (
	"fmt"

	"github.com/zenoss/serviced/cli/api"
)

var DefaultAPITest = APITest{}

type APITest struct {
	api.API
}

func InitAPITest(args ...string) {
	New(DefaultAPITest).Run(args)
}

func (t APITest) StartServer() {
	fmt.Println("starting server")
}

func ExampleServicedCLI_CmdInit_daemon() {
	InitAPITest("serviced", "--master")

	// Output:
	// starting server
}

func ExampleServicedCLI_CmdInit_logging() {
	InitAPITest("serviced", "--vmodule", "abcd", "--master")

	// Output:
	// syntax error: expect comma-separated list of filename=N
	// starting server
}
