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

// +build integration,!quick

package elasticsearch

import (
	"fmt"
	"os"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/config"
	coordclient "github.com/control-center/serviced/coordinator/client"
	coordzk "github.com/control-center/serviced/coordinator/client/zookeeper"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/isvcs"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume"
	_ "github.com/control-center/serviced/volume/rsync"
	"github.com/control-center/serviced/zzk"
	"github.com/zenoss/glog"
	. "gopkg.in/check.v1"
)

const (
	HOSTID = "deadbeef"
)

var unused int
var unusedStr string
var id string
var addresses []string
var version datastore.VersionedEntity

var err error

// MockStorageDriver is an interface that mock the storage subsystem
type MockStorageDriver struct {
	exportPath string
}

func (m MockStorageDriver) ExportPath() string {
	return m.exportPath
}

func (m MockStorageDriver) SetClients(clients ...string) {
}

func (m MockStorageDriver) Sync() error {
	return nil
}

func (m MockStorageDriver) Restart() error {
	return nil
}

func (m MockStorageDriver) Stop() error {
	return nil
}

func TestMain(m *testing.M) {
	docker.StartKernel()
	os.Exit(m.Run())
}

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

//Instantiate the gocheck suite. Initialize the DaoTest and the embedded FacadeIntegrationTset
var _ = Suite(&DaoTest{})

//DaoTest gocheck test type for setting up isvcs and other resources needed by tests
type DaoTest struct {
	facade.FacadeIntegrationTest
	Dao    *ControlPlaneDao
	zkConn coordclient.Connection
}

//SetUpSuite is run before the tests to ensure elastic, zookeeper etc. are running.
func (dt *DaoTest) SetUpSuite(c *C) {

	config.LoadOptions(config.Options{
		IsvcsPath: c.MkDir(),
	})

	dt.Port = 9202
	isvcs.Init(isvcs.DEFAULT_ES_STARTUP_TIMEOUT_SECONDS, "json-file", map[string]string{"max-file": "5", "max-size": "10m"}, nil, true, false)
	isvcs.Mgr.SetVolumesDir(c.MkDir())
	esServicedClusterName, _ := utils.NewUUID36()
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-serviced", "cluster", esServicedClusterName); err != nil {
		c.Fatalf("Could not set elasticsearch-serviced clustername: %s", err)
	}
	esLogstashClusterName, _ := utils.NewUUID36()
	if err := isvcs.Mgr.SetConfigurationOption("elasticsearch-logstash", "cluster", esLogstashClusterName); err != nil {
		c.Fatalf("Could not set elasticsearch-logstash clustername: %s", err)
	}
	if err := isvcs.Mgr.Start(); err != nil {
		c.Fatalf("Could not start es container: %s", err)
	}
	dt.MappingsFile = "controlplane.json"
	dt.FacadeIntegrationTest.SetUpSuite(c)

	dsn := coordzk.NewDSN([]string{"127.0.0.1:2181"},
		time.Second*15,
		1*time.Second,
		0,
		1*time.Second,
		1*time.Second,
	).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zClient, err := coordclient.New("zookeeper", dsn, "", nil)
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	}

	zzk.InitializeLocalClient(zClient)

	dt.zkConn, err = zzk.GetLocalConnection("/")
	if err != nil {
		c.Fatalf("could not get zk connection %v", err)
	}

	tmpdir := c.MkDir()
	err = volume.InitDriver(volume.DriverTypeRsync, tmpdir, []string{})
	c.Assert(err, IsNil)

	dt.Dao, err = NewControlSvc("localhost", int(dt.Port), dt.Facade, "", 4979)
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	} else {
		for i := 0; i < 10; i += 1 {
			id := strconv.Itoa(i)
			dt.Dao.RemoveService(id, &unused)
		}
		for i := 100; i < 110; i += 1 {
			id := strconv.Itoa(i)
			dt.Dao.RemoveService(id, &unused)
		}
	}
}

