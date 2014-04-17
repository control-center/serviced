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
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/commons"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/facade"
	"github.com/zenoss/serviced/isvcs"
	_ "github.com/zenoss/serviced/volume"
	_ "github.com/zenoss/serviced/volume/btrfs"
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/zzk"
	. "gopkg.in/check.v1"

	"reflect"
	"strconv"
	"strings"
	"testing"
	"time"
)

const (
	HOSTID    = "hostID"
	HOSTIPSID = "HostIPsId"
)

var unused int
var unusedStr string
var id string
var addresses []string
var controlPlaneDao *ControlPlaneDao

//var err error

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

var _ = Suite(&daoTest{facade.FacadeTest{DomainPath: "../../domain"}})

type daoTest struct {
	facade.FacadeTest
}

func (dt *daoTest) SetUpSuite(c *C) {
	dt.Port = 9202
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	isvcs.Mgr.Wipe()
	if err := isvcs.Mgr.Start(); err != nil {
		c.Fatalf("Could not start es container: %s", err)
	}
	dt.MappingsFile = "controlplane.json"
	dt.FacadeTest.SetUpSuite(c)
	//	func NewControlSvc(hostName string, port int, facade *facade.Facade, zookeepers []string, varpath, vfs string) (*ControlPlaneDao, error) {

	var err error
	controlPlaneDao, err = NewControlSvc("localhost", int(dt.Port), dt.Facade, addresses, "/tmp", "rsync")
	if err != nil {
		glog.Fatalf("Could not start es container: %s", err)
	} else {
		for i := 0; i < 10; i += 1 {
			id := strconv.Itoa(i)
			controlPlaneDao.RemoveService(id, &unused)
		}
		for i := 100; i < 110; i += 1 {
			id := strconv.Itoa(i)
			controlPlaneDao.RemoveService(id, &unused)
		}
	}
}

func (dt *daoTest) SetUpTest(c *C) {
	dt.FacadeTest.SetUpTest(c)
	if err := dt.Facade.CreateDefaultPool(dt.CTX); err != nil {
		c.Fatalf("could not create default pool:", err)
	}

	// create the account credentials
	if err := createSystemUser(controlPlaneDao); err != nil {
		c.Fatalf("could not create systemuser:", err)
	}

	if err := dt.Facade.CreateDefaultPool(dt.CTX); err != nil {
		c.Fatalf("could not create default pool:", err)
	}
}

func (dt *daoTest) TestDao_NewService(t *C) {
	service := dao.Service{}
	controlPlaneDao.RemoveService("default", &unused)
	err := controlPlaneDao.AddService(service, &id)
	if err == nil {
		t.Errorf("Expected failure to create service %-v", service)
		t.Fail()
	}

	service.Id = "default"
	err = controlPlaneDao.AddService(service, &id)
	if err != nil {
		t.Errorf("Failure creating service %-v with error: %s", service, err)
		t.Fail()
	}

	err = controlPlaneDao.AddService(service, &id)
	if err == nil {
		t.Errorf("Expected error creating redundant service %-v", service)
		t.Fail()
	}
}

func (dt *daoTest) TestDao_UpdateService(t *C) {
	controlPlaneDao.RemoveService("default", &unused)

	service, _ := dao.NewService()
	service.Id = "default"
	controlPlaneDao.AddService(*service, &id)

	service.Name = "name"
	err := controlPlaneDao.UpdateService(*service, &unused)
	if err != nil {
		t.Errorf("Failure updating service %-v with error: %s", service, err)
		t.Fail()
	}

	result := dao.Service{}
	controlPlaneDao.GetService("default", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = service.UpdatedAt
	result.CreatedAt = service.CreatedAt
	if !service.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *service)
		t.Fail()
	}
}

