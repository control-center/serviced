package serviced

import (
	"encoding/json"
	"testing"
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
	var testState []ContainerState

	err := json.Unmarshal([]byte(example_state), &testState)
	if err != nil {
		t.Fatalf("Problem unmarshaling test state: ", err)
	}
}
