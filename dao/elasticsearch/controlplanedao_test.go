/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/
package elasticsearch

import "github.com/zenoss/serviced/dao"
import "testing"

var	controlPlaneDao, err = NewControlPlaneDao("localhost", 9200)

func TestNewControlPlaneDao(t *testing.T) {
  if err == nil {
    controlPlaneDao.DeleteResourcePool( "default")
  } else {
		t.Errorf("Did not expect error: %s", err)
		t.Fail()
	}
}

func TestDao_NewResourcePool(t *testing.T) {
  resourcePool := dao.ResourcePool { }
  controlPlaneDao.DeleteResourcePool( "default")
  err := controlPlaneDao.NewResourcePool( &resourcePool)
  if err == nil {
    t.Errorf("Expected failure to create resource pool %-v", resourcePool)
    t.Fail()
  }

  resourcePool.Id = "default"
  err = controlPlaneDao.NewResourcePool( &resourcePool)
  if err != nil {
    t.Errorf("Failure creating resource pool %-v with error: %s", resourcePool, err)
    t.Fail()
  }

  err = controlPlaneDao.NewResourcePool( &resourcePool)
  if err == nil {
    t.Errorf("Expected error creating redundant resource pool %-v", resourcePool)
    t.Fail()
  }
}
func TestDao_UpdateResourcePool(t *testing.T) {
  controlPlaneDao.DeleteResourcePool( "default")

  resourcePool := dao.ResourcePool { "default", "", 0, 0, 0}
  controlPlaneDao.NewResourcePool( &resourcePool)

  resourcePool = dao.ResourcePool { "default", "", 1, 1, 1}
  err := controlPlaneDao.UpdateResourcePool( &resourcePool)

  if err != nil {
    t.Errorf("Failure updating resource pool %-v with error: %s", resourcePool, err)
    t.Fail()
  }

  actualResourcePool, _ := controlPlaneDao.GetResourcePool( "default")
  if resourcePool != actualResourcePool {
    t.Errorf("%+v != %+v", actualResourcePool, resourcePool)
    t.Fail()
  }
}