func (dt *daoTest) TestDao_GetService(t *C) {
	controlPlaneDao.RemoveService("default", &unused)

	service, _ := dao.NewService()
	service.Id = "default"
	controlPlaneDao.AddService(*service, &id)

	var result dao.Service
	err := controlPlaneDao.GetService("default", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//	  as far as I can tell this is a limitation with Go
	result.UpdatedAt = service.UpdatedAt
	result.CreatedAt = service.CreatedAt
	if err == nil {
		if !service.Equals(&result) {
			t.Errorf("GetService Failed: expected=%+v, actual=%+v", service, result)
		}
	} else {
		t.Errorf("Unexpected Error Retrieving Service: err=%s", err)
	}
}

func (dt *daoTest) TestDao_GetServices(t *C) {
	controlPlaneDao.RemoveService("0", &unused)
	controlPlaneDao.RemoveService("1", &unused)
	controlPlaneDao.RemoveService("2", &unused)
	controlPlaneDao.RemoveService("3", &unused)
	controlPlaneDao.RemoveService("4", &unused)
	controlPlaneDao.RemoveService("01", &unused)
	controlPlaneDao.RemoveService("011", &unused)
	controlPlaneDao.RemoveService("02", &unused)
	controlPlaneDao.RemoveService("default", &unused)

	service, _ := dao.NewService()
	service.Id = "default"
	service.Name = "name"
	service.Description = "description"
	service.Instances = 0
	controlPlaneDao.AddService(*service, &id)

	var result []*dao.Service
	err := controlPlaneDao.GetServices(new(dao.EntityRequest), &result)
	if err == nil && len(result) == 1 {
		//XXX the time.Time types fail comparison despite being equal...
		//	  as far as I can tell this is a limitation with Go
		result[0].UpdatedAt = service.UpdatedAt
		result[0].CreatedAt = service.CreatedAt
		if !result[0].Equals(service) {
			t.Errorf("expected [%+v] actual=%+v", *service, result)
			t.Fail()
		}
	} else {
		t.Errorf("Error Retrieving Services: err=%s, result=%s", err, result)
		t.Fail()
	}
}

func (dt *daoTest) TestDao_StartService(t *C) {
	controlPlaneDao.RemoveService("0", &unused)
	controlPlaneDao.RemoveService("01", &unused)
	controlPlaneDao.RemoveService("011", &unused)
	controlPlaneDao.RemoveService("02", &unused)

	s0, _ := dao.NewService()
	s0.Id = "0"
	s0.DesiredState = dao.SVC_STOP

	s01, _ := dao.NewService()
	s01.Id = "01"
	s01.ParentServiceId = "0"
	s01.DesiredState = dao.SVC_STOP

	s011, _ := dao.NewService()
	s011.Id = "011"
	s011.ParentServiceId = "01"
	s011.DesiredState = dao.SVC_STOP

	s02, _ := dao.NewService()
	s02.Id = "02"
	s02.ParentServiceId = "0"
	s02.DesiredState = dao.SVC_STOP

	controlPlaneDao.AddService(*s0, &id)
	controlPlaneDao.AddService(*s01, &id)
	controlPlaneDao.AddService(*s011, &id)
	controlPlaneDao.AddService(*s02, &id)

	if err:= controlPlaneDao.StartService("0", &unusedStr); err !=nil{
		t.Fatalf("could not start services: %v", err)
	}

	service := dao.Service{}
	controlPlaneDao.GetService("0", &service)
	if service.DesiredState != dao.SVC_RUN {
		t.Errorf("Service: 0 not requested to run: %+v", service)
		t.Fail()
	}

	controlPlaneDao.GetService("01", &service)
	if service.DesiredState != dao.SVC_RUN {
		t.Errorf("Service: 01 not requested to run: %+v", service)
		t.Fail()
	}

	controlPlaneDao.GetService("011", &service)
	if service.DesiredState != dao.SVC_RUN {
		t.Errorf("Service: 011 not requested to run: %+v", service)
		t.Fail()
	}

	controlPlaneDao.GetService("02", &service)
	if service.DesiredState != dao.SVC_RUN {
		t.Errorf("Service: 02 not requested to run: %+v", service)
		t.Fail()
	}
}

func (dt *daoTest) TestDao_GetTenantId(t *C) {
	controlPlaneDao.RemoveService("0", &unused)
	controlPlaneDao.RemoveService("01", &unused)
	controlPlaneDao.RemoveService("011", &unused)

	var err error
	var tenantId string
	err = controlPlaneDao.GetTenantId("0", &tenantId)
	if err == nil {
		t.Errorf("Expected failure for getting tenantId for 0")
		t.Fail()
	}

	s0, _ := dao.NewService()
	s0.Id = "0"

	s01, _ := dao.NewService()
	s01.Id = "01"
	s01.ParentServiceId = "0"

	s011, _ := dao.NewService()
	s011.Id = "011"
	s011.ParentServiceId = "01"

	controlPlaneDao.AddService(*s0, &id)
	controlPlaneDao.AddService(*s01, &id)
	controlPlaneDao.AddService(*s011, &id)

	tenantId = ""
	err = controlPlaneDao.GetTenantId("0", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = controlPlaneDao.GetTenantId("01", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}

	tenantId = ""
	err = controlPlaneDao.GetTenantId("011", &tenantId)
	if err != nil || tenantId != "0" {
		t.Errorf("Failure getting tenantId for 0, err=%s, tenantId=%s", err, tenantId)
		t.Fail()
	}
}

func (dt *daoTest) TestDaoValidServiceForStart(t *C) {
	testService := dao.Service{
		Id: "TestDaoValidServiceForStart_ServiceId",
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Name:        "TestDaoValidServiceForStart_EndpointName",
				Protocol:    "tcp",
				PortNumber:  8081,
				Application: "websvc",
				Purpose:     "import",
			},
		},
	}
	err := controlPlaneDao.validateServicesForStarting(testService, nil)
	if err != nil {
		t.Error("Services failed validation for starting: ", err)
	}
}

