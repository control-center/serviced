/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/
package elasticsearch

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons"
	coordclient "github.com/zenoss/serviced/coordinator/client"
	coordzk "github.com/zenoss/serviced/coordinator/client/zookeeper"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/addressassignment"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicetemplate"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/isvcs"
	_ "github.com/zenoss/serviced/volume"
	_ "github.com/zenoss/serviced/volume/btrfs"
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/zzk"
	. "gopkg.in/check.v1"
)

const (
	HOSTID    = "hostID"
	HOSTIPSID = "HostIPsId"
)

var unused int
var unusedStr string
var id string
var addresses []string

var err error

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

//Instantiate the gocheck suite. Initialize the DaoTest and the embedded FacadeTest
var _ = Suite(&DaoTest{})

//DaoTest gocheck test type for setting up isvcs and other resources needed by tests
type DaoTest struct {
	facade.FacadeTest
	Dao *ControlPlaneDao
}

//SetUpSuite is run before the tests to ensure elastic, zookeeper etc. are running.
func (dt *DaoTest) SetUpSuite(c *C) {
	dt.Port = 9202
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	isvcs.Mgr.Wipe()
	if err := isvcs.Mgr.Start(); err != nil {
		c.Fatalf("Could not start es container: %s", err)
	}
	dt.MappingsFile = "controlplane.json"
	dt.FacadeTest.SetUpSuite(c)

	dsn := coordzk.NewDSN([]string{"127.0.0.1:2181"}, time.Second*15).String()
	glog.Infof("zookeeper dsn: %s", dsn)
	zclient, err := coordclient.New("zookeeper", dsn, "", nil)
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	}
	dt.Dao, err = NewControlSvc("localhost", int(dt.Port), dt.Facade, zclient, "/tmp", "rsync", "localhost:5000")
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
	dt.FacadeTest.SetUpTest(c)
	//DAO tests expect default pool and system user

	if err := dt.Facade.CreateDefaultPool(dt.CTX); err != nil {
		c.Fatalf("could not create default pool:", err)
	}

	// create the account credentials
	if err := createSystemUser(dt.Dao); err != nil {
		c.Fatalf("could not create systemuser:", err)
	}
}

//TearDownSuite stops all isvcs
func (dt *DaoTest) TearDownSuite(c *C) {
	isvcs.Mgr.Stop()
	dt.FacadeTest.TearDownSuite(c)
}

func (dt *DaoTest) TestDao_NewService(t *C) {
	svc := service.Service{}
	err := dt.Dao.AddService(svc, &id)
	if err == nil {
		t.Errorf("Expected failure to create service %-v", svc)
		t.Fail()
	}

	svc.Id = "default"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"
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
}

func (dt *DaoTest) TestDao_UpdateService(t *C) {
	dt.Dao.RemoveService("default", &unused)

	svc, _ := service.NewService()
	svc.Id = "default"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	svc.Name = "name"
	err = dt.Dao.UpdateService(*svc, &unused)
	if err != nil {
		t.Errorf("Failure updating service %-v with error: %s", svc, err)
		t.Fail()
	}

	result := service.Service{}
	dt.Dao.GetService("default", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	if !svc.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *svc)
		t.Fail()
	}
}
func (dt *DaoTest) TestDao_UpdateServiceWithConfigFile(t *C) {
	svc, _ := service.NewService()
	svc.Id = "default"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"

	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)
	//Conf file update shouldn't occur because original service didn't have conf files
	confFile := servicedefinition.ConfigFile{Content: "Test content", Filename: "testname"}
	svc.ConfigFiles = map[string]servicedefinition.ConfigFile{"testname": confFile}
	err = dt.Dao.UpdateService(*svc, &unused)
	t.Assert(err, IsNil)

	result := service.Service{}
	dt.Dao.GetService("default", &result)
	t.Assert(0, Equals, len(result.ConfigFiles))

	//test update conf file works
	svc, _ = service.NewService()
	svc.Id = "default_conf"
	svc.Name = "default"
	svc.PoolID = "default"
	svc.Launch = "auto"
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
	err = dt.Dao.RemoveService(svc.Id, &unused)
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
	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	var result service.Service
	err = dt.Dao.GetService(svc.Id, &result)
	t.Assert(err, IsNil)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = svc.UpdatedAt
	result.CreatedAt = svc.CreatedAt
	if !svc.Equals(&result) {
		t.Errorf("GetService Failed: expected=%+v, actual=%+v", svc, result)
	}
}

