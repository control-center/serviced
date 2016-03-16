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

// +build integration

package scheduler

import (
	"fmt"
	"testing"
	"time"

	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/datastore/elastic"
	"github.com/control-center/serviced/dfs"
	"github.com/control-center/serviced/dfs/docker"
	"github.com/control-center/serviced/dfs/registry"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/zzk"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) {
	TestingT(t)
}

type LocalSyncTest struct {
	zzk.ZZKTestSuite
	elastic.ElasticTest
	facade    *facade.Facade
	CTX       datastore.Context
	zkConn    coordclient.Connection
	scheduler scheduler
}

var _ = Suite(&LocalSyncTest{})

func (lst *LocalSyncTest) SetUpSuite(c *C) {
	// Init ZZKTestSuite (starts Zookeeper)
	lst.ZZKTestSuite.SetUpSuite(c)

	// Connect to zookeeper
	dsn := coordzk.NewDSN([]string{"127.0.0.1:2181"}, time.Second*15).String()
	//c.Logf("zookeeper dsn: %s", dsn)
	zClient, err := coordclient.New("zookeeper", dsn, "", nil)
	if err != nil {
		c.Fatalf("Could not connect to zookeeper: %s", err)
	}

	zzk.InitializeLocalClient(zClient)

	lst.zkConn, err = zzk.GetLocalConnection("/")
	if err != nil {
		c.Fatalf("Could not get zk connection: %s", err)
	}

	// Init ElasticTest (starts elasticsearch)
	lst.Port = 9202
	lst.MappingsFile = "testmappings.json"
	lst.Index = "controlplane"
	lst.ElasticTest.SetUpSuite(c)

	// Set up Facade
	datastore.Register(lst.Driver())
	lst.CTX = datastore.Get()

	lst.facade = facade.New()
	fzzk := facade.GetFacadeZZK(lst.facade)
	lst.facade.SetZZK(fzzk)
	regindex := registry.NewRegistryIndexClient(lst.facade)
	dockerclient, err := docker.NewDockerClient()
	if err != nil {
		c.Fatalf("Could not get docker client: %s", err)
	}
	dfs := dfs.NewDistributedFilesystem(dockerclient, regindex, nil, nil, nil, 300*time.Second)
	dfs.SetTmp("/tmp/localsync-test")
	lst.facade.SetDFS(dfs)
}

func (lst *LocalSyncTest) SetUpTest(c *C) {
	lst.ZZKTestSuite.SetUpTest(c)
	lst.ElasticTest.SetUpTest(c)
	lst.scheduler.facade = lst.facade
}

func (lst *LocalSyncTest) TearDownTest(c *C) {
	//lst.ElasticTest.TearDownTest(c) // Does not exist
	//lst.ZZKTestSuite.TearDownTest(c) // Does not exist
}

func (lst *LocalSyncTest) TearDownSuite(c *C) {
	lst.ElasticTest.TearDownSuite(c)
	lst.ZZKTestSuite.TearDownSuite(c)
}

// Acceptance test cleanup code used to clean up "too much" and exposed
// this problem (items deleted in middle of sync can get partially
// restored, see CC-1896 and CC-1884), though not reliably. Officially
// test that here now.
func (lst *LocalSyncTest) TestLocalSync_NonInterference(c *C) {
	// Add 10 pools
	for i := 1; i <= 10; i++ {
		newPool := &pool.ResourcePool{
			ID:    fmt.Sprintf("deadpool%d", i),
			Realm: "testRealm",
		}
		c.Logf("Adding pool: %s", newPool.ID)
		if err := lst.facade.AddResourcePool(lst.CTX, newPool); err != nil {
			c.Fatalf("AddResourcePool(%s) failed: $s", newPool.ID, err)
		}
	}

	// Spin off local sync
	go func() {
		time.Sleep(250 * time.Millisecond)
		c.Logf("Calling doSync")
		defer c.Logf("doSync returned")
		lst.scheduler.doSync(lst.zkConn)
	}()

	// Delete all pools
	allPools, err := lst.facade.GetResourcePools(lst.CTX)
	if err != nil {
		c.Fatalf("Could not retrieve list of pools: %s", err)
	}
	for _, pool := range allPools {
		c.Logf("Deleting pool: %s", pool.ID)
		if err := lst.facade.RemoveResourcePool(lst.CTX, pool.ID); err != nil {
			c.Fatalf("Could not remove pool %s: %s", pool.ID, err)
		}
	}

	// Check /pools in ZK
	time.Sleep(time.Second)
	pools, err := lst.zkConn.Children("/pools")
	if len(pools) > 0 {
		c.Errorf("Found these pools in zookeeper, expected none: %v", pools)
	}
}
