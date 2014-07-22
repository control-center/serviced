// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
)

const (
	NilService = "NilService"
)

var DefaultServiceAPITest = ServiceAPITest{
	services:  DefaultTestServices,
	pools:     DefaultTestPools,
	snapshots: DefaultTestSnapshots,
}

var DefaultTestServices = []*service.Service{
	{
		ID:             "test-service-1",
		Name:           "Zenoss",
		Startup:        "startup command 1",
		Instances:      0,
		InstanceLimits: domain.MinMax{0, 0},
		ImageID:        "quay.io/zenossinc/tenantid1-core5x",
		PoolID:         "default",
		DesiredState:   1,
		Launch:         "auto",
		DeploymentID:   "Zenoss-resmgr",
		Runs: map[string]string{
			"hello":   "echo hello world",
			"goodbye": "echo goodbye world",
		},
	}, {
		ID:             "test-service-2",
		Name:           "Zope",
		Startup:        "startup command 2",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1},
		ImageID:        "quay.io/zenossinc/tenantid2-core5x",
		PoolID:         "default",
		DesiredState:   1,
		Launch:         "auto",
		DeploymentID:   "Zenoss-core",
	}, {
		ID:             "test-service-3",
		Name:           "zencommand",
		Startup:        "startup command 3",
		Instances:      2,
		InstanceLimits: domain.MinMax{2, 2},
		ImageID:        "quay.io/zenossinc/tenantid1-opentsdb",
		PoolID:         "remote",
		DesiredState:   1,
		Launch:         "manual",
		DeploymentID:   "Zenoss-core",
	},
}

var (
	ErrNoServiceFound = errors.New("no service found")
	ErrInvalidService = errors.New("invalid service")
	ErrCmdNotFound    = errors.New("command not found")
)

type ServiceAPITest struct {
	api.API
	fail      bool
	services  []*service.Service
	pools     []*pool.ResourcePool
	snapshots []string
}

func InitServiceAPITest(args ...string) {
	New(DefaultServiceAPITest).Run(args)
}

func (t ServiceAPITest) GetServices() ([]*service.Service, error) {
	if t.fail {
		return nil, ErrInvalidService
	}
	return t.services, nil
}

func (t ServiceAPITest) GetResourcePools() ([]*pool.ResourcePool, error) {
	if t.fail {
		return nil, ErrInvalidService
	}
	return t.pools, nil
}

func (t ServiceAPITest) GetService(id string) (*service.Service, error) {
	if t.fail {
		return nil, ErrInvalidService
	}

	for _, s := range t.services {
		if s.ID == id {
			return s, nil
		}
	}

	return nil, nil
}

func (t ServiceAPITest) AddService(config api.ServiceConfig) (*service.Service, error) {
	if t.fail {
		return nil, ErrInvalidService
	} else if config.Name == NilService {
		return nil, nil
	}

	endpoints := make([]service.ServiceEndpoint, len(*config.LocalPorts)+len(*config.RemotePorts))
	i := 0
	for _, e := range *config.LocalPorts {
		e.Purpose = "local"
		endpoints[i] = service.BuildServiceEndpoint(e)
		i++
	}
	for _, e := range *config.RemotePorts {
		e.Purpose = "remote"
		endpoints[i] = service.BuildServiceEndpoint(e)
		i++
	}

	s := service.Service{
		ID:             fmt.Sprintf("%s-%s-%s", config.Name, config.PoolID, config.ImageID),
		Name:           config.Name,
		PoolID:         config.PoolID,
		ImageID:        config.ImageID,
		Endpoints:      endpoints,
		Startup:        config.Command,
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1},
	}

	return &s, nil
}

func (t ServiceAPITest) RemoveService(config api.RemoveServiceConfig) error {
	if s, err := t.GetService(config.ServiceID); err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	return nil
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.Service, error) {
	var s service.Service

	if err := json.NewDecoder(reader).Decode(&s); err != nil {
		return nil, ErrInvalidService
	}

	if _, err := t.GetService(s.ID); err != nil {
		return nil, err
	}

	return &s, nil
}

func (t ServiceAPITest) StartService(id string) error {
	if s, err := t.GetService(id); err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	return nil
}

func (t ServiceAPITest) StopService(id string) error {
	if s, err := t.GetService(id); err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	return nil
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) error {
	if _, err := t.GetService(config.ServiceID); err != nil {
		return err
	}
	return nil
}