func (dt *DaoTest) TestDao_GetServices(t *C) {
	svc, _ := service.NewService()
	svc.Id = "default"
	svc.Name = "name"
	svc.PoolID = "default"
	svc.Launch = "auto"
	svc.Description = "description"
	svc.Instances = 0

	err := dt.Dao.AddService(*svc, &id)
	t.Assert(err, IsNil)

	var result []*service.Service
	err = dt.Dao.GetServices(new(dao.EntityRequest), &result)
	t.Assert(err, IsNil)
	t.Assert(len(result), Equals, 1)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result[0].UpdatedAt = svc.UpdatedAt
	result[0].CreatedAt = svc.CreatedAt
	if !result[0].Equals(svc) {
		t.Errorf("expected [%+v] actual=%+v", *svc, result)
		t.Fail()
	}
}

func (dt *DaoTest) TestStoppingParentStopsChildren(t *C) {
	svc := service.Service{
		Id:             "ParentServiceID",
		Name:           "ParentService",
		Startup:        "/usr/bin/ping -c localhost",
		Description:    "Ping a remote host a fixed number of times",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1},
		ImageID:        "test/pinger",
		PoolID:         "default",
		DesiredState:   1,
		Launch:         "auto",
		Endpoints:      []service.ServiceEndpoint{},
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
	childService1 := service.Service{
		Id:              "childService1",
		Name:            "childservice1",
		Launch:          "auto",
		PoolID:          "default",
		Startup:         "/bin/sh -c \"while true; do echo hello world 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	childService2 := service.Service{
		Id:              "childService2",
		Name:            "childservice2",
		Launch:          "auto",
		PoolID:          "default",
		Startup:         "/bin/sh -c \"while true; do echo date 10; sleep 3; done\"",
		ParentServiceID: "ParentServiceID",
	}
	// add a service with a subservice
	id := "ParentServiceID"
	var err error
	if err = dt.Dao.AddService(svc, &id); err != nil {
		glog.Fatalf("Failed Loading Parent Service Service: %+v, %s", svc, err)
	}

	childService1Id := "childService1"
	childService2Id := "childService2"
	if err = dt.Dao.AddService(childService1, &childService1Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 1: %+v, %s", childService1, err)
	}
	if err = dt.Dao.AddService(childService2, &childService2Id); err != nil {
		glog.Fatalf("Failed Loading Child Service 2: %+v, %s", childService2, err)
	}
	var unused int
	var stringUnused string
	// start the service
	if err = dt.Dao.StartService(id, &stringUnused); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// stop the parent
	if err = dt.Dao.StopService(id, &unused); err != nil {
		glog.Fatalf("Unable to stop parent service: %+v, %s", svc, err)
	}
	// verify the children have all stopped
	query := fmt.Sprintf("ParentServiceID:%s AND NOT Launch:manual", id)
	var services []*service.Service
	err = dt.Dao.GetServices(query, &services)
	for _, subService := range services {
		if subService.DesiredState == 1 && subService.ParentServiceID == id {
			t.Errorf("Was expecting child services to be stopped %v", subService)
		}
	}

}

