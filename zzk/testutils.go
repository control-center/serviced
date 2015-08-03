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

package zzk

import (
	"time"

	"github.com/control-center/serviced/coordinator/client"
	"github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/isvcs"
	. "gopkg.in/check.v1"
)

// NOTE: this constant can be adjusted to satisfy race conditions
const ZKTestTimeout = 5 * time.Second

type ZZKTestSuite struct {
	isvcs.ManagerTestSuite
}

func (t *ZZKTestSuite) SetUpSuite(c *C) {
	t.ManagerTestSuite.AddTestService(t)
	t.ManagerTestSuite.SetUpSuite(c)
}

func (t *ZZKTestSuite) GetService(c *C) *isvcs.IService {
	// NOTE: if the service needs to be modified, copy the global var, rather than
	// overwrite it
	svcdef := isvcs.Zookeeper

	zk, err := isvcs.NewIService(svcdef)
	if err != nil {
		c.Fatalf("Error initializing zookeeper container: %s", err)
	}
	return zk
}

func (t *ZZKTestSuite) Create(c *C) {
	dsn := zookeeper.NewDSN([]string{"127.0.0.1:2181"}, time.Second*15).String()
	c.Logf("zookeeper dsn: %s", dsn)
	zclient, err := client.New("zookeeper", dsn, "", nil)
	if err != nil {
		c.Fatalf("Could not connect to zookeeper container: %s", err)
	}
	InitializeLocalClient(zclient)
}

func (t *ZZKTestSuite) Destroy(c *C) {
	ShutdownConnections()
}

func (t *ZZKTestSuite) SetUp(c *C) {
	// delete the contents of zookeeper for every test
	conn, err := GetLocalConnection("/")
	if err != nil {
		c.Fatalf("Could not get connection to zookeeper: %s", err)
	}

	children, err := conn.Children("/")
	for _, child := range children {
		if err := conn.Delete("/" + child); err != nil {
			c.Logf("Could not delete %s: %s", child, err)
		}
	}
}
