// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package host

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"

	"testing"
)

var hs HostStore
var ctx datastore.Context

func init() {

	esDriver := elastic.New("localhost", 9200, "twitter")
	err := esDriver.Initialize()
	if err != nil {

		glog.Infof("Error initializing db driver %v", err)
		return
	}
	hs = NewStore()
	ctx = datastore.NewContext(esDriver)
}

func Test_AddHost(t *testing.T) {
	if hs == nil {
		t.Fatalf("Test failed to initialize")
	}

	host := New()

	err := hs.Put(ctx, host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	host.Id = "testid"
	err = hs.Put(ctx, host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}

	//fill host with required values
	host.PoolId = "default_pool"
	err = hs.Put(ctx, host)
	if err == nil {
		t.Errorf("Expected failure to create host %-v", host)
	}
}

/*
func Test_UpdateHost(t *testing.T) {
	controlPlaneDao.RemoveHost("default", &unused)

	host :=NewHost()
	host.Id = "default"
	controlPlaneDao.AddHost(*host, &id)

	host.Name = "hostname"
	host.IpAddr = "127.0.0.1"
	err := controlPlaneDao.UpdateHost(*host, &unused)
	if err != nil {
		t.Errorf("Failure updating host %-v with error: %s", host, err)
		t.Fail()
	}

	var result =Host{}
	controlPlaneDao.GetHost("default", &result)
	result.CreatedAt = host.CreatedAt
	result.UpdatedAt = host.UpdatedAt

	if !reflect.DeepEqual(*host, result) {
		t.Errorf("%+v != %+v", result, host)
		t.Fail()
	}
}

func Test_GetHost(t *testing.T) {
	controlPlaneDao.RemoveHost("default", &unused)

	host :=NewHost()
	host.Id = "default"
	controlPlaneDao.AddHost(*host, &id)

	var result =Host{}
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

func Test_GetHosts(t *testing.T) {
	controlPlaneDao.RemoveHost("0", &unused)
	controlPlaneDao.RemoveHost("1", &unused)
	controlPlaneDao.RemoveHost("default", &unused)

	host :=NewHost()
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
*/
