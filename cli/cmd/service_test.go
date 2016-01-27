// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	//	"sort"
	"strings"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/domain/applicationendpoint"
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
)

const (
	NilService = "NilService"
)

var DefaultServiceAPITest = ServiceAPITest{
	errs:            make(map[string]error, 10),
	services:        DefaultTestServices,
	runningServices: DefaultTestRunningServices,
	pools:           DefaultTestPools,
	hosts:           DefaultTestHosts,
	snapshots:       DefaultTestSnapshots,
	endpoints:       DefaultEndpoints,
}

var DefaultTestServices = []service.Service{
	{
		ID:             "test-service-1",
		Name:           "Zenoss",
		Startup:        "startup command 1",
		Instances:      0,
		InstanceLimits: domain.MinMax{0, 0, 0},
		ImageID:        "quay.io/zenossinc/tenantid1-core5x",
		PoolID:         "default",
		DesiredState:   int(service.SVCRun),
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
		InstanceLimits: domain.MinMax{1, 1, 1},
		ImageID:        "quay.io/zenossinc/tenantid2-core5x",
		PoolID:         "default",
		DesiredState:   int(service.SVCRun),
		Launch:         "auto",
		DeploymentID:   "Zenoss-core",
	}, {
		ID:             "test-service-3",
		Name:           "zencommand",
		Startup:        "startup command 3",
		Instances:      2,
		InstanceLimits: domain.MinMax{2, 2, 2},
		ImageID:        "quay.io/zenossinc/tenantid1-opentsdb",
		PoolID:         "remote",
		DesiredState:   int(service.SVCRun),
		Launch:         "manual",
		DeploymentID:   "Zenoss-core",
	},
}

var DefaultTestRunningServices = []dao.RunningService{
	{
		ID:              "abcdefg",
		ServiceID:       "test-service-2",
		HostID:          "test-host-id-1",
		DockerID:        "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaa",
		Name:            "Zope",
		Startup:         "startup command 2",
		Instances:       1,
		ImageID:         "quay.io/zenossinc/tenantid2-core5x",
		PoolID:          "default",
		DesiredState:    int(service.SVCRun),
		InstanceID:      0,
		ParentServiceID: "test-service-1",
	},
	{
		ID:              "hijklmn",
		ServiceID:       "test-service-3",
		HostID:          "test-host-id-2",
		DockerID:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Name:            "zencommand",
		Startup:         "startup command 3",
		Instances:       2,
		ImageID:         "quay.io/zenossinc/tenantid1-opentsdb",
		PoolID:          "remote",
		DesiredState:    int(service.SVCRun),
		InstanceID:      0,
		ParentServiceID: "test-service-2",
	},
	{
		ID:              "opqrstu",
		ServiceID:       "test-service-3",
		HostID:          "test-host-id-2",
		DockerID:        "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbb",
		Name:            "zencommand",
		Startup:         "startup command 3",
		Instances:       2,
		ImageID:         "quay.io/zenossinc/tenantid1-opentsdb",
		PoolID:          "remote",
		DesiredState:    int(service.SVCRun),
		InstanceID:      1,
		ParentServiceID: "test-service-2",
	},
}

var DefaultEndpoints = []applicationendpoint.EndpointReport{
	{
		Endpoint: applicationendpoint.ApplicationEndpoint{
			ServiceID:     "test-service-2",
			InstanceID:    1,
			Application:   "endpointName1",
			Purpose:       "export",
			HostID:        "hostID1",
			HostIP:        "hostIP1",
			HostPort:      10,
			ContainerID:   "containerID1",
			ContainerIP:   "containerIP1",
			ContainerPort: 100,
		},
		Messages: []string{},
	}, {
		Endpoint: applicationendpoint.ApplicationEndpoint{
			ServiceID:     "test-service-2",
			InstanceID:    2,
			Application:   "endpointName2",
			Purpose:       "import",
			HostID:        "hostID2",
			HostIP:        "hostIP2",
			HostPort:      20,
			ContainerID:   "containerID2",
			ContainerIP:   "containerIP2",
			ContainerPort: 200,
		},
		Messages: []string{},
	},
}

