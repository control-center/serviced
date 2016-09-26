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
	"testing"

	"github.com/stretchr/testify/assert"

	regmocks "github.com/control-center/serviced/dfs/registry/mocks"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/user"
)

func TestSetupContainer_DockerLog(t *testing.T) {
	assert := assert.New(t)

	// Create a fake pull registry that doesn't pull images
	fakeRegistry := &regmocks.Registry{}

	// Create a fake HostAgent
	fakeHostAgent := &HostAgent{
		uiport:               ":443",
		dockerLogDriver:      "fakejson-log",
		dockerLogConfig:      map[string]string{"alpha": "one", "bravo": "two", "charlie": "three"},
		virtualAddressSubnet: "0.0.0.0",
		pullreg:              fakeRegistry,
	}

	// Create a fake service.Service
	fakeService := &service.Service{
		ImageID: "busybox:latest",
		ID:      "faketestService",
		Name:    "fakeTestServiceName",
	}
	fakeUser := user.User{}

	// Call setupContainer
	cfg, hcfg, servicestate, err := fakeHostAgent.createContainerConfig("unused", fakeService, 0, fakeUser, "unused")

	assert.NotNil(cfg)
	assert.NotNil(hcfg)
	assert.NotNil(hcfg.LogConfig)
	assert.NotNil(servicestate)
	assert.Nil(err)

	// Test that hostconfig values are as intended
	assert.Equal(hcfg.LogConfig.Type, "fakejson-log")
	assert.Equal(hcfg.LogConfig.Config["alpha"], "one")
	assert.Equal(hcfg.LogConfig.Config["bravo"], "two")
	assert.Equal(hcfg.LogConfig.Config["charlie"], "three")
}
