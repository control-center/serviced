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

package node

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/control-center/serviced/dao/mocks"
	regmocks "github.com/control-center/serviced/dfs/registry/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
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
        "/serviced": "/home/daniel/mygo/src/github.com/control-center/serviced/serviced"
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
		t.Fatalf("Problem unmarshaling test state: %s", err)
	}
}

func TestConfigureContainer_DockerLog(t *testing.T) {
	assert := assert.New(t)

	// Create a fake pull registry that doesn't pull images
	fakeRegistry := &regmocks.Registry{}
	fakeRegistry.On("SetConnection", mock.Anything).Return(nil)
	fakeRegistry.On("ImagePath", mock.Anything).Return("someimage", nil)
	fakeRegistry.On("PullImage", mock.Anything).Return(nil)

	// Create a fake HostAgent
	fakeHostAgent := &HostAgent{
		uiport:               ":443",
		dockerLogDriver:      "fakejson-log",
		dockerLogConfig:      map[string]string{"alpha": "one", "bravo": "two", "charlie": "three"},
		virtualAddressSubnet: "0.0.0.0",
		pullreg:              fakeRegistry,
	}

	// Create a fake client that won't make any RPC calls
	fakeClient := &mocks.ControlPlane{}

	// Create a fake service.Service
	fakeService := &service.Service{
		ImageID: "busybox:latest",
	}

	// Create a fake servicestate.ServiceState
	fakeServiceState := &servicestate.ServiceState{}

	fakeClient.On("GetTenantId", mock.Anything, mock.Anything).Return(nil)
	fakeClient.On("GetSystemUser", mock.Anything, mock.Anything).Return(nil)

	// Call configureContainer
	config, hostconfig, err := configureContainer(
		fakeHostAgent,
		fakeClient,
		fakeService,
		fakeServiceState,
		fakeHostAgent.virtualAddressSubnet)

	assert.NotNil(config)
	assert.NotNil(hostconfig)
	assert.Nil(err)

	// Test that hostconfig values are as intended
	assert.Equal(hostconfig.LogConfig.Type, "fakejson-log")
	assert.Equal(hostconfig.LogConfig.Config["alpha"], "one")
	assert.Equal(hostconfig.LogConfig.Config["bravo"], "two")
	assert.Equal(hostconfig.LogConfig.Config["charlie"], "three")
}
