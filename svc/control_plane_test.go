/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package svc

import (
	"database/sql"
	serviced "github.com/zenoss/serviced"
	client "github.com/zenoss/serviced/client"
	_ "github.com/ziutek/mymysql/godrv"
	"github.com/zenoss/glog"
	"net"
	"net/http"
	"net/rpc"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
)

var (
	server  serviced.ControlPlane
	lclient *client.ControlClient
	unused  int
	tempdir string
)

var connInfo *serviced.DatabaseConnectionInfo

func init() {
	var err error
	conStr := os.Getenv("CP_TEST_DB")
	if len(conStr) == 0 {
		conStr = "mysql://root@127.0.0.1:3306/cp_test"
	}
	connInfo, err = serviced.ParseDatabaseUri(conStr)
	if err != nil {
		panic(err)
	}
}

func cleanTestDB(t *testing.T) {
	db := connInfo.Database
	connInfo.Database = ""
	defer func() {
		connInfo.Database = db
	}()
	conn, err := sql.Open("mymysql", serviced.ToMymysqlConnectionString(connInfo))
	defer conn.Close()
	_, err = conn.Exec("DROP DATABASE IF EXISTS `" + db + "`")
	if err != nil {
		glog.Fatalf("Could not drop test database: %s", err)
	}
	_, err = conn.Exec("CREATE DATABASE `" + db + "`")
	if err != nil {
		glog.Fatalf("Could not create test database: %s", err)
	}
	cmdParts := make([]string, 0)
	cmdParts = append(cmdParts, []string{"-h", connInfo.Host}...)
	cmdParts = append(cmdParts, []string{"-P", strconv.Itoa(connInfo.Port)}...)
	cmdParts = append(cmdParts, []string{"-u", connInfo.User}...)
	if len(connInfo.Password) > 0 {
		cmdParts = append(cmdParts, []string{"--password", connInfo.Password}...)
	}
	cmdParts = append(cmdParts, db)
	cmdParts = append(cmdParts, []string{"-e", "source database.sql"}...)
	cmd := exec.Command("mysql", cmdParts...)
	glog.Infoln(strings.Join(cmd.Args, " "))
	output, err := cmd.Output()
	if err != nil {
		glog.Fatalf("Problem sourcing schema: %s", err, string(output))
	}
	glog.Infoln(string(output))
}

func setup(t *testing.T) {

	cleanTestDB(t)
	glog.Infof("Starting server with: %s", serviced.ToMymysqlConnectionString(connInfo))
	server, err := NewControlSvc("mysql://root@127.0.0.1:3306/cp_test", []string{})

	// register the server API
	rpc.RegisterName("ControlPlane", server)
	rpc.HandleHTTP()

	l, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		glog.Fatalf("net.Listen tcp :0 %v", err)
	}
	go http.Serve(l, nil) // start the server
	glog.Infof("Test Server started on %s", l.Addr().String())

	// setup the client
	lclient, err = client.NewControlClient(l.Addr().String())
	if err != nil {
		glog.Fatalf("Coult not start client %v", err)
	}
	glog.Infof("Client started: %v", lclient)
}

