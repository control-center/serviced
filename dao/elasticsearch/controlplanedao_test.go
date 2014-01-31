/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/
package elasticsearch

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/isvcs"
	"strconv"
	"strings"
	"testing"
)

const (
	HOSTID    = "hostID"
	HOSTIPSID = "HostIPsId"
)

var unused int
var id string
var addresses []string
var controlPlaneDao *ControlPlaneDao
var err error

func init() {
	isvcs.Init()
	isvcs.Mgr.SetVolumesDir("/tmp/serviced-test")
	isvcs.Mgr.Wipe()
	controlPlaneDao, err = NewControlSvc("localhost", 9200, addresses)
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
	if *host != result {
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
		if *host != result {
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
		if *hosts["default"] != *host {
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

	var unusedString string
	controlPlaneDao.StartService("0", &unusedString)

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

func TestDaoGetHostIPsNoHost(t *testing.T) {
	//Test not found
	hostIPs := dao.HostIPs{}
	err = controlPlaneDao.GetHostIPs(HOSTID, &hostIPs)
	if err == nil {
		t.Error("Expected not found error")
	}
	if !strings.HasPrefix(err.Error(), "Host not found for id "+HOSTID) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestDaoGetHostIPsNotFound(t *testing.T) {
	hostIPs := dao.HostIPs{}

	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	err = controlPlaneDao.GetHostIPs(HOSTID, &hostIPs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if hostIPs.Id != "" {
		t.Errorf("Expected uninitialzed id, got %v", hostIPs.Id)
	}
}

func TestDaoGetHostIPsNoIPs(t *testing.T) {
	hostIPs := dao.HostIPs{}
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	//store a HostIPs object
	hostIPs.Id = HOSTIPSID
	hostIPs.HostId = HOSTID
	_, err = newHostIPs(hostIPs.Id, hostIPs)
	defer deleteHostIPs(HOSTIPSID)
	if err != nil {
		t.Errorf("Error creating newHostIPs")
		return
	}

	err = controlPlaneDao.GetHostIPs(HOSTID, &hostIPs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if hostIPs.Id != HOSTIPSID {
		t.Errorf("Expected host ips id %v , got %v", HOSTIPSID, hostIPs.Id)
	}
	if hostIPs.HostId != HOSTID {
		t.Errorf("Expected host id %v, got %v", HOSTID, hostIPs.Id)
	}
	if hostIPs.PoolId != "" {
		t.Errorf("Expected uninitialzed poolid, got %v", hostIPs.PoolId)
	}
	if len(hostIPs.IPs) != 0 {
		t.Errorf("Expected %v IPs, got %v", 0, len(hostIPs.IPs))
	}

}

func TestDaoGetHostIPsWithIPs(t *testing.T) {
	hostIPs := dao.HostIPs{}
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	//store a HostIPs object
	hostIPs.Id = HOSTIPSID
	hostIPs.HostId = HOSTID
	hostIPs.IPs = []dao.HostIPResource{dao.HostIPResource{"valid", "testip", "ifname", []dao.AssignedPort{}}}
	_, err = newHostIPs(hostIPs.Id, hostIPs)
	defer deleteHostIPs(HOSTIPSID)
	if err != nil {
		t.Errorf("Error creating newHostIPs")
		return
	}

	err = controlPlaneDao.GetHostIPs(HOSTID, &hostIPs)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}
	if hostIPs.Id != HOSTIPSID {
		t.Errorf("Expected host ips id %v , got %v", HOSTIPSID, hostIPs.Id)
	}
	if hostIPs.HostId != HOSTID {
		t.Errorf("Expected host id %v, got %v", HOSTID, hostIPs.Id)
	}
	if hostIPs.PoolId != "" {
		t.Errorf("Expected uninitialzed poolid, got %v", hostIPs.PoolId)
	}
	if hostIPs.PoolId != "" {
		t.Errorf("Expected uninitialzed id, got %v", hostIPs.Id)
	}
	if len(hostIPs.IPs) != 1 {
		t.Errorf("Expected %v IPs, got %v", 1, len(hostIPs.IPs))
	}

}

func TestDaoRegisterHostIPsNoHost(t *testing.T) {
	ips := dao.HostIPs{}
	ips.HostId = HOSTID
	ips.IPs = []dao.HostIPResource{dao.HostIPResource{"valid", "testip", "ifname", []dao.AssignedPort{}}}

	err := controlPlaneDao.RegisterHostIPs(ips, &unused)
	if !strings.HasPrefix(err.Error(), "Host not found for id "+HOSTID) {
		t.Errorf("Unexpected error: %v", err)
	}
}

func removeHostIPs(hostId string) {
	hostIPs := dao.HostIPs{}
	err = controlPlaneDao.GetHostIPs(hostId, &hostIPs)
	glog.Infof("Getting HostIPs id: %v: %v, %v", hostIPs.Id, hostIPs, err)
	if err == nil && hostIPs.Id != "" {
		x, err := deleteHostIPs(hostIPs.Id)
		glog.Infof("Deleting HostIPs %v: %v: %v", hostIPs, err, x)
	}
}

func TestDaoRegisterHostIPs(t *testing.T) {
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)

	ips := dao.HostIPs{}
	ips.HostId = HOSTID
	ips.IPs = []dao.HostIPResource{dao.HostIPResource{"valid", "testip", "ifname", []dao.AssignedPort{}}}

	err := controlPlaneDao.RegisterHostIPs(ips, &unused)
	//remove any registered
	defer removeHostIPs(HOSTID)

	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//verify registration
	ips = dao.HostIPs{}
	controlPlaneDao.GetHostIPs(HOSTID, &ips)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ips.IPs) != 1 {
		t.Errorf("Expected 1 HostIPResource, found: %v", len(ips.IPs))
	}
	ipResource := ips.IPs[0]
	if ipResource.State != "valid" {
		t.Errorf("Unexpected state: %v", ipResource.State)
	}
	if ipResource.IPAddress != "testip" {
		t.Errorf("Unexpected IPAddress: %v", ipResource.IPAddress)
	}
	if ipResource.InterfaceName != "ifname" {
		t.Errorf("Unexpected InterfaceName: %v", ipResource.InterfaceName)
	}

}
func TestDaoRegisterHostIPsMerge(t *testing.T) {
	//Add host to test scenario where host exists but no IP resource registered
	host := dao.Host{}
	host.Id = HOSTID
	err = controlPlaneDao.AddHost(host, &id)
	defer controlPlaneDao.RemoveHost(HOSTID, &unused)

	ips := dao.HostIPs{}
	ips.HostId = HOSTID
	ips.IPs = []dao.HostIPResource{
		dao.HostIPResource{"valid", "testip", "ifname", []dao.AssignedPort{}},
		dao.HostIPResource{"deleted", "testip1", "ifname", []dao.AssignedPort{}},
	}

	err := controlPlaneDao.RegisterHostIPs(ips, &unused)
	//remove any registered
	defer removeHostIPs(HOSTID)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	ips.IPs = []dao.HostIPResource{
		dao.HostIPResource{"valid", "testip2", "ifname2", []dao.AssignedPort{}},
		dao.HostIPResource{"valid", "testip1", "ifname", []dao.AssignedPort{}},
	}
	err = controlPlaneDao.RegisterHostIPs(ips, &unused)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//verify registration
	ips = dao.HostIPs{}
	controlPlaneDao.GetHostIPs(HOSTID, &ips)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if len(ips.IPs) != 3 {
		t.Errorf("Expected 3 HostIPResources, found: %v", len(ips.IPs))
	}
	for _, ipResource := range ips.IPs {
		addr := ipResource.IPAddress
		switch addr {
		case "testip":
			{
				if ipResource.State != "deleted" {
					t.Errorf("Unexpected state: %v", ipResource.State)
				}

				if ipResource.InterfaceName != "ifname" {
					t.Errorf("Unexpected InterfaceName: %v", ipResource.InterfaceName)
				}
			}
		case "testip1":
			{
				if ipResource.State != "valid" {
					t.Errorf("Unexpected state: %v", ipResource.State)
				}

				if ipResource.InterfaceName != "ifname" {
					t.Errorf("Unexpected InterfaceName: %v", ipResource.InterfaceName)
				}
			}
		case "testip2":
			{
				if ipResource.State != "valid" {
					t.Errorf("Unexpected state: %v", ipResource.State)
				}

				if ipResource.InterfaceName != "ifname2" {
					t.Errorf("Unexpected InterfaceName: %v", ipResource.InterfaceName)
				}
			}
		default:
			{
				t.Errorf("Unexpected IP resource %v", ipResource)
			}
		}

	}
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

	deleteHostIPs(HOSTIPSID)
}