func (dt *DaoTest) TestDao_StartService(t *C) {

	s0, _ := service.NewService()
	s0.Id = "0"
	s0.Name = "name"
	s0.PoolID = "default"
	s0.Launch = "auto"
	s0.DesiredState = service.SVCStop

	s01, _ := service.NewService()
	s01.Id = "01"
	s01.Name = "name"
	s01.PoolID = "default"
	s01.Launch = "auto"
	s01.ParentServiceID = "0"
	s01.DesiredState = service.SVCStop

	s011, _ := service.NewService()
	s011.Id = "011"
	s011.Name = "name"
	s011.PoolID = "default"
	s011.Launch = "auto"
	s011.ParentServiceID = "01"
	s011.DesiredState = service.SVCStop

	s02, _ := service.NewService()
	s02.Id = "02"
	s02.Name = "name"
	s02.PoolID = "default"
	s02.Launch = "auto"
	s02.ParentServiceID = "0"
	s02.DesiredState = service.SVCStop

	err := dt.Dao.AddService(*s0, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s01, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s011, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s02, &id)
	t.Assert(err, IsNil)

	if err := dt.Dao.StartService("0", &unusedStr); err != nil {
		t.Fatalf("could not start services: %v", err)
	}

	svc := service.Service{}
	dt.Dao.GetService("0", &svc)
	if svc.DesiredState != service.SVCRun {
		t.Errorf("Service: 0 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("01", &svc)
	if svc.DesiredState != service.SVCRun {
		t.Errorf("Service: 01 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("011", &svc)
	if svc.DesiredState != service.SVCRun {
		t.Errorf("Service: 011 not requested to run: %+v", svc)
		t.Fail()
	}

	dt.Dao.GetService("02", &svc)
	if svc.DesiredState != service.SVCRun {
		t.Errorf("Service: 02 not requested to run: %+v", svc)
		t.Fail()
	}
}

func (dt *DaoTest) TestDao_GetTenantId(t *C) {
	var err error
	var tenantId string
	err = dt.Dao.GetTenantId("0", &tenantId)
	if err == nil {
		t.Errorf("Expected failure for getting tenantId for 0")
		t.Fail()
	}

	s0, _ := service.NewService()
	s0.Name = "name"
	s0.PoolID = "default"
	s0.Launch = "auto"
	s0.Id = "0"

	s01, _ := service.NewService()
	s01.Id = "01"
	s01.ParentServiceID = "0"
	s01.Name = "name"
	s01.PoolID = "default"
	s01.Launch = "auto"

	s011, _ := service.NewService()
	s011.Id = "011"
	s011.ParentServiceID = "01"
	s011.Name = "name"
	s011.PoolID = "default"
	s011.Launch = "auto"

	err = dt.Dao.AddService(*s0, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s01, &id)
	t.Assert(err, IsNil)
	err = dt.Dao.AddService(*s011, &id)
	t.Assert(err, IsNil)

	tenantId = ""
	err = dt.Dao.GetTenantId("0", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = dt.Dao.GetTenantId("01", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = dt.Dao.GetTenantId("011", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}
}

func (dt *DaoTest) TestDaoValidServiceForStart(t *C) {
	testService := service.Service{
		Id: "TestDaoValidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "TestDaoValidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
				},
			},
		},
	}
	err := dt.Dao.validateServicesForStarting(&testService, nil)
	if err != nil {
		t.Error("Services failed validation for starting: ", err)
	}
}

func (dt *DaoTest) TestDaoInvalidServiceForStart(t *C) {
	testService := service.Service{
		Id: "TestDaoInvalidServiceForStart_ServiceID",
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "TestDaoInvalidServiceForStart_EndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				},
			},
		},
	}
	err := dt.Dao.validateServicesForStarting(&testService, nil)
	if err == nil {
		t.Error("Services should have failed validation for starting...")
	}
}

func (dt *DaoTest) TestRenameImageID(t *C) {
	imageId, err := dt.Dao.renameImageID("quay.io/zenossinc/daily-zenoss5-core:5.0.0_123", "X")
	if err != nil {
		t.Errorf("unexpected failure renamingImageID: %s", err)
		t.FailNow()
	}
	expected := "localhost:5000/X/daily-zenoss5-core"
	if imageId != expected {
		t.Errorf("expected image '%s' got '%s'", expected, imageId)
		t.FailNow()
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

	assignIPsHost, err := host.Build("", assignIPsPool.ID, []string{}...)
	if err != nil {
		t.Fatalf("Error creating host: %v", err)
	}
	assignIPsHost.ID = HOSTID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = dt.Facade.AddHost(dt.CTX, assignIPsHost)
	if err != nil {
		t.Fatalf("Failure creating resource host %-v with error: %s", assignIPsHost, err)
	}

	testService := service.Service{
		Id:     "assignIPsServiceID",
		Name:   "testsvc",
		Launch: "auto",
		PoolID: assignIPsPool.ID,
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				EndpointDefinition: servicedefinition.EndpointDefinition{
					Name:        "AssignIPsEndpointName",
					Protocol:    "tcp",
					PortNumber:  8081,
					Application: "websvc",
					Purpose:     "import",
					AddressConfig: servicedefinition.AddressResourceConfig{
						Port:     8081,
						Protocol: commons.TCP,
					},
				},
			},
		},
	}

	err = dt.Dao.AddService(testService, &id)
	if err != nil {
		t.Fatalf("Failure creating service %-v with error: %s", testService, err)
	}
	assignmentRequest := dao.AssignmentRequest{testService.Id, "", true}
	err = dt.Dao.AssignIPs(assignmentRequest, nil)
	if err != nil {
		t.Errorf("AssignIPs failed: %v", err)
	}

	assignments := []*addressassignment.AddressAssignment{}
	err = dt.Dao.GetServiceAddressAssignments(testService.Id, &assignments)
	if err != nil {
		t.Error("GetServiceAddressAssignments failed: %v", err)
	}
	if len(assignments) != 1 {
		t.Error("Expected 1 AddressAssignment but found ", len(assignments))
	}

}

