// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package facade

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/datastore/elastic"
	"github.com/zenoss/serviced/domain/host"

	"testing"
)

var (
	tf  *Facade
	ctx datastore.Context
)

func init() {

	esDriver, err := elastic.InitElasticTestMappings("controlplane", map[string]string{"host": "../host/host_mapping.json"})
	if err != nil {
		glog.Infof("Error initializing db driver %v", err)
		return
	}

	ctx = datastore.NewContext(esDriver)
	hs := host.NewStore()
	tf = New(hs)
}

func Test_HostCRUD(t *testing.T) {

	if tf == nil {
		t.Fatalf("Test failed to initialize")
	}

	testid := "facadetestid"
	defer tf.RemoveHost(ctx, testid)

	//fill host with required values
	h, err := host.Build("", "pool-id", []string{}...)
	h.Id = "facadetestid"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = tf.AddHost(ctx, h)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	//Test re-add fails
	err = tf.AddHost(ctx, h)
	if err == nil {
		t.Errorf("Expected already exists error: %v", err)
	}

	h2, err := tf.GetHost(ctx, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	host.HostEquals(t, h, h2)

	//Test update
	h.Memory = 1024
	err = tf.UpdateHost(ctx, h)
	h2, err = tf.GetHost(ctx, testid)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	host.HostEquals(t, h, h2)

	//test delete
	err = tf.RemoveHost(ctx, testid)
	h2, err = tf.GetHost(ctx, testid)
	if err != nil && !datastore.IsErrNoSuchEntity(err) {
		t.Errorf("Unexpected error: %v", err)
	}

}

/*
func Test_GetHosts(t *testing.T) {
	if tf == nil {
		t.Fatalf("Test failed to initialize")
	}
	hid1 := "gethosts1"
	hid2 := "gethosts2"

	defer tf.RemoveHost(ctx, hid1)
	defer tf.RemoveHost(ctx, hid2)

	host, err := host.Build("", "pool-id", []string{}...)
	host.Id = "Test_GetHosts1"
	if err != nil {
		t.Fatalf("Unexpected error building host: %v", err)
	}
	err = hs.Put(ctx, host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	time.Sleep(1000 * time.Millisecond)
	hosts, err := hs.GetUpTo(ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 1 {
		t.Errorf("Expected %v results, got %v :%v", 1, len(hosts), hosts)
	}

	host.Id = "Test_GetHosts2"
	err = hs.Put(ctx, host)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	time.Sleep(1000 * time.Millisecond)
	hosts, err = hs.GetUpTo(ctx, 1000)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	} else if len(hosts) != 2 {
		t.Errorf("Expected %v results, got %v :%v", 2, len(hosts), hosts)
	}

}
*/