func (dt *daoTest) TestDaoInvalidServiceForStart(t *C) {
	testService := dao.Service{
		Id: "TestDaoInvalidServiceForStart_ServiceId",
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Name:        "TestDaoInvalidServiceForStart_EndpointName",
				Protocol:    "tcp",
				PortNumber:  8081,
				Application: "websvc",
				Purpose:     "import",
				AddressConfig: dao.AddressResourceConfig{
					Port:     8081,
					Protocol: commons.TCP,
				},
			},
		},
	}
	err := controlPlaneDao.validateServicesForStarting(testService, nil)
	if err == nil {
		t.Error("Services should have failed validation for starting...")
	}
}

func (dt *daoTest) TestDaoAutoAssignIPs(t *C) {
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

	assignIPsHost := host.New()
	assignIPsHost.ID = HOSTID
	assignIPsHost.PoolID = assignIPsPool.ID
	assignIPsHost.IPs = assignIPsHostIPResources
	err = dt.Facade.AddHost(dt.CTX, assignIPsHost)

	testService := dao.Service{
		Id:     "assignIPsServiceID",
		PoolId: assignIPsPool.ID,
		Endpoints: []dao.ServiceEndpoint{
			dao.ServiceEndpoint{
				Name:        "AssignIPsEndpointName",
				Protocol:    "tcp",
				PortNumber:  8081,
				Application: "websvc",
				Purpose:     "import",
				AddressConfig: dao.AddressResourceConfig{
					Port:     8081,
					Protocol: commons.TCP,
				},
			},
		},
	}

	err = controlPlaneDao.AddService(testService, &id)
	if err != nil {
		t.Fatalf("Failure creating service %-v with error: %s", testService, err)
	}

	assignmentRequest := dao.AssignmentRequest{testService.Id, "", true}
	err = controlPlaneDao.AssignIPs(assignmentRequest, nil)
	if err != nil {
		t.Error("AssignIPs failed: %v", err)
	}

	assignments := []dao.AddressAssignment{}
	err = controlPlaneDao.GetServiceAddressAssignments(testService.Id, &assignments)
	if err != nil {
		t.Error("GetServiceAddressAssignments failed: %v", err)
	}
	if len(assignments) != 1 {
		t.Error("Expected 1 AddressAssignment but found ", len(assignments))
	}

	defer controlPlaneDao.RemoveService(testService.Id, &unused)
	defer dt.Facade.RemoveResourcePool(dt.CTX, assignIPsPool.ID)
	defer dt.Facade.RemoveHost(dt.CTX, assignIPsHost.ID)
}

func (dt *daoTest) TestRemoveAddressAssignment(t *C) {
	//test removing address when not present
	err := controlPlaneDao.RemoveAddressAssignment("fake", nil)
	if err == nil {
		t.Errorf("Expected error removing address %v", err)
	}
}