func (dt *DaoTest) TestRemoveAddressAssignment(t *C) {
	//test removing address when not present
	err := dt.Dao.RemoveAddressAssignment("fake", nil)
	if err == nil {
		t.Errorf("Expected error removing address %v", err)
	}
}

func (dt *DaoTest) TestAssignAddress(t *C) {
	aa := addressassignment.AddressAssignment{}
	aid := ""
	err := dt.Dao.AssignAddress(aa, &aid)
	if err == nil {
		t.Error("Expected error")
	}

	//set up host with IP
	hostid := "TestHost"
	ip := "10.0.1.5"
	endpoint := "default"
	serviceId := ""
	h, err := host.Build("", "default", []string{}...)
	t.Assert(err, IsNil)
	h.ID = hostid
	h.IPs = []host.HostIPResource{host.HostIPResource{hostid, ip, "ifname"}}
	err = dt.Facade.AddHost(dt.CTX, h)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
		return
	}
	defer dt.Facade.RemoveHost(dt.CTX, hostid)

	//set up service with endpoint
	svc, _ := service.NewService()
	svc.Name = "name"
	svc.Launch = "auto"
	svc.PoolID = "default"
	ep := service.ServiceEndpoint{}
	ep.Name = endpoint
	ep.AddressConfig = servicedefinition.AddressResourceConfig{8080, commons.TCP}
	svc.Endpoints = []service.ServiceEndpoint{ep}
	err = dt.Dao.AddService(*svc, &serviceId)
	t.Assert(err, IsNil)

	//test for bad service id
	aa = addressassignment.AddressAssignment{"", "static", hostid, "", ip, 100, "blamsvc", endpoint}
	aid = ""
	err = dt.Dao.AssignAddress(aa, &aid)
	if err == nil || "No such entity {kind:service, id:blamsvc}" != err.Error() {
		t.Errorf("Expected error adding address %v", err)
	}

	//test for bad endpoint id
	aa = addressassignment.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, "blam"}
	aid = ""
	err = dt.Dao.AssignAddress(aa, &aid)
	if err == nil || !strings.HasPrefix(err.Error(), "Endpoint blam not found on service") {
		t.Errorf("Expected error adding address %v", err)
	}

	// Valid assignment
	aa = addressassignment.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, endpoint}
	aid = ""
	err = dt.Dao.AssignAddress(aa, &aid)
	if err != nil {
		t.Errorf("Unexpected error adding address %v", err)
		return
	}

	// try to reassign; should fail
	aa = addressassignment.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, endpoint}
	other_aid := ""
	err = dt.Dao.AssignAddress(aa, &other_aid)
	if err == nil || "Address Assignment already exists" != err.Error() {
		t.Errorf("Expected error adding address %v", err)
	}

	//test removing address
	err = dt.Dao.RemoveAddressAssignment(aid, nil)
	if err != nil {
		t.Errorf("Unexpected error removing address %v", err)
	}
}

