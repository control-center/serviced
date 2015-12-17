// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build integration

package isvcs

import (
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)


var (
	defaultTestDockerLogDriver = "json-file"
	defaultTestDockerLogOptions = map[string]string{"max-file": "5", "max-size": "10m"}
)

type IServiceTest interface {
	GetService(c *C) *IService
	Create(c *C)
	Destroy(c *C)
	SetUp(c *C)
}

type ManagerTestSuite struct {
	manager      *Manager
	testservices []IServiceTest
}

func (t *ManagerTestSuite) SetUpSuite(c *C) {
	docker.StartKernel()
	t.manager = NewManager(utils.LocalDir("images"), "/tmp/serviced-test", defaultTestDockerLogDriver, defaultTestDockerLogOptions)
	for _, testservice := range t.testservices {
		svc := testservice.GetService(c)
		if err := t.manager.Register(svc); err != nil {
			c.Fatalf("Could not register %s: %s", svc.Name, err)
		}
	}
	t.manager.Wipe()
	if err := t.manager.Start(); err != nil {
		c.Fatalf("Could not start isvcs: %s", err)
	}

	for _, testservice := range t.testservices {
		testservice.Create(c)
	}
}

func (t *ManagerTestSuite) SetUpTest(c *C) {
	for _, testservice := range t.testservices {
		testservice.SetUp(c)
	}
}

func (t *ManagerTestSuite) TearDownSuite(c *C) {
	for _, testservice := range t.testservices {
		testservice.Destroy(c)
	}
	t.manager.Stop()
}

func (t *ManagerTestSuite) AddTestService(svc IServiceTest) {
	t.testservices = append(t.testservices, svc)
}