var (
	ErrNoServiceFound        = errors.New("no service found")
	ErrNoRunningServiceFound = errors.New("no matches found")
	ErrInvalidService        = errors.New("invalid service")
	ErrCmdNotFound           = errors.New("command not found")
	ErrStub                  = errors.New("stub for facade failed")
)

type ServiceAPITest struct {
	api.API
	errs            map[string]error
	services        []service.Service
	runningServices []dao.RunningService
	pools           []pool.ResourcePool
	hosts           []host.Host
	snapshots       []dao.SnapshotInfo
	endpoints       []applicationendpoint.EndpointReport
}

func InitServiceAPITest(args ...string) {
	c := New(DefaultServiceAPITest, utils.TestConfigReader(make(map[string]string)))
	c.exitDisabled = true
	c.Run(args)
}

func (t ServiceAPITest) GetServices() ([]service.Service, error) {
	if t.errs["GetServices"] != nil {
		return nil, t.errs["GetServices"]
	}
	return t.services, nil
}

func (t ServiceAPITest) GetRunningServices() ([]dao.RunningService, error) {
	if t.errs["GetRunningServices"] != nil {
		return nil, t.errs["GetRunningServices"]
	}
	return t.runningServices, nil
}

func (t ServiceAPITest) GetResourcePools() ([]pool.ResourcePool, error) {
	if t.errs["GetResourcePools"] != nil {
		return nil, t.errs["GetResourcePools"]
	}
	return t.pools, nil
}

func (t ServiceAPITest) GetHosts() ([]host.Host, error) {
	if t.errs["GetHosts"] != nil {
		return nil, t.errs["GetHosts"]
	}
	return t.hosts, nil
}

func (t ServiceAPITest) GetHostMap() (map[string]host.Host, error) {
	if t.errs["GetHostMap"] != nil {
		return nil, t.errs["GetHostMap"]
	}
	return make(map[string]host.Host), nil
}

func (t ServiceAPITest) GetEndpoints(serviceID string, reportImports, reportExports, validate bool) ([]applicationendpoint.EndpointReport, error) {
	if t.errs["GetEndpoints"] != nil {
		return nil, t.errs["GetEndpoints"]
	} else if serviceID == "test-service-2" {
		return t.endpoints, nil
	}
	return []applicationendpoint.EndpointReport{}, nil
}

func (t ServiceAPITest) GetService(id string) (*service.Service, error) {
	if t.errs["GetService"] != nil {
		return nil, t.errs["GetService"]
	}

	for i, s := range t.services {
		if s.ID == id {
			return &t.services[i], nil
		}
	}
	return nil, nil
}

func (t ServiceAPITest) AddService(config api.ServiceConfig) (*service.Service, error) {
	if t.errs["AddService"] != nil {
		return nil, t.errs["AddService"]
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
		ID:              fmt.Sprintf("%s-%s-%s", config.Name, config.ParentServiceID, config.ImageID),
		ParentServiceID: config.ParentServiceID,
		Name:            config.Name,
		ImageID:         config.ImageID,
		Endpoints:       endpoints,
		Startup:         config.Command,
		Instances:       1,
		InstanceLimits:  domain.MinMax{1, 1, 1},
	}

	return &s, nil
}

func (t ServiceAPITest) RemoveService(id string) error {
	if t.errs["RemoveService"] != nil {
		return t.errs["RemoveService"]
	}
	return nil
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.Service, error) {
	var svc service.Service

	if err := json.NewDecoder(reader).Decode(&svc); err != nil {
		return nil, ErrInvalidService
	}

	if _, err := t.GetService(svc.ID); err != nil {
		return nil, err
	}

	return &svc, nil
}

func (t ServiceAPITest) StartService(cfg api.SchedulerConfig) (int, error) {
	if t.errs["StartService"] != nil {
		return 0, t.errs["StartService"]
	}
	return 1, nil
}

