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
	"github.com/zenoss/serviced/isvcs"
	_ "github.com/zenoss/serviced/volume"
	_ "github.com/zenoss/serviced/volume/btrfs"
	_ "github.com/zenoss/serviced/volume/rsync"
	"github.com/zenoss/serviced/zzk"
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
var err error

func init() {
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	isvcs.Mgr.Wipe()
	controlPlaneDao, err = NewControlSvc("localhost", 9200, addresses, "/tmp", "rsync")
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

func TestNewControlPlaneDao(t *testing.T) {
	if err != nil {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
}

func TestDao_NewResourcePool(t *testing.T) {
	controlPlaneDao.RemoveResourcePool("default", &unused)
	pool := dao.ResourcePool{}
	err := controlPlaneDao.AddResourcePool(pool, &id)
	if err == nil {
		t.Errorf("Expected failure to create resource pool %-v", pool)
		t.Fail()
	}

	pool.Id = "default"
	err = controlPlaneDao.AddResourcePool(pool, &id)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", pool, err)
		t.Fail()
	}

	err = controlPlaneDao.AddResourcePool(pool, &id)
	if err == nil {
		t.Errorf("Expected error creating redundant resource pool %-v", pool)
		t.Fail()
	}
}
func TestDao_UpdateResourcePool(t *testing.T) {
	controlPlaneDao.RemoveResourcePool("default", &unused)

	pool, _ := dao.NewResourcePool("default")
	controlPlaneDao.AddResourcePool(*pool, &id)

	pool.Priority = 1
	pool.CoreLimit = 1
	pool.MemoryLimit = 1
	err := controlPlaneDao.UpdateResourcePool(*pool, &unused)

	if err != nil {
		t.Errorf("Failure updating resource pool %-v with error: %s", pool, err)
		t.Fail()
	}

	result := dao.ResourcePool{}
	controlPlaneDao.GetResourcePool("default", &result)
	result.CreatedAt = pool.CreatedAt
	result.UpdatedAt = pool.UpdatedAt
	if *pool != result {
		t.Errorf("%+v != %+v", *pool, result)
		t.Fail()
	}
}

func TestDao_GetResourcePool(t *testing.T) {
	controlPlaneDao.RemoveResourcePool("default", &unused)
	pool, _ := dao.NewResourcePool("default")
	pool.Priority = 1
	pool.CoreLimit = 1
	pool.MemoryLimit = 1
	controlPlaneDao.AddResourcePool(*pool, &id)

	result := dao.ResourcePool{}
	err := controlPlaneDao.GetResourcePool("default", &result)
	result.CreatedAt = pool.CreatedAt
	result.UpdatedAt = pool.UpdatedAt
	if err == nil {
		if *pool != result {
			t.Errorf("Unexpected ResourcePool: expected=%+v, actual=%+v", pool, result)
		}
	} else {
		t.Errorf("Unexpected Error Retrieving ResourcePool: err=%s", err)
	}
}

func TestDao_GetResourcePools(t *testing.T) {
	controlPlaneDao.RemoveResourcePool("default", &unused)

	pool, _ := dao.NewResourcePool("default")
	pool.Priority = 1
	pool.CoreLimit = 2
	pool.MemoryLimit = 3
	controlPlaneDao.AddResourcePool(*pool, &id)

	var result map[string]*dao.ResourcePool
	err := controlPlaneDao.GetResourcePools(new(dao.EntityRequest), &result)
	if err == nil && len(result) == 1 {
		result["default"].CreatedAt = pool.CreatedAt
		result["default"].UpdatedAt = pool.UpdatedAt
		if *result["default"] != *pool {
			t.Errorf("expected [%+v] actual=%s", *pool, result)
			t.Fail()
		}
	} else {
		t.Errorf("Unexpected Error Retrieving ResourcePools: err=%s", result)
		t.Fail()
	}
}

func TestDao_AddHost(t *testing.T) {
	host := dao.Host{}
	controlPlaneDao.RemoveHost("default", &unused)
	err := controlPlaneDao.AddHost(host, &id)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
		t.Fail()
	}

	host.Id = "default"
	err = controlPlaneDao.AddHost(host, &id)
	if err != nil {
		t.Errorf("Failure creating host %-v with error: %s", host, err)
		t.Fail()
	}

	err = controlPlaneDao.AddHost(host, &id)
	if err == nil {
		t.Errorf("Expected error creating redundant host %-v", host)
		t.Fail()
	}
}
func TestDao_UpdateHost(t *testing.T) {
	controlPlaneDao.RemoveHost("default", &unused)

	host := dao.NewHost()
	host.Id = "default"
	controlPlaneDao.AddHost(*host, &id)

	host.Name = "hostname"
	host.IpAddr = "127.0.0.1"
	err := controlPlaneDao.UpdateHost(*host, &unused)
	if err != nil {
		t.Errorf("Failure updating host %-v with error: %s", host, err)
		t.Fail()
	}

	var result = dao.Host{}
	controlPlaneDao.GetHost("default", &result)
	result.CreatedAt = host.CreatedAt
	result.UpdatedAt = host.UpdatedAt

	if !reflect.DeepEqual(*host, result) {
		t.Errorf("%+v != %+v", result, host)
		t.Fail()
	}
}