//SetUpTest run before each test.
func (dt *DaoTest) SetUpTest(c *C) {
	//Facade tests delete the contents of the database for every test
	dt.FacadeIntegrationTest.SetUpTest(c)
	//DAO tests expect default pool and system user

	if err := dt.Facade.CreateDefaultPool(dt.CTX, "default"); err != nil {
		c.Fatalf("could not create default pool: %s", err)
	}

	// create the account credentials
	if err := dt.Facade.CreateSystemUser(dt.CTX); err != nil {
		c.Fatalf("could not create systemuser: %s", err)
	}
}

//TearDownSuite stops all isvcs
func (dt *DaoTest) TearDownSuite(c *C) {
	isvcs.Mgr.Stop()
	dt.FacadeIntegrationTest.TearDownSuite(c)
}

func (dt *DaoTest) TestDao_ValidateEndpoints(t *C) {
	var id string

	// These tests were originally added wtih https://github.com/control-center/serviced/pull/1787,
	// but were skipped for CC 1.1 by https://github.com/control-center/serviced/pull/1811
	t.Skip("this test needs to be restored once we validate endpoints for service add, update, etc")

	// test 1: add tenant with dup ep
	svc := service.Service{
		ID:           "test_tenant",
		Name:         "test_tenant",
		PoolID:       "default",
		Launch:       "auto",
		DeploymentID: "deployment_id",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		},
	}
	err := dt.Dao.AddService(svc, &id)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 2: success
	svc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
	}
	err = dt.Dao.AddService(svc, &id)
	t.Assert(err, IsNil)

	// test 3: update service with dup ep
	svc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 4: add child service with dup ep
	svc2 := service.Service{
		ID:              "test_service_1",
		Name:            "test_service_1",
		ParentServiceID: svc.ID,
		PoolID:          svc.PoolID,
		Launch:          svc.Launch,
		DeploymentID:    svc.DeploymentID,
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		},
	}
	err = dt.Dao.AddService(svc2, &id)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 5: add child success
	svc2.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
	}
	err = dt.Dao.AddService(svc2, &id)
	t.Assert(err, IsNil)

	// test 6: update parent service with dup id on child
	svc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 7: update parent service success
	svc.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_3", Application: "test_ep_3", Purpose: "export"}),
	}
	err = dt.Dao.UpdateService(svc, &unused)
	t.Assert(err, IsNil)

	// test 8: update child service with dup id on parent
	svc2.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_1", Application: "test_ep_1", Purpose: "export"}),
	}
	err = dt.Dao.UpdateService(svc2, &unused)
	t.Check(err, NotNil)
	t.Check(strings.Contains(err.Error(), "found duplicate endpoint name"), Equals, true)

	// test 9: update child service success
	svc2.Endpoints = []service.ServiceEndpoint{
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_2", Application: "test_ep_2", Purpose: "export"}),
		service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{Name: "test_ep_4", Application: "test_ep_4", Purpose: "export"}),
	}
	err = dt.Dao.UpdateService(svc2, &unused)
	t.Assert(err, IsNil)
}

func (dt *DaoTest) TestDao_NewService(t *C) {
	svc := service.Service{}
	err := dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected failure to create service %-v", svc)
		t.Fail()
	}

	svc.ID = "default"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err = dt.Dao.AddService(svc, &id)
	if err != nil {
		t.Errorf("Failure creating service %-v with error: %s", svc, err)
		t.Fail()
	}

	err = dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected error creating redundant service %-v", svc)
		t.Fail()
	}

	svc.ID = ""
	err = dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected error creating service with same name and parent: %-v", svc)
		t.Fail()
	}
}