func (dt *daoTest) TestAssignAddress(t *C) {
	aa := dao.AddressAssignment{}
	aid := ""
	err := controlPlaneDao.AssignAddress(aa, &aid)
	if err == nil {
		t.Error("Expected error")
	}

	//set up host with IP
	hostid := "TestHost"
	ip := "testip"
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
	service, _ := dao.NewService()
	ep := dao.ServiceEndpoint{}
	ep.Name = endpoint
	ep.AddressConfig = dao.AddressResourceConfig{8080, commons.TCP}
	service.Endpoints = []dao.ServiceEndpoint{ep}
	controlPlaneDao.AddService(*service, &serviceId)
	if err != nil {
		t.Errorf("Unexpected error adding service: %v", err)
		return
	}
	defer controlPlaneDao.RemoveService(serviceId, &unused)

	//test for bad service id
	aa = dao.AddressAssignment{"", "static", hostid, "", ip, 100, "blamsvc", endpoint}
	aid = ""
	err = controlPlaneDao.AssignAddress(aa, &aid)
	if err == nil || "Found 0 Services with id blamsvc" != err.Error() {
		t.Errorf("Expected error adding address %v", err)
	}

	//test for bad endpoint id
	aa = dao.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, "blam"}
	aid = ""
	err = controlPlaneDao.AssignAddress(aa, &aid)
	if err == nil || !strings.HasPrefix(err.Error(), "Endpoint blam not found on service") {
		t.Errorf("Expected error adding address %v", err)
	}

	// Valid assignment
	aa = dao.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, endpoint}
	aid = ""
	err = controlPlaneDao.AssignAddress(aa, &aid)
	if err != nil {
		t.Errorf("Unexpected error adding address %v", err)
		return
	}

	// try to reassign; should fail
	aa = dao.AddressAssignment{"", "static", hostid, "", ip, 100, serviceId, endpoint}
	other_aid := ""
	err = controlPlaneDao.AssignAddress(aa, &other_aid)
	if err == nil || "Address Assignment already exists" != err.Error() {
		t.Errorf("Expected error adding address %v", err)
	}

	//test removing address
	err = controlPlaneDao.RemoveAddressAssignment(aid, nil)
	if err != nil {
		t.Errorf("Unexpected error removing address %v", err)
	}
}