func (dt *DaoTest) TestDao_ServiceTemplate(t *C) {
	glog.V(0).Infof("TestDao_AddServiceTemplate started")
	defer glog.V(0).Infof("TestDao_AddServiceTemplate finished")

	var (
		unused     int
		templateId string
		templates  map[string]*servicetemplate.ServiceTemplate
	)

	// Clean up old templates...
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	for id, _ := range templates {
		if e := dt.Dao.RemoveServiceTemplate(id, &unused); e != nil {
			t.Fatalf("Failure removing service template %s with error: %s", id, e)
		}
	}

	template := servicetemplate.ServiceTemplate{
		ID:          "",
		Name:        "test_template",
		Description: "test template",
	}

	if e := dt.Dao.AddServiceTemplate(template, &templateId); e != nil {
		t.Fatalf("Failure adding service template %+v with error: %s", template, e)
	}

	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if templates[templateId] == nil {
		t.Fatalf("Expected to find template that was added (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	template.ID = templateId
	template.Description = "test_template_modified"
	if e := dt.Dao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if templates[templateId] == nil {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
	if templates[templateId].Description != "test_template_modified" {
		t.Fatalf("Expected template to be modified. It hasn't changed!")
	}
	if e := dt.Dao.RemoveServiceTemplate(templateId, &unused); e != nil {
		t.Fatalf("Failure removing service template with error: %s", e)
	}
	time.Sleep(1 * time.Second) // race condition. :(
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 0 {
		t.Fatalf("Expected zero templates. Found %d", len(templates))
	}
	if e := dt.Dao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected 1 template. Found %d", len(templates))
	}
	if templates[templateId] == nil {
		t.Fatalf("Expected to find template that was updated (%s), but did not.", templateId)
	}
	if templates[templateId].Name != "test_template" {
		t.Fatalf("Expected to find test_template. Found %s", templates[templateId].Name)
	}
}

func (dt *DaoTest) TestDao_SnapshotRequest(t *C) {
	t.Skip("TODO: fix this test")

	glog.V(0).Infof("TestDao_SnapshotRequest started")
	defer glog.V(0).Infof("TestDao_SnapshotRequest finished")

	dsn := coordzk.DSN{
		Servers: []string{"127.0.0.1:2181"},
		Timeout: time.Second * 10,
	}
	cclient, _ := coordclient.New("zookeeper", dsn.String(), "", nil)
	zkDao := zzk.NewZkDao(cclient)

	srExpected := dao.SnapshotRequest{
		Id:            "request13",
		ServiceID:     "12345",
		SnapshotLabel: "foo",
		SnapshotError: "bar",
	}
	if err := zkDao.AddSnapshotRequest(&srExpected); err != nil {
		t.Fatalf("Failure adding snapshot request %+v with error: %s", srExpected, err)
	}
	glog.V(0).Infof("adding duplicate snapshot request - expecting failure on next line like: node already exists")
	if err := zkDao.AddSnapshotRequest(&srExpected); err == nil {
		t.Fatalf("Should have seen failure adding duplicate snapshot request %+v", srExpected)
	}

	srResult := dao.SnapshotRequest{}
	if err := zkDao.LoadSnapshotRequest(srExpected.Id, &srResult); err != nil {
		t.Fatalf("Failure loading snapshot request %+v with error: %s", srResult, err)
	}
	if !reflect.DeepEqual(srExpected, srResult) {
		t.Fatalf("Failure comparing snapshot request expected:%+v result:%+v", srExpected, srResult)
	}

	srExpected.ServiceID = "67890"
	srExpected.SnapshotLabel = "bin"
	srExpected.SnapshotError = "baz"
	if err := zkDao.UpdateSnapshotRequest(&srExpected); err != nil {
		t.Fatalf("Failure updating snapshot request %+v with error: %s", srResult, err)
	}

	if err := zkDao.LoadSnapshotRequest(srExpected.Id, &srResult); err != nil {
		t.Fatalf("Failure loading snapshot request %+v with error: %s", srResult, err)
	}
	if !reflect.DeepEqual(srExpected, srResult) {
		t.Fatalf("Failure comparing snapshot request expected:%+v result:%+v", srExpected, srResult)
	}

	if err := zkDao.RemoveSnapshotRequest(srExpected.Id); err != nil {
		t.Fatalf("Failure removing snapshot request %+v with error: %s", srExpected, err)
	}
	if err := zkDao.RemoveSnapshotRequest(srExpected.Id); err == nil {
		t.Fatalf("Failure removing non-existant snapshot request expected %+v", srExpected)
	}
}

func (dt *DaoTest) TestDao_NewSnapshot(t *C) {
	t.Skip("TODO: fix this test")
	// this is technically not a unit test since it depends on the leader
	// starting a watch for snapshot requests and the code here is time
	// dependent waiting for that leader to start the watch
	return

	glog.V(0).Infof("TestDao_NewSnapshot started")
	defer glog.V(0).Infof("TestDao_NewSnapshot finished")

	time.Sleep(2 * time.Second) // wait for Leader to start watching for snapshot requests

	service := service.Service{}
	service.Id = "service-without-quiesce"
	dt.Dao.RemoveService(service.Id, &unused)
	// snapshot should work for services without Snapshot Pause/Resume
	err := dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.Id = "service1-quiesce"
	dt.Dao.RemoveService(service.Id, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.Id)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.Id)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.Id = "service2-quiesce"
	dt.Dao.RemoveService(service.Id, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.Id)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.Id)
	err = dt.Dao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	err = dt.Dao.Snapshot(service.Id, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 1st snapshot with label:%s", id)

	err = dt.Dao.Snapshot(service.Id, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 2nd snapshot with label:%s", id)

	time.Sleep(10 * time.Second)
}

func (dt *DaoTest) TestUser_UserOperations(t *C) {
	user := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := dt.Dao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}

	newUser := dao.User{}
	err = dt.Dao.GetUser("Pepe", &newUser)
	if err != nil {
		t.Fatalf("Failure getting user %s", err)
	}

	// make sure they are the same user
	if newUser.Name != user.Name {
		t.Fatalf("Retrieved an unexpected user %v", newUser)
	}

	// make sure the password was hashed
	if newUser.Password == "Pepe" {
		t.Fatalf("Did not hash the password %+v", user)
	}

	unused := 0
	err = dt.Dao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}
}

func (dt *DaoTest) TestUser_ValidateCredentials(t *C) {
	user := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := dt.Dao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	var isValid bool
	attemptUser := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	err = dt.Dao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}

	if !isValid {
		t.Fatalf("Unable to authenticate user credentials")
	}

	unused := 0
	err = dt.Dao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}

	// update the user
	user.Password = "pepe2"
	err = dt.Dao.UpdateUser(user, &unused)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	attemptUser.Password = "Pepe2"
	// make sure we can validate against the updated credentials
	err = dt.Dao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}
}