func (dt *DaoTest) TestDao_UpdateService(t *C) {
	dt.Dao.RemoveService("default", &unused)

	svc, _ := service.NewService()
	svc.ID = "default0"
	svc.Name = "default0"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	svc.Name = "name"
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)
	if err != nil {
		t.Errorf("Failure updating service %-v with error: %s", svc, err)
		t.Fail()
	}

	result := service.Service{}
	dt.Dao.GetService("default0", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	t.Assert(svc.Equals(&result), Equals, true)

	svc, _ = service.NewService()
	svc.ID = "default1"
	svc.Name = "default1"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	svc.Name = "name"
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, Equals, facade.ErrServiceCollision)
}
func (dt *DaoTest) TestDao_UpdateServiceWithConfigFile(t *C) {
	svc, _ := service.NewService()
	svc.ID = "default"
	svc.Name = "default0"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"

	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	confFile := servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"}
	svc.ConfigFiles = map[string]servicedefinition.ConfigFile{"testname": confFile}
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)

	result := service.Service{}
	dt.Dao.GetService("default", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))

	//test update conf file works
	svc, _ = service.NewService()
	svc.ID = "default_conf"
	svc.Name = "default1"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	svc.ImageID = "image_id"
	svc.OriginalConfigs = map[string]servicedefinition.ConfigFile{"testname": confFile}

	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(1, Equals, len(result.OriginalConfigs))
	t.Assert(result.ConfigFiles, DeepEquals, svc.OriginalConfigs)
	t.Assert(result.ConfigFiles, DeepEquals, result.OriginalConfigs)

	confFile2 := servicedefinition.ConfigFile{Content: "Test content 2", Filename: "testname"}
	svc.ConfigFiles = map[string]servicedefinition.ConfigFile{"testname": confFile2}
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(result.ConfigFiles["testname"], DeepEquals, confFile2)
	t.Assert(result.ConfigFiles, Not(DeepEquals), result.OriginalConfigs)

	//now delete service and re-add, it should have previous modified config file
	err = dt.Dao.RemoveService(svc.ID, &unused)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	dt.Dao.GetService("default_conf", &result)
	t.Assert(1, Equals, len(result.ConfigFiles))
	t.Assert(result.ConfigFiles["testname"], DeepEquals, confFile2)
	t.Assert(result.ConfigFiles, Not(DeepEquals), result.OriginalConfigs)

}

func (dt *DaoTest) TestDao_GetService(t *C) {
	svc, _ := service.NewService()
	svc.Name = "testname"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.DeploymentID = "deployment_id"
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	var result service.Service
	err = dt.Dao.GetService(svc.ID, &result)
	t.Assert(err, IsNil)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	if !svc.Equals(&result) {
		t.Errorf("GetService Failed: expected=%+v, actual=%+v", svc, result)
	}
}