func (dt *daoTest) TestDao_ServiceTemplate(t *C) {
	glog.V(0).Infof("TestDao_AddServiceTemplate started")
	defer glog.V(0).Infof("TestDao_AddServiceTemplate finished")

	var unused int
	var templateId string
	template := dao.ServiceTemplate{
		Id:          "",
		Name:        "test_template",
		Description: "test template",
	}

	if e := controlPlaneDao.AddServiceTemplate(template, &templateId); e != nil {
		t.Fatalf("Failure adding service template %+v with error: %s", template, e)
	}

	var templates map[string]*dao.ServiceTemplate
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
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
	template.Id = templateId
	template.Description = "test_template_modified"
	if e := controlPlaneDao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
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
	if e := controlPlaneDao.RemoveServiceTemplate(templateId, &unused); e != nil {
		t.Fatalf("Failure removing service template with error: %s", e)
	}
	time.Sleep(1 * time.Second) // race condition. :(
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	if len(templates) != 0 {
		t.Fatalf("Expected zero templates. Found %d", len(templates))
	}
	if e := controlPlaneDao.UpdateServiceTemplate(template, &unused); e != nil {
		t.Fatalf("Failure updating service template %+v with error: %s", template, e)
	}
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
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

func (dt *daoTest) TestDao_SnapshotRequest(t *C) {
	glog.V(0).Infof("TestDao_SnapshotRequest started")
	defer glog.V(0).Infof("TestDao_SnapshotRequest finished")

	zkDao := &zzk.ZkDao{[]string{"127.0.0.1:2181"}}

	srExpected := dao.SnapshotRequest{Id: "request13",
		ServiceId: "12345", SnapshotLabel: "foo", SnapshotError: "bar"}
	if err := zkDao.AddSnapshotRequest(&srExpected); err != nil {
		t.Fatalf("Failure adding snapshot request %+v with error: %s", srExpected, err)
	}
	glog.V(0).Infof("adding duplicate snapshot request - expecting failure on next line like: zk: node already exists")
	if err := zkDao.AddSnapshotRequest(&srExpected); err == nil {
		t.Fatalf("Should have seen failure adding duplicate snapshot request %+v", srExpected)
	}

	srResult := dao.SnapshotRequest{}
	if _, err := zkDao.LoadSnapshotRequest(srExpected.Id, &srResult); err != nil {
		t.Fatalf("Failure loading snapshot request %+v with error: %s", srResult, err)
	}
	if !reflect.DeepEqual(srExpected, srResult) {
		t.Fatalf("Failure comparing snapshot request expected:%+v result:%+v", srExpected, srResult)
	}

	srExpected.ServiceId = "67890"
	srExpected.SnapshotLabel = "bin"
	srExpected.SnapshotError = "baz"
	if err := zkDao.UpdateSnapshotRequest(&srExpected); err != nil {
		t.Fatalf("Failure updating snapshot request %+v with error: %s", srResult, err)
	}

	if _, err := zkDao.LoadSnapshotRequest(srExpected.Id, &srResult); err != nil {
		t.Fatalf("Failure loading snapshot request %+v with error: %s", srResult, err)
	}
	if !reflect.DeepEqual(srExpected, srResult) {
		t.Fatalf("Failure comparing snapshot request expected:%+v result:%+v", srExpected, srResult)
	}

	if err := zkDao.RemoveSnapshotRequest(srExpected.Id); err != nil {
		t.Fatalf("Failure removing snapshot request %+v with error: %s", srExpected, err)
	}
	if err := zkDao.RemoveSnapshotRequest(srExpected.Id); err != nil {
		t.Fatalf("Failure removing non-existant snapshot request %+v", srExpected)
	}
}

func (dt *daoTest) TestDao_NewSnapshot(t *C) {
	// this is technically not a unit test since it depends on the leader
	// starting a watch for snapshot requests and the code here is time
	// dependent waiting for that leader to start the watch
	return

	glog.V(0).Infof("TestDao_NewSnapshot started")
	defer glog.V(0).Infof("TestDao_NewSnapshot finished")

	time.Sleep(2 * time.Second) // wait for Leader to start watching for snapshot requests

	service := dao.Service{}
	service.Id = "service-without-quiesce"
	controlPlaneDao.RemoveService(service.Id, &unused)
	// snapshot should work for services without Snapshot Pause/Resume
	err := controlPlaneDao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.Id = "service1-quiesce"
	controlPlaneDao.RemoveService(service.Id, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.Id)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.Id)
	err = controlPlaneDao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	service.Id = "service2-quiesce"
	controlPlaneDao.RemoveService(service.Id, &unused)
	service.Snapshot.Pause = fmt.Sprintf("STATE=paused echo %s quiesce $STATE", service.Id)
	service.Snapshot.Resume = fmt.Sprintf("STATE=resumed echo %s quiesce $STATE", service.Id)
	err = controlPlaneDao.AddService(service, &id)
	if err != nil {
		t.Fatalf("Failure creating service %+v with error: %s", service, err)
	}

	err = controlPlaneDao.Snapshot(service.Id, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 1st snapshot with label:%s", id)

	err = controlPlaneDao.Snapshot(service.Id, &id)
	if err != nil {
		t.Fatalf("Failure creating snapshot for service %+v with error: %s", service, err)
	}
	if id == "" {
		t.Fatalf("Failure creating snapshot for service %+v - label is empty", service)
	}
	glog.V(0).Infof("successfully created 2nd snapshot with label:%s", id)

	time.Sleep(10 * time.Second)
}

func (dt *daoTest) TestDao_TestingComplete(t *C) {
	controlPlaneDao.RemoveService("default", &unused)
	controlPlaneDao.RemoveService("0", &unused)
	controlPlaneDao.RemoveService("01", &unused)
	controlPlaneDao.RemoveService("011", &unused)
	controlPlaneDao.RemoveService("02", &unused)
}

func (dt *daoTest) TestUser_UserOperations(t *C) {
	user := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := controlPlaneDao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}

	newUser := dao.User{}
	err = controlPlaneDao.GetUser("Pepe", &newUser)
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
	err = controlPlaneDao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}
}

func (dt *daoTest) TestUser_ValidateCredentials(t *C) {
	user := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	id := "Pepe"
	err := controlPlaneDao.AddUser(user, &id)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	var isValid bool
	attemptUser := dao.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	err = controlPlaneDao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}

	if !isValid {
		t.Fatalf("Unable to authenticate user credentials")
	}

	unused := 0
	err = controlPlaneDao.RemoveUser("Pepe", &unused)
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}

	// update the user
	user.Password = "pepe2"
	err = controlPlaneDao.UpdateUser(user, &unused)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	attemptUser.Password = "Pepe2"
	// make sure we can validate against the updated credentials
	err = controlPlaneDao.ValidateCredentials(attemptUser, &isValid)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}
}
