package serviced

import (
	"github.com/zenoss/serviced/dao"

	"encoding/json"
	"fmt"
	"strings"
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

func TestGetInfo(t *testing.T) {

	ip, err := GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	agent := HostAgent{}
	host := dao.Host{}
	err = agent.GetInfo([]string{}, &host)
	if err != nil {
		t.Errorf("Unexpected error %v", err)
	}
	if len(host.IPs) != 1 {
		t.Errorf("Unexpected result %v", host.IPs)
	}
	if host.IpAddr != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}
	if host.IPs[0].IPAddress != ip {
		t.Errorf("Expected ip %v, got %v", ip, host.IPs)
	}

	err = agent.GetInfo([]string{"127.0.0.1"}, &host)
	if err == nil || err.Error() != "Loopback address 127.0.0.1 cannot be used to register a host" {
		t.Errorf("Unexpected error %v", err)
	}

}

func TestRegisterIPResources(t *testing.T) {

	ips, err := getIPResources("dummy_hostId", "123")
	if err == nil || err.Error() != "IP address 123 not valid for this host" {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ips, err = getIPResources("dummy_hostId", "127.0.0.1")
	if err == nil || err.Error() != "Loopback address 127.0.0.1 cannot be used to register a host" {
		t.Errorf("Unexpected error %v", err)
	}
	if len(ips) != 0 {
		t.Errorf("Unexpected result %v", ips)
	}

	ip, err := GetIPAddress()
	if err != nil {
		t.Fatalf("Unexpected error %v", err)
	}

	validIPs := []string{ip, strings.ToLower(ip), strings.ToUpper(ip), fmt.Sprintf("   %v   ", ip)}
	for _, validIP := range validIPs {
		ips, err = getIPResources("dummy_hostId", validIP)
		if err != nil {
			t.Errorf("Unexpected error %v", err)
		}
		if len(ips) != 1 {
			t.Errorf("Unexpected result %v", ips)
		}
	}
}
