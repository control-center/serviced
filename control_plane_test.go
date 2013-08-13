/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package serviced

import (
	"database/sql"
	_ "github.com/ziutek/mymysql/godrv"
	"log"
	"net"
	"net/http"
	"net/rpc"
	"os/exec"
	"testing"
)

var (
	server  ControlPlane
	client  *ControlClient
	unused  int
	tempdir string
)

var (
	database_name     = "cp_test"
	database_user     = "root"
	database_password = ""
)

func connectionString() string {
	return database_name + "/" + database_user + "/" + database_password
}

func cleanTestDB(t *testing.T) {
	conn, err := sql.Open("mymysql", "/"+database_user+"/")
	defer conn.Close()
	_, err = conn.Exec("DROP DATABASE IF EXISTS `" + database_name + "`")
	if err != nil {
		log.Fatal("Could not drop test database:", err)
	}
	_, err = conn.Exec("CREATE DATABASE `" + database_name + "`")
	if err != nil {
		log.Fatal("Could not create test database: ", err)
	}
	cmd := exec.Command("mysql", "-u", "root", database_name, "-e", "source database.sql")
	output, err := cmd.Output()
	if err != nil {
		log.Fatal("Problem sourcing schema", err)
	}
	log.Print(string(output))
}

func setup(t *testing.T) {

	cleanTestDB(t)
	server, err := NewControlSvc(connectionString())

	// register the server API
	rpc.RegisterName("ControlPlane", server)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatalf("net.Listen tcp :0 %v", err)
	}
	go http.Serve(l, nil) // start the server
	log.Printf("Test Server started on %s", l.Addr().String())

	// setup the client
	client, err = NewControlClient(l.Addr().String())
	if err != nil {
		log.Fatalf("Coult not start client %v", err)
	}
	log.Printf("Client started: %v", client)
}

func TestControlAPI(t *testing.T) {
	setup(t)

	request := EntityRequest{}
	var hosts map[string]*Host = nil

	err := client.GetHosts(request, &hosts)
	if err != nil {
		log.Fatalf("Could not get hosts", err)
	}
	host, err := CurrentContextAsHost()
	log.Printf("Got a currentContextAsHost()\n")
	if err != nil {
		t.Fatal("Could not get currentContextAsHost", err)
	}
	err = client.AddHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not add host", err)
	}

	host.Name = "foo"
	err = client.UpdateHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not update host", err)
	} else {
		log.Print("update of host is OK")
	}
	err = client.GetHosts(request, &hosts)
	if err != nil {
		t.Fatal("Error getting updated hosts.", err)
	}
	if hosts[host.Id].Name != "foo" {
		t.Fatal("Expected host to be named foo.", err)
	}

	err = client.RemoveHost(host.Id, &unused)
	if err != nil {
		t.Fatal("Could not remove host.", err)
	}
	hosts = nil
	err = client.GetHosts(request, &hosts)
	if err != nil {
		t.Fatal("Error getting updated hosts.", err)
	}
	_, exists := hosts[host.Id]
	if exists {
		t.Fatal("Host was not removed.", err)
	}

	var services []*Service
	err = client.GetServices(request, &services)
	if err != nil {
		t.Fatal("Error getting services.", err)
	}
	if len(services) != 0 {
		t.Fatal("Expecting 0 services")
	}

	/*
		service, err := NewService()
		if err != nil {
			t.Fatal("Error creating new service.")
		}
		service.Name = "helloworld"
		err = client.AddService(*service, &unused)
		if err != nil {
			t.Fatal("Could not add service.")
		}
		services = nil
		err = client.GetServices(request, &services)
		if err != nil {
			t.Fatal("Error getting services.")
		}
		if len(services) != 1 {
			t.Fatal("Expecting 1 service, got ", len(services))
		}
		if services[0].Id != service.Id {
			t.Fatalf("Created service %s but got back %s", services[0].Id, service.Id)
		}

		service.Name = "Roger"
		err = client.UpdateService(*service, &unused)
		if err != nil {
			t.Fatalf("Could not save service.")
		}
		err = client.GetServices(request, &services)
		if err != nil {
			t.Fatal("Error getting services.")
		}
		if len(services) != 1 {
			t.Fatal("Expecting 1 service, got ", len(services))
		}
		if services[0].Id != service.Id {
			t.Fatalf("Created service %s but got back %s", services[0].Id, service.Id)
		}

		err = client.RemoveService(service.Id, &unused)
		if err != nil {
			t.Fatal("error removing service.")
		}
		services = nil
		err = client.GetServices(request, &services)
		if err != nil {
			t.Fatal("Error getting services.")
		}
		if len(services) != 0 {
			t.Fatal("Expecting 0 service, got ", len(services))
		}
	*/

	services = nil
	err = client.GetServicesForHost("dasdfasdf", &services)
	log.Printf("Got %d services", len(services))
	if err == nil {
		t.Fatal("Expected error looking for non-existent service.")
	}

	var pools map[string]*ResourcePool = nil
	err = client.GetResourcePools(request, &pools)
	if err != nil {
		t.Fatal("Problem getting empty resource pool list.", err)
	}

	pool, _ := NewResourcePool()
	pool.Name = "unit_test_pool"
	err = client.AddResourcePool(*pool, &unused)
	if err != nil {
		t.Fatal("Problem adding resource pool", err)
	}

	err = client.RemoveResourcePool(pool.Id, &unused)
	if err != nil {
		t.Fatal("Problem removing resource pool", err)
	}

	pools = nil
	err = client.GetResourcePools(request, &pools)
	if err != nil {
		t.Fatal("Problem getting empty resource pool list.")
	}
	if len(pools) != 0 {
		t.Fatal("Expected 0 pools: ", len(pools))
	}
}

func TestServiceStart(t *testing.T) {

	cleanTestDB(t)
	host, err := CurrentContextAsHost()
	log.Printf("Got a currentContextAsHost()\n")
	if err != nil {
		t.Fatal("Could not get currentContextAsHost", err)
	}
	err = client.AddHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not add host", err)
	}

	pool, _ := NewResourcePool()
	pool.Name = "default"
	err = client.AddResourcePool(*pool, &unused)
	if err != nil {
		t.Fatal("Problem adding resource pool", err)
	}
	err = client.AddHostToResourcePool(PoolHost{HostId: host.Id, PoolId: pool.Id}, &unused)
	if err != nil {
		t.Fatal("Problem adding host to resource pool", err)
	}

	// add a new service
	service, _ := NewService()
	service.Name = "My test service!"
	service.PoolId = pool.Id
	service.Startup = "/bin/sh -c \"while true; do echo hello world; sleep 1; done\""
	err = client.AddService(*service, &unused)
	if err != nil {
		t.Fatal("Could not add service: ", err)
	}

	// start the service
	var hostId string
	err = client.StartService(service.Id, &hostId)
	if err != nil {
		t.Fatal("Got error starting service: ", err)
	}

	var services []*Service
	// get the services for a host
	err = client.GetServicesForHost(host.Id, &services)
	if err != nil {
		t.Fatal("Could not get services for host: ", err)
	}
	log.Printf("Got %d services for %s", len(services), host.Id)
}