func (t ServiceAPITest) StartProxy(config api.ControllerOptions) error {
	if s, err := t.GetService(config.ServiceID); err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	fmt.Printf("%s\n", strings.Join(config.Command, " "))
	return nil
}

func (t ServiceAPITest) StartShell(config api.ShellConfig) error {
	if s, err := t.GetService(config.ServiceID); err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	fmt.Printf("%s %s\n", config.Command, strings.Join(config.Args, " "))
	return nil
}

func (t ServiceAPITest) RunShell(config api.ShellConfig) error {
	s, err := t.GetService(config.ServiceID)
	if err != nil {
		return err
	} else if s == nil {
		return ErrNoServiceFound
	}

	command, ok := s.Runs[config.Command]
	if !ok {
		return ErrCmdNotFound
	}

	fmt.Printf("%s %s\n", command, strings.Join(config.Args, " "))
	return nil
}

func (t ServiceAPITest) GetSnapshotsByServiceID(id string) ([]string, error) {
	if t.fail {
		return nil, ErrInvalidSnapshot
	}

	var snapshots []string
	for _, s := range t.snapshots {
		if strings.HasPrefix(s, id) {
			snapshots = append(snapshots, s)
		}
	}

	return snapshots, nil
}

func (t ServiceAPITest) AddSnapshot(id string) (string, error) {
	s, err := t.GetService(id)
	if err != nil {
		return "", ErrInvalidSnapshot
	} else if s == nil {
		return "", nil
	}

	return fmt.Sprintf("%s-snapshot", id), nil
}

func TestServicedCLI_CmdServiceList_one(t *testing.T) {
	serviceID := "test-service-1"

	expected, err := DefaultServiceAPITest.GetService(serviceID)
	if err != nil {
		t.Fatal(err)
	}

	var actual service.Service
	output := pipe(InitServiceAPITest, "serviced", "service", "list", serviceID)
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update Service.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdServiceList_all(t *testing.T) {
	expected, err := DefaultServiceAPITest.GetServices()
	if err != nil {
		t.Fatal(err)
	}

	var actual []*service.Service
	output := pipe(InitServiceAPITest, "serviced", "service", "list", "--verbose")
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update Service.Equals?
	if len(actual) != len(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
	for i := range actual {
		if !actual[i].Equals(expected[i]) {
			t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
		}
	}
}

func ExampleServicedCLI_CmdServiceList() {
	// Gofmt cleans up the spaces at the end of each row
	InitServiceAPITest("serviced", "service", "list")
}

func ExampleServicedCLI_CmdServiceList_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	// Error retrieving service
	pipeStderr(InitServiceAPITest, "serviced", "service", "list", "test-service-1")
	// Error retrieving all services
	pipeStderr(InitServiceAPITest, "serviced", "service", "list")

	// Output:
	// invalid service
	// invalid service
}

func ExampleServicedCLI_CmdServiceList_err() {
	DefaultServiceAPITest.services = nil
	defer func() { DefaultServiceAPITest.services = DefaultTestServices }()
	// Service not found
	pipeStderr(InitServiceAPITest, "serviced", "service", "list", "test-service-0")
	// No Services found
	pipeStderr(InitServiceAPITest, "serviced", "service", "list")

	// Output:
	// service not found
	// no services found
}

func ExampleServicedCLI_CmdServiceList_complete() {
	InitServiceAPITest("serviced", "service", "list", "--generate-bash-completion")

	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	InitServiceAPITest("serviced", "service", "list", "--generate-bash-completion")

	// Output:
	// test-service-1
	// test-service-2
	// test-service-3
}

func ExampleServicedCLI_CmdServiceAdd() {
	InitServiceAPITest("serviced", "service", "add", "test-service", "test-pool", "test-image", "bash -c lsof")

	// Output:
	// test-service-test-pool-test-image
}

func ExampleServicedCLI_CmdServiceAdd_usage() {
	InitServiceAPITest("serviced", "service", "add")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    add - Adds a new service
	//
	// USAGE:
	//    command add [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service add NAME POOLID IMAGEID COMMAND
	//
	// OPTIONS:
	//    -p 	`-p option -p option` Expose a port for this service (e.g. -p tcp:3306:mysql)
	//    -q 	`-q option -q option` Map a remote service port (e.g. -q tcp:3306:mysql)
}

func ExampleServicedCLI_CmdServiceAdd_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "add", "test-service", "test-pool", "test-image", "bash -c lsof")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceAdd_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "add", NilService, "test-pool", "test-image", "bash -c lsof")

	// Output:
	// received nil service definition
}