func TestControlAPI(t *testing.T) {
	setup(t)

	var err error
	var request serviced.EntityRequest

	var pools map[string]*serviced.ResourcePool = nil
	err = lclient.GetResourcePools(request, &pools)
	if err != nil {
		t.Fatal("Problem getting empty resource pool list.", err)
	}

	pool, _ := serviced.NewResourcePool("unit_test_pool")
	err = lclient.AddResourcePool(*pool, &unused)
	if err != nil {
		t.Fatal("Problem adding resource pool", err)
	}

	err = lclient.RemoveResourcePool(pool.Id, &unused)
	if err != nil {
		t.Fatal("Problem removing resource pool", err)
	}

	pools = nil
	err = lclient.GetResourcePools(request, &pools)
	if err != nil {
		t.Fatal("Problem getting empty resource pool list.")
	}
	if len(pools) != 1 {
		t.Fatal("Expected 1 pools, got ", len(pools))
	}

	var hosts map[string]*serviced.Host = nil

	err = lclient.GetHosts(request, &hosts)
	if err != nil {
		glog.Fatalf("Could not get hosts, %s", err)
	}

	host, err := serviced.CurrentContextAsHost("default")
	if err != nil {
		t.Fatal("Could not get currentContextAsHost", err)
	}
	err = lclient.AddHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not add host", err)
	}

	host.Name = "foo"
	err = lclient.UpdateHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not update host", err)
	} else {
		glog.Infoln("update of host is OK")
	}
	err = lclient.GetHosts(request, &hosts)
	if err != nil {
		t.Fatal("Error getting updated hosts.", err)
	}
	if hosts[host.Id].Name != "foo" {
		t.Fatal("Expected host to be named foo.", err)
	}

	err = lclient.RemoveHost(host.Id, &unused)
	if err != nil {
		t.Fatal("Could not remove host.", err)
	}
	hosts = nil
	err = lclient.GetHosts(request, &hosts)
	if err != nil {
		t.Fatal("Error getting updated hosts.", err)
	}
	_, exists := hosts[host.Id]
	if exists {
		t.Fatal("Host was not removed.", err)
	}

	var services []*serviced.Service
	err = lclient.GetServices(request, &services)
	if err != nil {
		t.Fatal("Error getting services.", err)
	}
	if len(services) != 0 {
		t.Fatal("Expecting 0 services")
	}

	err = lclient.GetServicesForHost("dasdfasdf", &services)
	if err == nil {
		t.Fatal("Expected error looking for non-existent service.")
	}

	service, err := serviced.NewService()
	if err != nil {
		t.Fatal("Could not create a new service object")
	}
	service.Name = "HelloWorld"
	service.Startup = "/bin/sh -c \"while true; do echo hello world; sleep 1; done\""
	service.Description = "a simple daemon"
	service.Instances = 1
	service.ImageId = "base"
	service.PoolId = "default"
	service.DesiredState = 0
	service.Endpoints = &[]serviced.ServiceEndpoint{
		serviced.ServiceEndpoint{serviced.TCP, 3306, "mysql", "remote"}}
	err = lclient.AddService(*service, &unused)
	if err != nil {
		t.Fatal("Could not add a service to the control plane")
	}

	err = lclient.GetServices(request, &services)
	if err != nil {
		t.Fatalf("Could not get services after adding a service: %s", err)
	}
	if len(services) != 1 {
		t.Fatalf("Expected 1 service, got %d", len(services))
	}
	if services[0].Endpoints == nil || len(*services[0].Endpoints) == 0 {
		t.Fatalf("Expected 1 endpoint, got nil or 0")
	}

	err = lclient.RemoveService(service.Id, &unused)
	if err != nil {
		t.Fatal("Could not delete the helloWorld service")
	}

}

func TestServiceStart(t *testing.T) {

	cleanTestDB(t)

	var err error
	pool, _ := serviced.NewResourcePool("default")
	err = lclient.AddResourcePool(*pool, &unused)
	if err != nil {
		t.Fatal("Problem adding resource pool", err)
	}

	host, err := serviced.CurrentContextAsHost("default")
	glog.Infoln("Got a currentContextAsHost()")
	if err != nil {
		t.Fatal("Could not get currentContextAsHost", err)
	}
	err = lclient.AddHost(*host, &unused)
	if err != nil {
		t.Fatal("Could not add host", err)
	}

	// add a new service
	service, _ := serviced.NewService()
	service.Name = "My test service!"
	service.PoolId = pool.Id
	service.Startup = "/bin/sh -c \"while true; do echo hello world; sleep 1; done\""
	err = lclient.AddService(*service, &unused)
	if err != nil {
		t.Fatal("Could not add service: ", err)
	}

	// start the service
	var hostId string
	err = lclient.StartService(service.Id, &hostId)
	if err != nil {
		t.Fatal("Got error starting service: ", err)
	}

	var services []*serviced.Service
	// get the services for a host
	err = lclient.GetServicesForHost(host.Id, &services)
	if err != nil {
		t.Fatal("Could not get services for host: ", err)
	}
	glog.Infof("Got %d services for %s", len(services), host.Id)
}
