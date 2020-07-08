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

// +build unit

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/utils"
)

var DefaultHostAPITest = HostAPITest{
	pools: DefaultTestPools,
	hosts: DefaultTestHosts,
}

var DefaultTestHosts = []host.Host{
	{
		ID:             "test-host-id-1",
		PoolID:         "default",
		Name:           "alpha",
		IPAddr:         "127.0.0.1",
		Cores:          4,
		Memory:         4 * 1024 * 1024 * 1024,
		PrivateNetwork: "172.16.42.0/24",
	}, {
		ID:             "test-host-id-2",
		PoolID:         "default",
		Name:           "beta",
		IPAddr:         "192.168.0.1",
		Cores:          2,
		Memory:         512 * 1024 * 1024,
		PrivateNetwork: "10.0.0.1/66",
	}, {
		ID:             "test-host-id-3",
		PoolID:         "testpool",
		Name:           "gamma",
		IPAddr:         "0.0.0.0",
		Cores:          1,
		Memory:         1 * 1024 * 1024 * 1024,
		PrivateNetwork: "158.16.4.27/9090",
	},
}

var (
	ErrNoHostFound = errors.New("no host found")
	ErrInvalidHost = errors.New("invalid host")
)

type HostAPITest struct {
	api.API
	fail         bool
	registerFail bool
	writeFail    bool
	pools        []pool.ResourcePool
	hosts        []host.Host
}

func InitHostAPITest(args ...string) {
	c := New(DefaultHostAPITest, utils.TestConfigReader(make(map[string]string)), MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}

func (t HostAPITest) GetHosts() ([]host.Host, error) {
	if t.fail {
		return nil, ErrInvalidHost
	}
	return t.hosts, nil
}

func (t HostAPITest) GetResourcePools() ([]pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidPool
	}
	return t.pools, nil
}

func (t HostAPITest) GetHost(id string) (*host.Host, error) {
	if t.fail {
		return nil, ErrInvalidHost
	}

	for _, h := range t.hosts {
		if h.ID == id {
			return &h, nil
		}
	}

	return nil, nil
}

func (t HostAPITest) AddHost(config api.HostConfig) (*host.Host, []byte, error) {
	if t.fail {
		return nil, nil, ErrInvalidHost
	} else if config.PoolID == NilPool {
		return nil, nil, nil
	}

	h := host.New()
	h.ID = fmt.Sprintf("%s-%s", config.Address.Host, config.PoolID)
	h.PoolID = config.PoolID
	h.IPAddr = config.Address.Host
	return h, []byte("Fake HostKeys"), nil
}

func (t HostAPITest) RemoveHost(id string) error {
	if h, err := t.GetHost(id); err != nil {
		return err
	} else if h == nil {
		return ErrNoHostFound
	}
	return nil
}

func (t HostAPITest) RegisterRemoteHost(h *host.Host, nat utils.URL, data []byte, prompt bool) error {
	if t.registerFail {
		return errors.New("Forcing RemoteRegisterHost to fail for testing")
	}
	return nil
}

func (t HostAPITest) WriteDelegateKey(filename string, data []byte) error {
	if t.writeFail {
		return errors.New("Forcing WriteDelegateKey to fail for testing")
	}
	return nil
}

func (t HostAPITest) GetHostWithAuthInfo(id string) (*api.AuthHost, error) {
	if t.fail {
		return nil, ErrInvalidHost
	}

	for _, h := range t.hosts {
		if h.ID == id {
			return &api.AuthHost{h, true}, nil
		}
	}

	return nil, nil
}

func (t HostAPITest) GetHostsWithAuthInfo() ([]api.AuthHost, error) {
	if t.fail {
		return nil, ErrInvalidHost
	}
	authHosts := []api.AuthHost{}
	for _, h := range t.hosts {
		authHosts = append(authHosts, api.AuthHost{h, true})
	}
	return authHosts, nil
}