func TestDao_GetHost(t *testing.T) {
	controlPlaneDao.RemoveHost("default", &unused)

	host := dao.NewHost()
	host.Id = "default"
	controlPlaneDao.AddHost(*host, &id)

	var result = dao.Host{}
	err := controlPlaneDao.GetHost("default", &result)
	result.CreatedAt = host.CreatedAt
	result.UpdatedAt = host.UpdatedAt
	if err == nil {
		if !reflect.DeepEqual(*host, result) {
			t.Errorf("Unexpected Host: expected=%+v, actual=%+v", host, result)
		}
	} else {
		t.Errorf("Unexpected Error Retrieving Host: err=%s", err)
	}
}

func TestDao_GetHosts(t *testing.T) {
	controlPlaneDao.RemoveHost("0", &unused)
	controlPlaneDao.RemoveHost("1", &unused)
	controlPlaneDao.RemoveHost("default", &unused)

	host := dao.NewHost()
	host.Id = "default"
	host.Name = "hostname"
	host.IpAddr = "127.0.0.1"
	err := controlPlaneDao.AddHost(*host, &id)
	if err == nil {
		t.Errorf("Expected error on host having loopback ip address")
		t.Fail()
	}
	host.IpAddr = "10.0.0.1"
	err = controlPlaneDao.AddHost(*host, &id)
	if err != nil {
		t.Errorf("Unexpected error on adding host: %s", err)
		t.Fail()
	}

	var hosts map[string]*dao.Host
	err = controlPlaneDao.GetHosts(new(dao.EntityRequest), &hosts)
	if err == nil && len(hosts) == 1 {
		hosts["default"].CreatedAt = host.CreatedAt
		hosts["default"].UpdatedAt = host.UpdatedAt
		if !reflect.DeepEqual(*hosts["default"], *host) {
			t.Errorf("expected [%+v] actual=%s", host, hosts)
			t.Fail()
		}
	} else {
		t.Errorf("Unexpected Error Retrieving Hosts: hosts=%+v, err=%s", hosts, err)
		t.Fail()
	}
}