func (dt *DaoTest) TestDao_StartService(t *C) {

	s0, _ := service.NewService()
	s0.ID = "0"
	s0.Name = "name0"
	s0.PoolID = "default"
	s0.Launch = "auto"
	s0.DesiredState = int(service.SVCStop)
	s0.DeploymentID = "deployment_id"

	s01, _ := service.NewService()
	s01.ID = "01"
	s01.Name = "name1"
	s01.PoolID = "default"
	s01.Launch = "auto"
	s01.ParentServiceID = "0"
	s01.DesiredState = int(service.SVCStop)
	s01.DeploymentID = "deployment_id"

	s011, _ := service.NewService()
	s011.ID = "011"
	s011.Name = "name2"
	s011.PoolID = "default"
	s011.Launch = "auto"
	s011.ParentServiceID = "01"
	s011.DesiredState = int(service.SVCStop)
	s011.DeploymentID = "deployment_id"

	s02, _ := service.NewService()
	s02.ID = "02"
	s02.Name = "name3"
	s02.DeploymentID = "deployment_id2"

	s02.PoolID = "default"
	s02.Launch = "auto"
	s02.ParentServiceID = "0"
	s02.DesiredState = int(service.SVCStop)

	err := dt.Dao.AddService(*s0, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s01, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s011, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s02, &id)
	t.Assert(err, IsNil)

	// For this test, tenant mounts are valid.
	dt.Dfs().On("VerifyTenantMounts", "0").Return(nil)

	var affected int
	if err := dt.Dao.StartService(dao.ScheduleServiceRequest{[]string{"0"}, true, true, false}, &affected); err != nil {
		t.Fatalf("could not start services: %v", err)
	}

	svc := service.Service{}
	dt.Dao.GetService("0", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 0 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("01", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 01 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("011", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 011 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("02", &svc)
	if svc.DesiredState != int(service.SVCRun) {
		t.Errorf("Service: 02 not requested to run: %+v", svc)
		t.Fail()
	}
}

func (dt *DaoTest) TestDaoAutoAssignIPs(t *C) {
	assignIPsPool := pool.New("assignIPsPoolID")
	fmt.Printf("%s\n", assignIPsPool.ID)
	err := dt.Facade.AddResourcePool(dt.CTX, assignIPsPool)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	ipAddress1 := "192.168.100.10"
	ipAddress2 := "10.50.9.1"

	assignIPsHostIPResources := []host.HostIPResource{}
	oneHostIPResource := host.HostIPResource{}
	oneHostIPResource.HostID = HOSTID
	oneHostIPResource.IPAddress = ipAddress1
	oneHostIPResource.InterfaceName = "eth0"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
	oneHostIPResource.HostID = HOSTID
	oneHostIPResource.IPAddress = ipAddress2
	oneHostIPResource.InterfaceName = "eth1"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost, err := host.Build("", "65535", assignIPsPool.ID, "", []string{}...)
	if err != nil {
		t.Fatalf("Error creating host: %v", err)
	}
	assignIPsHost.ID = HOSTID
	assignIPsHost.IPs = assignIPsHostIPResources
	_, err = dt.Facade.AddHost(dt.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("Failure creating resource host %-v with error: %s", assignIPsHost, err)
	}

	testService := service.Service{
		ID:           "assignIPsServiceID",
		Name:         "testsvc",
		Launch:       "auto",
		PoolID:       assignIPsPool.ID,
		DeploymentID: "deployment_id",
		Endpoints: []service.ServiceEndpoint{
			service.BuildServiceEndpoint(servicedefinition.EndpointDefinition{
				Name:        "AssignIPsEndpointName",
				Protocol:    "tcp",
				PortNumber:  8081,
				Application: "websvc",
				Purpose:     "import",
				AddressConfig: servicedefinition.AddressResourceConfig{
					Port:     8081,
					Protocol: commons.TCP,
				},
			}),
		},
	}

	err = dt.Dao.AddService(testService, &id)
	if err != nil {
		t.Fatalf("Failure creating service %-v with error: %s", testService, err)
	}
	assignmentRequest := addressassignment.AssignmentRequest{testService.ID, "", true, 0, "", ""}
	err = dt.Dao.AssignIPs(assignmentRequest, nil)
	if err != nil {
		t.Errorf("AssignIPs failed: %v", err)
	}

	assignments, err := dt.Facade.GetServiceAddressAssignments(dt.CTX, testService.ID)
	if err != nil {
		t.Errorf("GetServiceAddressAssignments failed: %v", err)
	}
	if len(assignments) != 1 {
		t.Errorf("Expected 1 AddressAssignment but found %d", len(assignments))
	}
}

func (dt *DaoTest) TestDao_NewSnapshot(t *C) {
	// this is technically not a unit test since it depends on the leader
	// starting a watch for snapshot requests and the code here is time
	// dependent waiting for that leader to start the watch
	t.Skip("TODO: fix this test")

	glog.V(0).Infof("TestDao_NewSnapshot started")
	defer glog.V(0).Infof("TestDao_NewSnapshot finished")

	time.Sleep(2 * time.Second) // wait for Leader to start watching for snapshot requests

	service := service.Service{}
	service.ID = "service-without-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	// snapshot should work for services without Snapshot Pause/Resume
	err := dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.ID = "service1-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.ID)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.ID)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.ID = "service2-quiesce"
	dt.Dao.RemoveService(service.ID, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.ID)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.ID)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	req := dao.SnapshotRequest{
		ServiceID: service.ID,
	}
	err = dt.Dao.Snapshot(req, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 1st snapshot with label:%s", id)

	err = dt.Dao.Snapshot(req, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 2nd snapshot with label:%s", id)

	time.Sleep(10 * time.Second)
}
