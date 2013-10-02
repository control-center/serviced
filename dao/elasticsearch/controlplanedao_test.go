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
import "time"


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

  //exists function broken
  //_, err = controlPlaneDao.NewResourcePool( &resourcePool)
  //if err == nil {
  //  t.Errorf("Expected error creating redundant resource pool %-v", resourcePool)
  //  t.Fail()
  //}
}
func TestDao_UpdateResourcePool(t *testing.T) {
  controlPlaneDao.DeleteResourcePool( "default")

  resourcePool := dao.ResourcePool { "default", "", 0, 0, 0}
  controlPlaneDao.UpdateResourcePool( &resourcePool)

  //TODO expect an ERROR, "default" doesn't exist (elasticgo exists is broken...)
  //if err != nil {
  //  t.Errorf("Expected error updating non-existant resource pool %-v", resourcePool)
  //  t.Fail()
  //}

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
  if err != nil {
    t.Errorf( "expected no error adding document: error=%s", err)
    t.Fail()
  }

  //XXX on very first execution, this test fails without the following settle time
  time.Sleep( 250 * time.Millisecond)
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
