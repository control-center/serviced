package cmd

import (
	"errors"
	"fmt"

	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/serviced/api"
)

var DefaultHostAPITest = HostAPITest{hosts: DefaultTestHosts}

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
	hosts []host.Host
}

func InitHostAPITest(args ...string) {
	New(DefaultHostAPITest).Run(args)
}

func (t HostAPITest) ListHosts() ([]host.Host, error) {
	return t.hosts, nil
}

func (t HostAPITest) GetHost(id string) (*host.Host, error) {
	for _, h := range t.hosts {
		if h.ID == id {
			return &h, nil
		}
	}

	return nil, ErrNoHostFound
}

func (t HostAPITest) AddHost(config api.HostConfig) (*host.Host, error) {
	if config.IPAddr == "" || config.PoolID == "" {
		return nil, ErrInvalidHost
	}

	h := host.New()
	h.ID = fmt.Sprintf("%s-%s", config.IPAddr, config.PoolID)
	return h, nil
}

func (t HostAPITest) RemoveHost(id string) error {
	for _, h := range t.hosts {
		if h.ID == id {
			return nil
		}
	}

	return ErrNoHostFound
}

func ExampleServicedCli_cmdHostList() {
	InitHostAPITest("serviced", "host", "list", "--verbose")

	// Output:
	// [
	//    {
	//      "ID": "test-host-id-1",
	//      "Name": "alpha",
	//      "PoolID": "default",
	//      "IPAddr": "127.0.0.1",
	//      "Cores": 4,
	//      "Memory": 4294967296,
	//      "PrivateNetwork": "172.16.42.0/24",
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "IPs": null
	//    },
	//    {
	//      "ID": "test-host-id-2",
	//      "Name": "beta",
	//      "PoolID": "default",
	//      "IPAddr": "192.168.0.1",
	//      "Cores": 2,
	//      "Memory": 536870912,
	//      "PrivateNetwork": "10.0.0.1/66",
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "IPs": null
	//    },
	//    {
	//      "ID": "test-host-id-3",
	//      "Name": "gamma",
	//      "PoolID": "testpool",
	//      "IPAddr": "0.0.0.0",
	//      "Cores": 1,
	//      "Memory": 1073741824,
	//      "PrivateNetwork": "158.16.4.27/9090",
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "IPs": null
	//    }
	//  ]
}

func ExampleServicedCli_cmdHostAdd() {
	InitHostAPITest("serviced", "host", "+", "10.0.0.1", "testpool")
	InitHostAPITest("serviced", "host", "add", "127.0.0.1:8080", "default")

	// Output:
	// 10.0.0.1-testpool
	// 127.0.0.1-default
}

func ExampleServicedCli_cmdHostRemove() {
	InitHostAPITest("serviced", "host", "remove", "test-host-id-3")

	// Output:
	// Done
}