func ExampleServicedCLI_CmdServiceAdd_complete() {
	InitServiceAPITest("serviced", "service", "add", "test-service", "--generate-bash-completion")

	// Output:
	// test-pool-id-1
	// test-pool-id-2
	// test-pool-id-3
}

func ExampleServicedCLI_CmdServiceRemove() {
	InitServiceAPITest("serviced", "service", "remove", "test-service-1")
	InitServiceAPITest("serviced", "service", "remove", "test-service-2")
	InitServiceAPITest("serviced", "service", "remove", "-R", "test-service-2")
	InitServiceAPITest("serviced", "service", "remove", "-R=false", "test-service-2")

	// Output:
	// test-service-1
	// test-service-2
	// test-service-2
	// test-service-2
}

func ExampleServicedCLI_CmdServiceRemove_usage() {
	InitServiceAPITest("serviced", "service", "remove")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    remove - Removes an existing service
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service remove SERVICEID ...
	//
	// OPTIONS:
	//    --remove-snapshots, -R	Remove snapshots associated with removed service

}

func ExampleServicedCLI_CmdServiceRemove_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "remove", "test-service-0")

	// Output:
	// no service found
}

func ExampleServicedCLI_CmdServiceRemove_complete() {
	InitServiceAPITest("serviced", "service", "remove", "--generate-bash-completion")
	fmt.Println("")
	InitServiceAPITest("serviced", "service", "remove", "test-service-2", "--generate-bash-completion")

	// Output:
	// test-service-1
	// test-service-2
	// test-service-3
	//
	// test-service-1
	// test-service-3
}

func ExampleServicedCLI_CmdServiceEdit() {
	// This opens an editor, so I am not sure how to test this yet :)
	InitServiceAPITest("serviced", "service", "edit", "test-service-1")
}

func ExampleServicedCLI_CmdServiceEdit_usage() {
	// In the output, under OPTIONS, if you see a 'vim' or 'emacs',
	// it is because you have the following
	// environment vairable set: GIT_EDITOR
	// echo $GIT_EDITOR
	InitServiceAPITest("serviced", "service", "edit")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    edit - Edits an existing service in a text editor
	//
	// USAGE:
	//    command edit [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service edit SERVICEID
	//
	// OPTIONS:
	//    --editor, -e 	Editor used to update the service definition
}

func ExampleServicedCLI_CmdServiceEdit_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	// Failed to get service
	pipeStderr(InitServiceAPITest, "serviced", "service", "edit", "test-service-1")
	// TODO: Failed to update service

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceEdit_err() {
	// Service not found
	pipeStderr(InitServiceAPITest, "serviced", "service", "edit", "test-service-0")
	// TODO: Nil Service after update

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceAssignIPs() {
	// Auto-assign
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-1")
	// Manual-assign
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-2", "127.0.0.1")

	// Output:
	//
}

func ExampleServicedCLI_CmdServiceAssignIPs_usage() {
	InitServiceAPITest("serviced", "service", "assign-ip")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    assign-ip - Assigns an IP address to a service's endpoints requiring an explicit IP address
	//
	// USAGE:
	//    command assign-ip [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service assign-ip SERVICEID [IPADDRESS]
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceAssignIPs_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "assign-ip", "test-service-3")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceAssignIPs_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "assign-ip", "test-service-0", "100.99.88.1")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStart() {
	InitServiceAPITest("serviced", "service", "start", "test-service-1")

	// Output:
	// Service scheduled to start.
}

func ExampleServicedCLI_CmdServiceStart_usage() {
	InitServiceAPITest("serviced", "service", "start")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    start - Starts a service
	//
	// USAGE:
	//    command start [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service start SERVICEID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceStart_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "start", "test-service-1")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceStart_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "start", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStop() {
	InitServiceAPITest("serviced", "service", "stop", "test-service-2")

	// Output:
	// Service scheduled to stop.
}

func ExampleServicedCLI_CmdServiceStop_usage() {
	InitServiceAPITest("serviced", "service", "stop")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    stop - Stops a service
	//
	// USAGE:
	//    command stop [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service stop SERVICEID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceStop_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "stop", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceProxy_usage() {
	// FIXME: Non-reproducible error on buildbox
	InitServiceAPITest("serviced", "service", "proxy")

	// Incorrect Usage.
	//
	// NAME:
	//    proxy - Starts a server proxy for a container
	//
	// USAGE:
	//    command proxy [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service proxy SERVICEID COMMAND
	//
	// OPTIONS:
	//    --muxport '22250'			multiplexing port to use
	//    --mux				enable port multiplexing
	//    --tls				enable tls
	//    --keyfile 				path to private key file (defaults to compiled in private keys
	//    --certfile 				path to public certificate file (defaults to compiled in public cert)
	//    --endpoint '10.87.103.1:4979'	serviced endpoint address
	//    --autorestart			restart process automatically when it finishes
	//    --logstash				forward service logs via logstash-forwarder
	//
}

