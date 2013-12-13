package agent

import (
	"encoding/json"
	"fmt"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"testing"
	"time"
)

const example_state = `
[{
    "ID": "af5163d96bdc1532875f0f0601ae32a7eafadfdd287d9df6cf2b2020ddfb930d",
    "Created": "2013-09-04T22:35:32.473288901-05:00",
    "Path": "/serviced/serviced",
    "Args": [
        "proxy",
        "dda3d6af-61ef-35ff-4632-086af9b78c90",
        "/bin/nc -l 3306"
    ],
    "Config": {
        "Hostname": "af5163d96bdc",
        "User": "",
        "Memory": 0,
        "MemorySwap": 0,
        "CpuShares": 0,
        "AttachStdin": false,
        "AttachStdout": false,
        "AttachStderr": false,
        "PortSpecs": [
            "3306"
        ],
        "Tty": false,
        "OpenStdin": false,
        "StdinOnce": false,
        "Env": null,
        "Cmd": [
            "/serviced/serviced",
            "proxy",
            "dda3d6af-61ef-35ff-4632-086af9b78c90",
            "/bin/nc -l 3306"
        ],
        "Dns": [
            "8.8.8.8",
            "8.8.4.4"
        ],
        "Image": "base",
        "Volumes": {
            "/serviced": {}
        },
        "VolumesFrom": "",
        "WorkingDir": "",
        "Entrypoint": [],
        "NetworkDisabled": false,
        "Privileged": false
    },
    "State": {
        "Running": true,
        "Pid": 5232,
        "ExitCode": 0,
        "StartedAt": "2013-09-04T22:35:32.485677934-05:00",
        "Ghost": false
    },
    "Image": "b750fe79269d2ec9a3c593ef05b4332b1d1a02a62b4accb2c21d589ff2f5f2dc",
    "NetworkSettings": {
        "IPAddress": "172.17.0.4",
        "IPPrefixLen": 16,
        "Gateway": "172.17.42.1",
        "Bridge": "docker0",
        "PortMapping": {
            "Tcp": {
                "3306": "49156"
            },
            "Udp": {}
        }
    },
    "SysInitPath": "/usr/bin/docker",
    "ResolvConfPath": "/var/lib/docker/containers/af5163d96bdc1532875f0f0601ae32a7eafadfdd287d9df6cf2b2020ddfb930d/resolv.conf",
    "Volumes": {
        "/serviced": "/home/daniel/mygo/src/github.com/zenoss/serviced/serviced"
    },
    "VolumesRW": {
        "/serviced": true
    }
}]
`

// Test parsing container state from docker.
func TestParseContainerState(t *testing.T) {
	var testState []serviced.ContainerState

	err := json.Unmarshal([]byte(example_state), &testState)
	if err != nil {
		t.Fatalf("Problem unmarshaling test state: ", err)
	}
	fmt.Printf("%s", testState)

}

var injectionTests = []struct {
	service  dao.Service
	expected string
}{
	{dao.Service{
		Id:              "1234567890",
		Name:            "ls",
		Context:         "",
		Startup:         "ls",
		Description:     "Run ls",
		Instances:       1,
		ImageId:         "test/ls",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "ls"},
	{dao.Service{
		Id:              "1234567890",
		Name:            "bash",
		Context:         "{\"City\": \"Austin\", \"State\": \"Texas\"}",
		Startup:         "/bin/bash",
		Description:     "Run bash",
		Instances:       1,
		ImageId:         "test/bash",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "/bin/bash"},
	{dao.Service{
		Id:              "1234567890",
		Name:            "/bin/sh",
		Context:         "{\"Command\": \"/bin/sh\"}",
		Startup:         "{{(context .).Command}}",
		Description:     "Run /bin/sh",
		Instances:       1,
		ImageId:         "test/single",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "/bin/sh"},
	{dao.Service{
		Id:              "1234567890",
		Name:            "pinger",
		Context:         "{\"RemoteHost\": \"zenoss.com\", \"Count\": 32}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context .).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		ImageId:         "test/pinger",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}, "/usr/bin/ping -c 32 zenoss.com"},
}

func TestContextInjection(t *testing.T) {
	var client dao.ControlPlane
	for _, it := range injectionTests {
		if err := injectContext(&it.service, client); err != nil {
			t.Error(err)
		}

		result := it.service.Startup

		if result != it.expected {
			t.Errorf("Expecting \"%s\" got \"%s\"\n", it.expected, result)
		}
	}
}

func TestIncompleteInjection(t *testing.T) {
	service := dao.Service{
		Id:              "1234567890",
		Name:            "pinger",
		Context:         "{\"RemoteHost\": \"zenoss.com\"}",
		Startup:         "/usr/bin/ping -c {{(context .).Count}} {{(context .).RemoteHost}}",
		Description:     "Ping a remote host a fixed number of times",
		Instances:       1,
		ImageId:         "test/pinger",
		PoolId:          "default",
		DesiredState:    1,
		Launch:          "auto",
		Endpoints:       []dao.ServiceEndpoint{},
		ParentServiceId: "0987654321",
		CreatedAt:       time.Now(),
		UpdatedAt:       time.Now(),
	}

	var client dao.ControlPlane
	if err := injectContext(&service, client); err != nil {
		t.Error(err)
	}

	result := service.Startup

	if result == "/usr/bin/ping -c 64 zenoss.com" {
		t.Errorf("Not expecting a match")
	}
}