func TestServicedCLI_CmdHostList_one(t *testing.T) {
	hostID := "test-host-id-1"

	expected, err := DefaultHostAPITest.GetHost(hostID)
	if err != nil {
		t.Fatal(err)
	}

	var actual host.Host
	output := captureStdout(func() { InitHostAPITest("serviced", "host", "list", "test-host-id-1") })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update Host.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdHostList_all(t *testing.T) {
	expected, err := DefaultHostAPITest.GetHosts()
	if err != nil {
		t.Fatal(err)
	}

	var actual []host.Host
	output := captureStdout(func() { InitHostAPITest("serviced", "host", "list", "--verbose") })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update Host.Equals?
	if len(actual) != len(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
	for i, _ := range actual {
		if !actual[i].Equals(&expected[i]) {
			t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
		}
	}
}

func ExampleServicedCLI_CmdHostList() {
	// The result displays spaces at the end of each row, which gofmt cleans up
	InitHostAPITest("serviced", "host", "list")
}

func ExampleServicedCLI_CmdHostList_fail() {
	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	// Error retrieving host
	pipeStderr(func() { InitHostAPITest("serviced", "host", "list", "test-host-id-1") })
	// Error retrieving all hosts
	pipeStderr(func() { InitHostAPITest("serviced", "host", "list") })

	// Output:
	// invalid host
	// invalid host
}

func ExampleServicedCLI_CmdHostList_err() {
	DefaultHostAPITest.hosts = make([]host.Host, 0)
	defer func() { DefaultHostAPITest.hosts = DefaultTestHosts }()
	// Host not found
	pipeStderr(func() { InitHostAPITest("serviced", "host", "list", "test-host-id-0") })
	// No hosts found
	pipeStderr(func() { InitHostAPITest("serviced", "host", "list") })

	// Output:
	// host not found
	// no hosts found
}

func ExampleServicedCLI_CmdHostList_complete() {
	InitHostAPITest("serviced", "host", "list", "--generate-bash-completion")

	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	InitHostAPITest("serviced", "host", "list", "--generate-bash-completion")

	// Output:
	// test-host-id-1
	// test-host-id-2
	// test-host-id-3
}

/* The output of this command is dynamic, so disabling until we figure out how to do this

func ExampleServicedCLI_CmdHostAdd() {
	// Success
	InitHostAPITest("serviced", "host", "add", "127.0.0.111:8080", "default")

	// Output:
	// Wrote delegate key file to IP-127-0-0-111.delegate.key
	// 127.0.0.111-default
}
*/

func ExampleServicedCLI_CmdHostAdd_register() {
	// Register host.  Do not write key file
	InitHostAPITest("serviced", "host", "add", "--register", "127.0.0.33:8080", "default")

	// Output:
	// Registered host at 127.0.0.33
	// 127.0.0.33-default
}

/* The output of this command is dynamic, so disabling until we figure out how to do this

func ExampleServicedCLI_CmdHostAdd_registerfail() {
	// Register host failed.  Write key file
	DefaultHostAPITest.registerFail = true
	defer func() { DefaultHostAPITest.registerFail = false }()
	pipeStderr(func(){InitHostAPITest( "serviced", "host", "add", "--register", "127.0.0.1:8080", "default")})

	// Output:
	// Wrote delegate key file to IP-127-0-0-1.delegate.key
	// 127.0.0.1-default
	// Error registering host: Forcing RemoteRegisterHost to fail for testing
}
*/

/* The output of this command is dynamic, so disabling until we figure out how to do this

func ExampleServicedCLI_CmdHostAdd_keyfile() {
	// Specify location of key file.
	pipeStderr(func(){InitHostAPITest( "serviced", "host", "add", "--key-file", "foobar", "127.0.0.1:8080", "default")})

	// Output:
	// Wrote delegate key file to foobar
	// 127.0.0.1-default
}
*/

/* The output of this command is dynamic, so disabling until we figure out how to do this

func ExampleServicedCLI_CmdHostAdd_keyfilefail() {
	// Failure writing keyfile
	DefaultHostAPITest.writeFail = true
	defer func() { DefaultHostAPITest.writeFail = false }()
	pipeStderr(func(){InitHostAPITest( "serviced", "host", "add", "127.0.0.1:8080", "default")})

	// Output:
	// 127.0.0.1-default
	// Error writing delegate key file "IP-127-0-0-1.delegate.key": Forcing WriteDelegateKey to fail for testing
}
*/

/* The output of this command is dynamic, so disabling until we figure out how to do this

func ExampleServicedCLI_CmdHostAdd_keyfileRegister() {
	// Specify location of key file and register host.
	// Write the key file even though we registered.
	pipeStderr(func(){InitHostAPITest( "serviced", "host", "add", "--register", "--key-file", "foobar", "127.0.0.3:8080", "default")})

	// Output:
	// Registered host at 127.0.0.3
	// Wrote delegate key file to foobar
	// 127.0.0.3-default
}
*/

func ExampleServicedCLI_CmdHostAdd_badurl() {
	// Bad URL
	InitHostAPITest("serviced", "host", "add", "badurl", "default")

	// Output:
	// bad format: badurl; must be formatted as HOST:PORT
}

func ExampleServicedCLI_CmdHostAdd_fail() {
	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	pipeStderr(func() { InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", "default") })

	// Output:
	// invalid host
}

func ExampleServicedCLI_CmdHostAdd_err() {
	pipeStderr(func() { InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", NilPool) })

	// Output:
	// received nil host
}

func ExampleServicedCLI_CmdHostAdd_complete() {
	InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", "--generate-bash-completion")

	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", "--generate-bash-completion")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
	// test-pool-id-3
}

func ExampleServicedCLI_CmdHostRemove() {
	InitHostAPITest("serviced", "host", "remove", "test-host-id-3")

	// Output:
	// test-host-id-3
}

func ExampleServicedCLI_CmdHostRemove_usage() {
	InitHostAPITest("serviced", "host", "rm")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    remove - Removes an existing host
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced host remove HOSTID ...
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdHostRemove_err() {
	pipeStderr(func() { InitHostAPITest("serviced", "host", "remove", "test-host-id-0") })

	// Output:
	// test-host-id-0: no host found
}

func ExampleServicedCLI_CmdHostRemove_complete() {
	InitHostAPITest("serviced", "host", "rm", "--generate-bash-completion")
	fmt.Println("")
	InitHostAPITest("serviced", "host", "rm", "test-host-id-2", "--generate-bash-completion")

	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	InitHostAPITest("serviced", "host", "rm", "--generate-bash-completion")

	// Output:
	// test-host-id-1
	// test-host-id-2
	// test-host-id-3
	//
	// test-host-id-1
	// test-host-id-3
}

func ExampleServicedCLI_CmdHostRegister_usage() {
	InitHostAPITest("serviced", "host", "register")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    register - Set the authentication keys to use for this host. When KEYSFILE is -, read from stdin.

	// USAGE:
	//    command register [command options] [arguments...]

	// DESCRIPTION:
	//    serviced host register KEYSFILE

	// OPTIONS:
}