func ExampleServicedCLI_CmdServiceShell() {
	InitServiceAPITest("serviced", "service", "shell", "test-service-1", "some", "command")

	// Output:
	// some command
}

/*
removed test due to --endpoint
func ExampleServicedCLI_CmdServiceShell_usage() {
	// FIXME: IP in --endpoint is too specific
	InitServiceAPITest("serviced", "service", "shell")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    shell - Starts a service instance
	//
	// USAGE:
	//    command shell [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service shell SERVICEID COMMAND
	//
	// OPTIONS:
	//    --saveas, -s 				saves the service instance with the given name
	//    --interactive, -i				runs the service instance as a tty
	//    --mount '--mount option --mount option'	bind mount: HOST_PATH[,CONTAINER_PATH]
	//    --endpoint '10.87.103.1:4979'		endpoint for remote serviced (example.com:4979)
	//    -v '0'					log level for V logs

}
*/

func ExampleServicedCLI_CmdServiceShell_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "shell", "test-service-0", "some", "command")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceRun_list() {
	InitServiceAPITest("serviced", "service", "run", "test-service-1")

	// Output:
	// hello
	// goodbye
}

func ExampleServicedCLI_CmdServiceRun_exec() {
	InitServiceAPITest("serviced", "service", "run", "-i", "test-service-1", "hello", "-i")

	// Output:
	// echo hello world -i
}

/*
removed test due to --endpoint
func ExampleServicedCLI_CmdServiceRun_usage() {
	InitServiceAPITest("serviced", "service", "run")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    run - Runs a service command in a service instance
	//
	// USAGE:
	//    command run [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service run SERVICEID COMMAND [ARGS]
	//
	// OPTIONS:
	//    --interactive, -i				runs the service instance as a tty
	//    --mount '--mount option --mount option'	bind mount: HOST_PATH[,CONTAINER_PATH]
	//    --endpoint '10.87.103.1:4979'		endpoint for remote serviced (example.com:4979)
}
*/

func ExampleServicedCLI_CmdServiceRun_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "run", "test-service-0", "goodbye")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceRun_complete() {
	// FIXME: Does not print run commands
	InitServiceAPITest("serviced", "service", "run", "--generate-bash-completion")
	fmt.Println("")
	InitServiceAPITest("serviced", "service", "run", "test-service-1", "--generate-bash-completion")
	fmt.Println("")
	InitServiceAPITest("serviced", "service", "run", "test-service-2", "--generate-bash-completion")

	// test-service-1
	// test-service-2
	// test-service-3
	//
	// hello
	// goodbye
}

// TODO: ServicedCLI.CmdServiceAttach
// TODO: ServicedCLI.CmdServiceAction

func ExampleServicedCLI_CmdServiceListSnapshots() {
	InitServiceAPITest("serviced", "service", "list-snapshots", "test-service-1")

	// Output:
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
}

func ExampleServicedCLI_CmdServiceListSnapshots_usage() {
	InitServiceAPITest("serviced", "service", "list-snapshots")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    list-snapshots - Lists the snapshots for a service
	//
	// USAGE:
	//    command list-snapshots [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service list-snapshots SERVICEID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceListSnapshots_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-1")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceListSnapshots_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-3")

	// Output:
	// no snapshots found
}

func ExampleServicedCLI_CmdServiceSnapshot() {
	InitServiceAPITest("serviced", "service", "snapshot", "test-service-2")

	// Output:
	// test-service-2-snapshot
}

func ExampleServicedCLI_CmdServiceSnapshot_usage() {
	InitServiceAPITest("serviced", "service", "snapshot")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    snapshot - Takes a snapshot of the service
	//
	// USAGE:
	//    command snapshot [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service snapshot SERVICEID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceSnapshot_fail() {
	DefaultServiceAPITest.fail = true
	defer func() { DefaultServiceAPITest.fail = false }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "snapshot", "test-service-1")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceSnapshot_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "snapshot", "test-service-0")

	// Output:
	// service not found
}
