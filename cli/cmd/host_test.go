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
	fail  bool
	pools []pool.ResourcePool
	hosts []host.Host
}

func InitHostAPITest(args ...string) {
	New(DefaultHostAPITest, utils.TestConfigReader(make(map[string]string))).Run(args)
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

func (t HostAPITest) AddHost(config api.HostConfig) (*host.Host, error) {
	if t.fail {
		return nil, ErrInvalidHost
	} else if config.PoolID == NilPool {
		return nil, nil
	}

	h := host.New()
	h.ID = fmt.Sprintf("%s-%s", config.Address.Host, config.PoolID)
	return h, nil
}

func (t HostAPITest) RemoveHost(id string) error {
	if h, err := t.GetHost(id); err != nil {
		return err
	} else if h == nil {
		return ErrNoHostFound
	}
	return nil
}

func TestServicedCLI_CmdHostList_one(t *testing.T) {
	hostID := "test-host-id-1"

	expected, err := DefaultHostAPITest.GetHost(hostID)
	if err != nil {
		t.Fatal(err)
	}

	var actual host.Host
	output := pipe(InitHostAPITest, "serviced", "host", "list", "test-host-id-1")
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
	output := pipe(InitHostAPITest, "serviced", "host", "list", "--verbose")
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
	pipeStderr(InitHostAPITest, "serviced", "host", "list", "test-host-id-1")
	// Error retrieving all hosts
	pipeStderr(InitHostAPITest, "serviced", "host", "list")

	// Output:
	// invalid host
	// invalid host
}

func ExampleServicedCLI_CmdHostList_err() {
	DefaultHostAPITest.hosts = make([]host.Host, 0)
	defer func() { DefaultHostAPITest.hosts = DefaultTestHosts }()
	// Host not found
	pipeStderr(InitHostAPITest, "serviced", "host", "list", "test-host-id-0")
	// No hosts found
	pipeStderr(InitHostAPITest, "serviced", "host", "list")

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

func ExampleServicedCLI_CmdHostAdd() {
	// Bad URL
	InitHostAPITest("serviced", "host", "add", "badurl", "default")
	// Success
	InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", "default")

	// Output:
	// bad format: badurl; must be formatted as HOST:PORT
	// 127.0.0.1-default
}

func ExampleServicedCLI_CmdHostAdd_usage() {
	InitHostAPITest("serviced", "host", "add")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    add - Adds a new host
	//
	// USAGE:
	//    command add [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced host add HOST:PORT RESOURCE_POOL
	//
	// OPTIONS:
	//    --memory 	Memory to allocate on this host, e.g. 20G, 50%
}

func ExampleServicedCLI_CmdHostAdd_fail() {
	DefaultHostAPITest.fail = true
	defer func() { DefaultHostAPITest.fail = false }()
	pipeStderr(InitHostAPITest, "serviced", "host", "add", "127.0.0.1:8080", "default")

	// Output:
	// invalid host
}

func ExampleServicedCLI_CmdHostAdd_err() {
	pipeStderr(InitHostAPITest, "serviced", "host", "add", "127.0.0.1:8080", NilPool)

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
	pipeStderr(InitHostAPITest, "serviced", "host", "remove", "test-host-id-0")

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