func (t ServiceAPITest) RestartService(cfg api.SchedulerConfig) (int, error) {
	if t.errs["RestartService"] != nil {
		return 0, t.errs["RestartService"]
	}
	return 1, nil
}

func (t ServiceAPITest) StopService(cfg api.SchedulerConfig) (int, error) {
	if s, err := t.GetService(cfg.ServiceID); err != nil {
		return 0, err
	} else if s == nil {
		return 0, ErrNoServiceFound
	}

	return 1, nil
}

func (t ServiceAPITest) StopRunningService(hostID string, serviceStateID string) error {
	for _, rs := range t.runningServices {
		if rs.HostID == hostID && rs.ID == serviceStateID {
			return nil
		}
	}

	return ErrNoRunningServiceFound
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) error {
	if t.errs["AssignIP"] != nil {
		return t.errs["AssignIP"]
	}
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

func (t ServiceAPITest) RunShell(config api.ShellConfig, stopChan chan struct{}) (int, error) {
	s, err := t.GetService(config.ServiceID)
	if err != nil {
		return 1, err
	} else if s == nil {
		return 1, ErrNoServiceFound
	}

	command, ok := s.Runs[config.Command]
	if !ok {
		return 1, ErrCmdNotFound
	}

	fmt.Printf("%s %s\n", command, strings.Join(config.Args, " "))
	return 0, nil
}

func (t ServiceAPITest) GetSnapshotsByServiceID(id string) ([]dao.SnapshotInfo, error) {
	if t.errs["GetSnapshotsByServiceID"] != nil {
		return nil, t.errs["GetSnapshotsByServiceID"]
	}

	var snapshots []dao.SnapshotInfo
	for _, s := range t.snapshots {
		if strings.HasPrefix(s.SnapshotID, id) {
			snapshots = append(snapshots, s)
		}
	}

	return snapshots, nil
}

func (t ServiceAPITest) AddSnapshot(config api.SnapshotConfig) (string, error) {
	if t.errs["AddSnapshot"] != nil {
		return "", t.errs["AddSnapshot"]
	}

	return fmt.Sprintf("%s-snapshot description=%q tags=%q", config.ServiceID, config.Message, config.Tag), nil
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
		if !actual[i].Equals(&expected[i]) {
			t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
		}
	}
}

func ExampleServicedCLI_CmdServiceList() {
	// Gofmt cleans up the spaces at the end of each row
	InitServiceAPITest("serviced", "service", "list")
}

func ExampleServicedCLI_CmdServiceList_fail() {
	DefaultServiceAPITest.errs["GetServices"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetServices"] = nil }()
	// Error retrieving service
	pipeStderr(InitServiceAPITest, "serviced", "service", "list", "test-service-0")
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

	DefaultServiceAPITest.errs["GetServices"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetServices"] = nil }()
	InitServiceAPITest("serviced", "service", "list", "--generate-bash-completion")

	// Output:
	// test-service-1
	// test-service-2
	// test-service-3
}

func ExampleServicedCLI_CmdServiceAdd() {
	InitServiceAPITest("serviced", "service", "add", "--parent-id", "test-service-1", "test-service", "test-image", "bash -c lsof")

	// Output:
	// test-service-test-service-1-test-image
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
	//    serviced service add NAME IMAGEID COMMAND
	//
	// OPTIONS:
	//    -p 		`-p option -p option` Expose a port for this service (e.g. -p tcp:3306:mysql)
	//    -q 		`-q option -q option` Map a remote service port (e.g. -q tcp:3306:mysql)
	//    --parent-id 	Parent service ID for which this service relates
}

func ExampleServicedCLI_CmdServiceAdd_fail() {
	DefaultServiceAPITest.errs["AddService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["AddService"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "add", "--parent-id", "test-service-1", "test-service", "test-image", "bash -c lsof")

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceAdd_missingParentArg() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "add", "test-service", "test-pool", "test-image", "bash -c lsof")

	// Output:
	// Must specify a parent service ID
}

func ExampleServicedCLI_CmdServiceAdd_parentNotFound() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "add", "--parent-id", "test-parent", NilService, "test-image", "bash -c lsof")

	// Output:
	// Error searching for parent service: service not found
}