func TestDao_GetResourcePool(t *testing.T) {
  controlPlaneDao.DeleteResourcePool( "default")
  resourcePool := dao.ResourcePool {"default", "", 1, 1, 1}
  err = controlPlaneDao.NewResourcePool( &resourcePool)

  result, err := controlPlaneDao.GetResourcePool( "default")
  if err == nil {
    if resourcePool != result {
      t.Errorf( "Unexpected ResourcePool: expected=%+v, actual=%+v", resourcePool, result)
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving ResourcePool: err=%s", err)
  }
}

func TestDao_GetResourcePools(t *testing.T) {
  controlPlaneDao.DeleteResourcePool( "default")

  one := dao.ResourcePool{"default", "", 1, 2, 3}
  err := controlPlaneDao.NewResourcePool( &one)
  result, err := controlPlaneDao.GetResourcePools()
  if err == nil {
    if len(result) != 1 || result[0] != one {
      t.Errorf( "expected [%+v] actual=%s", one, result)
      t.Fail()
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving ResourcePools: err=%s", result)
    t.Fail()
  }
}

func TestDao_NewHost(t *testing.T) {
  host := dao.Host { }
  controlPlaneDao.DeleteHost( "default")
  err := controlPlaneDao.NewHost( &host)
  if err == nil {
    t.Errorf("Expected failure to create host %-v", host)
    t.Fail()
  }

  host.Id = "default"
  err = controlPlaneDao.NewHost( &host)
  if err != nil {
    t.Errorf("Failure creating host %-v with error: %s", host, err)
    t.Fail()
  }

  err = controlPlaneDao.NewHost( &host)
  if err == nil {
    t.Errorf("Expected error creating redundant host %-v", host)
    t.Fail()
  }
}
func TestDao_UpdateHost(t *testing.T) {
  controlPlaneDao.DeleteHost( "default")

  host := dao.Host { "default", "", "", "", 0, 0, ""}
  controlPlaneDao.NewHost( &host)

  host = dao.Host { "default", "hostname", "", "127.0.0.1", 0, 0, ""}
  err := controlPlaneDao.UpdateHost( &host)
  if err != nil {
    t.Errorf("Failure updating host %-v with error: %s", host, err)
    t.Fail()
  }

  actualHost, _ := controlPlaneDao.GetHost( "default")
  if host != actualHost {
    t.Errorf("%+v != %+v", actualHost, host)
    t.Fail()
  }
}

func TestDao_GetHost(t *testing.T) {
  controlPlaneDao.DeleteHost( "default")
  host := dao.Host { "default", "", "", "", 0, 0, ""}
  err = controlPlaneDao.NewHost( &host)

  result, err := controlPlaneDao.GetHost( "default")
  if err == nil {
    if host != result {
      t.Errorf( "Unexpected Host: expected=%+v, actual=%+v", host, result)
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving Host: err=%s", err)
  }
}

func TestDao_GetHosts(t *testing.T) {
  controlPlaneDao.DeleteHost( "default")

  one := dao.Host { "default", "hostname", "", "127.0.1.1", 0, 0, ""}
  err := controlPlaneDao.NewHost( &one)
  result, err := controlPlaneDao.GetHosts()
  if err == nil {
    if len(result) != 1 || result[0] != one {
      t.Errorf( "expected [%+v] actual=%s", one, result)
      t.Fail()
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving Hosts: err=%s", result)
    t.Fail()
  }
}

func TestDao_NewService(t *testing.T) {
  service := dao.Service { }
  controlPlaneDao.DeleteService( "default")
  err := controlPlaneDao.NewService( &service)
  if err == nil {
    t.Errorf("Expected failure to create service %-v", service)
    t.Fail()
  }

  service.Id = "default"
  err = controlPlaneDao.NewService( &service)
  if err != nil {
    t.Errorf("Failure creating service %-v with error: %s", service, err)
    t.Fail()
  }

  err = controlPlaneDao.NewService( &service)
  if err == nil {
    t.Errorf("Expected error creating redundant service %-v", service)
    t.Fail()
  }
}

func TestDao_UpdateService(t *testing.T) {
  controlPlaneDao.DeleteService( "default")

  service := dao.Service { "default", "", "", "", 0, "", "", ""}
  controlPlaneDao.NewService( &service)

  service = dao.Service { "default", "servicename", "", "", 0, "", "", ""}
  err := controlPlaneDao.UpdateService( &service)
  if err != nil {
    t.Errorf("Failure updating service %-v with error: %s", service, err)
    t.Fail()
  }

  actualService, _ := controlPlaneDao.GetService( "default")
  if service != actualService {
    t.Errorf("%+v != %+v", actualService, service)
    t.Fail()
  }
}

func TestDao_GetService(t *testing.T) {
  controlPlaneDao.DeleteService( "default")
  service := dao.Service { "default", "", "", "", 0, "", "", ""}
  err = controlPlaneDao.NewService( &service)

  result, err := controlPlaneDao.GetService( "default")
  if err == nil {
    if service != result {
      t.Errorf( "Unexpected Service: expected=%+v, actual=%+v", service, result)
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving Service: err=%s", err)
  }
}

func TestDao_GetServices(t *testing.T) {
  controlPlaneDao.DeleteService( "default")
  one := dao.Service { "default", "ServiceName", "", "Service Description", 0, "", "", ""}
  err := controlPlaneDao.NewService( &one)
  result, err := controlPlaneDao.GetServices()
  if err == nil {
    if len(result) != 1 || result[0] != one {
      t.Errorf( "expected [%+v] actual=%s", one, result)
      t.Fail()
    }
  } else {
    t.Errorf( "Unexpected Error Retrieving Services: err=%s", result)
    t.Fail()
  }
}