func TestDao_NewService(t *testing.T) {
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

func TestDao_UpdateService(t *testing.T) {
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
	//    as far as I can tell this is a limitation with Go
	result.UpdatedAt = service.UpdatedAt
	result.CreatedAt = service.CreatedAt
	if !service.Equals(&result) {
		t.Errorf("Expected Service %+v, Actual Service %+v", result, *service)
		t.Fail()
	}
}

func TestDao_GetService(t *testing.T) {
	controlPlaneDao.RemoveService("default", &unused)

	service, _ := dao.NewService()
	service.Id = "default"
	controlPlaneDao.AddService(*service, &id)

	var result dao.Service
	err := controlPlaneDao.GetService("default", &result)
	//XXX the time.Time types fail comparison despite being equal...
	//    as far as I can tell this is a limitation with Go
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

func TestDao_GetServices(t *testing.T) {
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
		//    as far as I can tell this is a limitation with Go
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

func TestDao_StartService(t *testing.T) {
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

	controlPlaneDao.StartService("0", &unusedStr)

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

func TestDao_GetTenantId(t *testing.T) {
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

func testDaoHostExists(t *testing.T) {
	found, err := hostExists("blam")
	if found || err != nil {
		t.Errorf("Found %v; error: %v", found, err)
		t.FailNow()
	}

	host := dao.Host{}
	host.Id = "existsTest"
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost("existsTest", &unused)

	found, err = hostExists(id)
	if !found || err != nil {
		t.Errorf("Found %v; error: %v", found, err)
	}
}

func TestDaoValidServiceForStart(t *testing.T) {
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

func TestDaoInvalidServiceForStart(t *testing.T) {
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

func TestDaoGetPoolsIPInfo(t *testing.T) {
	assignIPsPool, _ := dao.NewResourcePool("assignIPsPoolID")
	err = controlPlaneDao.AddResourcePool(*assignIPsPool, &id)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	ipAddress1 := "192.168.100.10"
	ipAddress2 := "10.50.9.1"

	assignIPsHostIPResources := []dao.HostIPResource{}
	oneHostIPResource := dao.HostIPResource{}
	oneHostIPResource.HostId = HOSTID
	oneHostIPResource.IPAddress = ipAddress1
	oneHostIPResource.InterfaceName = "eth0"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
	oneHostIPResource.HostId = HOSTID
	oneHostIPResource.IPAddress = ipAddress2
	oneHostIPResource.InterfaceName = "eth1"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost := dao.Host{}
	assignIPsHost.Id = HOSTID
	assignIPsHost.PoolId = assignIPsPool.Id
	assignIPsHost.IPs = assignIPsHostIPResources
	err = controlPlaneDao.AddHost(assignIPsHost, &id)

	var poolsIpInfo []dao.HostIPResource
	err := controlPlaneDao.GetPoolsIPInfo(assignIPsPool.Id, &poolsIpInfo)
	if err != nil {
		t.Error("GetPoolIps failed")
	}
	if len(poolsIpInfo) != 2 {
		t.Error("Expected number of addresses: ", len(poolsIpInfo))
	}

	if poolsIpInfo[0].IPAddress != ipAddress1 {
		t.Error("Unexpected IP address: ", poolsIpInfo[0].IPAddress)
	}
	if poolsIpInfo[1].IPAddress != ipAddress2 {
		t.Error("Unexpected IP address: ", poolsIpInfo[1].IPAddress)
	}

	defer controlPlaneDao.RemoveResourcePool(assignIPsPool.Id, &unused)
	defer controlPlaneDao.RemoveHost(assignIPsHost.Id, &unused)
}

func TestDaoAutoAssignIPs(t *testing.T) {
	assignIPsPool, _ := dao.NewResourcePool("assignIPsPoolID")
	fmt.Printf("%s\n", assignIPsPool.Id)
	err = controlPlaneDao.AddResourcePool(*assignIPsPool, &id)
	if err != nil {
		t.Errorf("Failure creating resource pool %-v with error: %s", assignIPsPool, err)
		t.Fail()
	}

	ipAddress1 := "192.168.100.10"
	ipAddress2 := "10.50.9.1"

	assignIPsHostIPResources := []dao.HostIPResource{}
	oneHostIPResource := dao.HostIPResource{}
	oneHostIPResource.HostId = HOSTID
	oneHostIPResource.IPAddress = ipAddress1
	oneHostIPResource.InterfaceName = "eth0"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)
	oneHostIPResource.HostId = HOSTID
	oneHostIPResource.IPAddress = ipAddress2
	oneHostIPResource.InterfaceName = "eth1"
	assignIPsHostIPResources = append(assignIPsHostIPResources, oneHostIPResource)

	assignIPsHost := dao.Host{}
	assignIPsHost.Id = HOSTID
	assignIPsHost.PoolId = assignIPsPool.Id
	assignIPsHost.IPs = assignIPsHostIPResources
	err = controlPlaneDao.AddHost(assignIPsHost, &id)

	testService := dao.Service{
		Id:     "assignIPsServiceID",
		PoolId: assignIPsPool.Id,
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
	err := controlPlaneDao.AssignIPs(assignmentRequest, nil)
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
	defer controlPlaneDao.RemoveResourcePool(assignIPsPool.Id, &unused)
	defer controlPlaneDao.RemoveHost(assignIPsHost.Id, &unused)
}

func TestDaoGetHostNoIPs(t *testing.T) {
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	resultHost := dao.Host{}
	err = controlPlaneDao.GetHost(HOSTID, &resultHost)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if len(resultHost.IPs) != 0 {
		t.Errorf("Expected %v IPs, got %v", 0, len(resultHost.IPs))
	}
}

func TestDaoGetHostWithIPs(t *testing.T) {
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	host.IPs = []dao.HostIPResource{dao.HostIPResource{HOSTID, "testip", "ifname"}}
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	resultHost := dao.Host{}
	err = controlPlaneDao.GetHost(HOSTID, &resultHost)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if len(resultHost.IPs) != 1 {
		t.Errorf("Expected %v IPs, got %v", 1, len(resultHost.IPs))
	}
}

func TestRemoveAddressAssignment(t *testing.T) {
	//test removing address when not present
	err = controlPlaneDao.RemoveAddressAssignment("fake", nil)
	if err == nil {
		t.Errorf("Expected error removing address %v", err)
	}
}

func TestAssignAddress(t *testing.T) {
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
	host := dao.Host{}
	host.Id = hostid
	host.IPs = []dao.HostIPResource{dao.HostIPResource{hostid, ip, "ifname"}}
	err = controlPlaneDao.AddHost(host, &id)
	if err != nil {
		t.Errorf("Unexpected error adding host: %v", err)
		return
	}
	defer controlPlaneDao.RemoveHost(hostid, &unused)

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

func TestDao_SnapshotRequest(t *testing.T) {
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

func TestDao_NewSnapshot(t *testing.T) {
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
	err = controlPlaneDao.AddService(service, &id)
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

func TestDao_TestingComplete(t *testing.T) {
	controlPlaneDao.RemoveService("default", &unused)
	controlPlaneDao.RemoveService("0", &unused)
	controlPlaneDao.RemoveService("01", &unused)
	controlPlaneDao.RemoveService("011", &unused)
	controlPlaneDao.RemoveService("02", &unused)

	controlPlaneDao.RemoveResourcePool("default", &unused)

	controlPlaneDao.RemoveHost("default", &unused)
	controlPlaneDao.RemoveHost("0", &unused)
	controlPlaneDao.RemoveHost("1", &unused)
	controlPlaneDao.RemoveHost("existsTest", &unused)
	controlPlaneDao.RemoveHost(HOSTID, &unused)
}