func ExampleServicedCLI_CmdServiceRemove() {
	InitServiceAPITest("serviced", "service", "remove", "test-service-1")
	InitServiceAPITest("serviced", "service", "remove", "test-service-2")

	// Output:
	// test-service-1
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
	//    serviced service remove SERVICEID
	//
	// OPTIONS:

}

func ExampleServicedCLI_CmdServiceRemove_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "remove", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceRemove_failed() {
	DefaultServiceAPITest.errs["RemoveService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["RemoveService"] = nil }()

	pipeStderr(InitServiceAPITest, "serviced", "service", "remove", "test-service-1")

	// Output:
	// test-service-1: stub for facade failed
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
	DefaultServiceAPITest.errs["GetServices"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetServices"] = nil }()
	// Failed to get service
	pipeStderr(InitServiceAPITest, "serviced", "service", "edit", "test-service-0")
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
	DefaultServiceAPITest.errs["AssignIP"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["AssignIP"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "assign-ip", "test-service-3")

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceAssignIPs_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "assign-ip", "test-service-0", "100.99.88.1")

	// Output:
	// service not found
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
	//    --auto-launch	Recursively schedules child services
}

func ExampleServicedCLI_CmdServiceStart_fail() {
	DefaultServiceAPITest.errs["StartService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["StartService"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "start", "test-service-1")

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceStart_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "start", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStart() {
	InitServiceAPITest("serviced", "service", "start", "test-service-2")

	// Output:
	// Scheduled 1 service(s) to start
}

func ExampleServicedCLI_CmdServiceRestart_usage() {
	InitServiceAPITest("serviced", "service", "restart")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    restart - Restarts a service
	//
	// USAGE:
	//    command restart [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service restart { SERVICEID | INSTANCEID }
	//
	// OPTIONS:
	//    --auto-launch	Recursively schedules child services
}

func ExampleServicedCLI_CmdServiceRestart_fail() {
	DefaultServiceAPITest.errs["RestartService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["RestartService"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "restart", "test-service-1")

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceRestart_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "restart", "test-service-0")    // Non-existant service
	pipeStderr(InitServiceAPITest, "serviced", "service", "restart", "test-service-3/4")  // Non-existant instance
	pipeStderr(InitServiceAPITest, "serviced", "service", "restart", "test-service-3/a")  // Non-numeric instance number
	pipeStderr(InitServiceAPITest, "serviced", "service", "restart", "test-service-3/0b") // Non-numeric instance number

	// Output:
	// service not found
	// no matches found
	// service not found
	// service not found
}

func ExampleServicedCLI_CmdServiceRestart() {
	InitServiceAPITest("serviced", "service", "restart", "test-service-2")
	InitServiceAPITest("serviced", "service", "restart", "test-service-3/1") // Specific instance

	// Output:
	// Restarting 1 service(s)
	// Restarting 1 service(s)
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
	//    --auto-launch	Recursively schedules child services
}

func ExampleServicedCLI_CmdServiceStop_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "stop", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStop() {
	InitServiceAPITest("serviced", "service", "stop", "test-service-2")

	// Output:
	// Scheduled 1 service(s) to stop
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
	//    serviced service shell SERVICEID [COMMAND]
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
	// exit code 1
}

/*func ExampleServicedCLI_CmdServiceRun_list() {
	output := pipe(InitServiceAPITest, "serviced", "service", "run", "test-service-1")
	actual := strings.Split(string(output[:]), "\n")
	sort.Strings(actual)

	for _, item := range actual {
		fmt.Printf("%s\n", item)
	}

	// Output:
	// goodbye
	// hello
}
*/
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
	// exit code 1
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
	// test-service-1-snapshot-1 description 1
	// test-service-1-snapshot-2 description 2
	// test-service-1-invalid [DEPRECATED]

}

func TestServicedCLI_CmdServiceListSnapshots_ShowTagsShort(t *testing.T) {
	output := pipe(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-1", "-t")
	expected :=
		"Snapshot                                 Description        Tags" +
			"\ntest-service-1-snapshot-1                description 1      tag-1" +
			"\ntest-service-1-snapshot-2                description 2      tag-2,tag-3" +
			"\ntest-service-1-invalid [DEPRECATED]"

	outStr := TrimLines(fmt.Sprintf("%s", output))
	expected = TrimLines(expected)

	if expected != outStr {
		t.Fatalf("\ngot:\n%s\nwant:\n%s", outStr, expected)
	}
}

func TestServicedCLI_CmdServiceListSnapshots_ShowTagsLong(t *testing.T) {
	output := pipe(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-1", "--show-tags")
	expected :=
		"Snapshot                                 Description        Tags" +
			"\ntest-service-1-snapshot-1                description 1      tag-1" +
			"\ntest-service-1-snapshot-2                description 2      tag-2,tag-3" +
			"\ntest-service-1-invalid [DEPRECATED]"

	outStr := TrimLines(fmt.Sprintf("%s", output))
	expected = TrimLines(expected)

	if expected != outStr {
		t.Fatalf("\ngot:\n%s\nwant:\n%s", outStr, expected)
	}
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
	//    --show-tags, -t	shows the tags associated with each snapshot
}

func ExampleServicedCLI_CmdServiceListSnapshots_fail() {
	DefaultServiceAPITest.errs["GetSnapshotsByServiceID"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["GetSnapshotsByServiceID"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-1")

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceListSnapshots_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "list-snapshots", "test-service-3")

	// Output:
	// no snapshots found
}

func ExampleServicedCLI_CmdServiceSnapshot() {
	InitServiceAPITest("serviced", "service", "snapshot", "test-service-2")

	// Output:
	// test-service-2-snapshot description="" tags=""
}

func ExampleServicedCLI_CmdServiceSnapshot_withDescription() {
	InitServiceAPITest("serviced", "service", "snapshot", "test-service-2", "-d", "some description")

	// Output:
	// test-service-2-snapshot description="some description" tags=""
}

func ExampleServicedCLI_CmdServiceSnapshot_withDescriptionAndTag() {
	InitServiceAPITest("serviced", "service", "snapshot", "test-service-2", "-d", "some description", "-t", "tag1")

	// Output:
	// test-service-2-snapshot description="some description" tags="tag1"
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
	//    --description, -d 	a description of the snapshot
	//    --tag, -t 		a unique tag for the snapshot

}

func ExampleServicedCLI_CmdServiceSnapshot_fail() {
	DefaultServiceAPITest.errs["AddSnapshot"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["AddSnapshot"] = nil }()
	pipeStderr(InitServiceAPITest, "serviced", "service", "snapshot", "test-service-1")

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceSnapshot_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "snapshot", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceEndpoints_usage() {
	InitServiceAPITest("serviced", "service", "endpoints")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    endpoints - List the endpoints defined for the service
	//
	// USAGE:
	//    command endpoints [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service endpoints SERVICEID
	//
	// OPTIONS:
	//    --imports, -i	include only imported endpoints
	//    --all, -a		include all endpoints (imports and exports)
	//    --verify, -v		verify endpoints

}

func ExampleServicedCLI_CmdServiceEndpoints_err() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "endpoints", "test-service-0")

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceEndpoints_worksNoEndpoints() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "endpoints", "test-service-1")

	// Output:
	// Zenoss - no endpoints defined
}

func ExampleServicedCLI_CmdServiceEndpoints_works() {
	pipeStderr(InitServiceAPITest, "serviced", "service", "endpoints", "test-service-2")

	// Output:
	// Name    ServiceID         Endpoint         Purpose    Host       HostIP     HostPort    ContainerID     ContainerIP     ContainerPort
	// Zope    test-service-2    endpointName1    export     hostID1    hostIP1    10          containerID1    containerIP1    100
	// Zope    test-service-2    endpointName2    import     hostID2    hostIP2    20          containerID2    containerIP2    200
}
