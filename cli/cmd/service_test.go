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
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
)

const (
	NilService = "NilService"
)

var DefaultServiceAPITest = ServiceAPITest{
	errs:      make(map[string]error, 10),
	services:  DefaultTestServices,
	pools:     DefaultTestPools,
	hosts:     DefaultTestHosts,
	snapshots: DefaultTestSnapshots,
	endpoints: DefaultEndpoints,
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
		Commands: map[string]domain.Command{
			"hello": domain.Command{
				Command:         "echo hello world",
				Description:     "Just says 'hello world'.",
				CommitOnSuccess: false,
			},
			"goodbye": domain.Command{
				Command:         "echo goodbye world",
				Description:     "Just says 'goodbye world'.",
				CommitOnSuccess: false,
			},
		},
		Endpoints: []service.ServiceEndpoint{
			service.ServiceEndpoint{
				Application: "zproxy",
				Name:        "zproxy",
				PortList:    DefaultTestPublicEndpointPorts,
				PortNumber:  8080,
				Protocol:    "tcp",
				Purpose:     "export",
			},
			service.ServiceEndpoint{
				Application: "zope",
				Name:        "zope",
				PortNumber:  9080,
				Protocol:    "tcp",
				Purpose:     "import",
			},
		},
		ConfigFiles: map[string]servicedefinition.ConfigFile{
			"/etc/test.conf": servicedefinition.ConfigFile{
				Filename:    "/etc/test.conf",
				Owner:       "1001",
				Permissions: "600",
				Content:     "#-----test.conf\n\n# This is a test conf file.\n",
			},
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
		Context: map[string]interface{}{
			"home.name":  "Alphas",
			"away.name":  "Bravos",
			"home.score": 19,
			"away.score": 12,
		},
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
	ErrNoServiceFound = errors.New("no service found")
	ErrInvalidService = errors.New("invalid service")
	ErrCmdNotFound    = errors.New("command not found")
	ErrStub           = errors.New("stub for facade failed")
)

type ServiceAPITest struct {
	api.API
	errs      map[string]error
	services  []service.Service
	pools     []pool.ResourcePool
	hosts     []host.Host
	snapshots []dao.SnapshotInfo
	endpoints []applicationendpoint.EndpointReport
}

func InitServiceAPITest(args ...string) {
	c := New(DefaultServiceAPITest, utils.TestConfigReader(make(map[string]string)), MockLogControl{})
	c.exitDisabled = true
	c.Run(args)
}

func (t ServiceAPITest) GetAllServiceDetails() ([]service.ServiceDetails, error) {
	if t.errs["GetAllServiceDetails"] != nil {
		return nil, t.errs["GetAllServiceDetails"]
	}
	return servicesToServiceDetails(t.services), nil
}

func (t ServiceAPITest) ResolveServicePath(name string, noprefix bool) ([]service.ServiceDetails, error) {
	if t.errs["ResolveServicePath"] != nil {
		return nil, t.errs["ResolveServicePath"]
	}
	return serviceDetailsByName(strings.TrimRight(name, "/")), nil
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

func (t ServiceAPITest) GetServiceDetails(id string) (*service.ServiceDetails, error) {
	if t.errs["GetServiceDetails"] != nil {
		return nil, t.errs["GetServiceDetails"]
	}

	for i, s := range t.services {
		if s.ID == id {
			details := serviceToServiceDetails(t.services[i])
			return &details, nil
		}
	}
	return nil, nil
}

func (t ServiceAPITest) AddService(config api.ServiceConfig) (*service.ServiceDetails, error) {
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

	s := service.ServiceDetails{
		ID:              fmt.Sprintf("%s-%s-%s", config.Name, config.ParentServiceID, config.ImageID),
		ParentServiceID: config.ParentServiceID,
		Name:            config.Name,
		ImageID:         config.ImageID,
		//Endpoints:       endpoints,
		Startup:        config.Command,
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1, 1},
	}

	return &s, nil
}

func (t ServiceAPITest) RemoveService(id string) error {
	if t.errs["RemoveService"] != nil {
		return t.errs["RemoveService"]
	}
	return nil
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.ServiceDetails, error) {
	var svc service.Service

	if err := json.NewDecoder(reader).Decode(&svc); err != nil {
		return nil, ErrInvalidService
	}

	if _, err := t.GetService(svc.ID); err != nil {
		return nil, err
	}

	details := serviceToServiceDetails(svc)
	return &details, nil
}

func (t ServiceAPITest) UpdateServiceObj(svc service.Service) (*service.ServiceDetails, error) {
	if _, err := t.GetService(svc.ID); err != nil {
		return nil, err
	}

	details := serviceToServiceDetails(svc)
	return &details, nil
}

func servicesToServiceDetails(svcs []service.Service) []service.ServiceDetails {
	detailsList := []service.ServiceDetails{}
	for _, svc := range svcs {
		details := serviceToServiceDetails(svc)
		detailsList = append(detailsList, details)
	}
	return detailsList
}

func serviceDetailsByName(name string) []service.ServiceDetails {
	for _, svc := range DefaultTestServices {
		if svc.Name == name || svc.ID == name {
			return servicesToServiceDetails([]service.Service{svc})
		}
	}
	return []service.ServiceDetails{}
}

func serviceToServiceDetails(svc service.Service) service.ServiceDetails {
	details := service.ServiceDetails{
		ID:              svc.ID,
		Name:            svc.Name,
		Description:     svc.Description,
		PoolID:          svc.PoolID,
		ImageID:         svc.ImageID,
		ParentServiceID: svc.ParentServiceID,
		Instances:       svc.Instances,
		InstanceLimits:  svc.InstanceLimits,
		RAMCommitment:   svc.RAMCommitment,
		Startup:         svc.Startup,
		DeploymentID:    svc.DeploymentID,
		DesiredState:    svc.DesiredState,
		Launch:          svc.Launch,
	}
	return details
}

func (t ServiceAPITest) StartService(cfg api.SchedulerConfig) (int, error) {
	if t.errs["StartService"] != nil {
		return 0, t.errs["StartService"]
	}
	return len(cfg.ServiceIDs), nil
}

func (t ServiceAPITest) RestartService(cfg api.SchedulerConfig) (int, error) {
	if t.errs["RestartService"] != nil {
		return 0, t.errs["RestartService"]
	}
	return len(cfg.ServiceIDs), nil
}

func (t ServiceAPITest) StopServiceInstance(serviceID string, instanceID int) error {
	if s, err := t.GetService(serviceID); err != nil {
		return err
	} else if s == nil {
		return errors.New("service not found")
	} else if s.Instances < instanceID {
		return errors.New("service not found")
	}
	return nil
}

func (t ServiceAPITest) StopService(cfg api.SchedulerConfig) (int, error) {
	for _, sid := range cfg.ServiceIDs {
		if s, err := t.GetService(sid); err != nil {
			return 0, err
		} else if s == nil {
			return 0, ErrNoServiceFound
		}
	}
	return len(cfg.ServiceIDs), nil
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) error {
	if t.errs["AssignIP"] != nil {
		return t.errs["AssignIP"]
	}
	return nil
}

func (t ServiceAPITest) RemoveIP(args []string) error {
	if t.errs["RemoveIP"] != nil {
		return t.errs["RemoveIP"]
	}
	return nil
}

func (t ServiceAPITest) SetIP(config api.IPConfig) error {
	if t.errs["SetIP"] != nil {
		return t.errs["SetIP"]
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

func (t ServiceAPITest) ClearEmergency(serviceID string) (int, error) {
	if t.errs["ClearEmergency"] != nil {
		return 0, t.errs["ClearEmergency"]
	}
	return 1, nil
}

func TestServicedCLI_CmdServiceList_one(t *testing.T) {
	serviceID := "test-service-1"

	expected, err := DefaultServiceAPITest.GetService(serviceID)
	if err != nil {
		t.Fatal(err)
	}

	var actual service.Service
	output := captureStdout(func() { InitServiceAPITest("serviced", "service", "list", serviceID) })
	if err := json.Unmarshal(output, &actual); err != nil {
		t.Fatalf("error unmarshaling resource: %s", err)
	}

	// Did you remember to update Service.Equals?
	if !actual.Equals(expected) {
		t.Fatalf("\ngot:\n%+v\nwant:\n%+v", actual, expected)
	}
}

func TestServicedCLI_CmdServiceList_all(t *testing.T) {
	expected, err := DefaultServiceAPITest.GetAllServiceDetails()
	if err != nil {
		t.Fatal(err)
	}

	var actual []*service.ServiceDetails
	output := captureStdout(func() { InitServiceAPITest("serviced", "service", "list", "--verbose") })
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
	DefaultServiceAPITest.errs["GetAllServiceDetails"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetAllServiceDetails"] = nil }()
	DefaultServiceAPITest.errs["ResolveServicePath"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["ResolveServicePath"] = nil }()
	// Error retrieving service
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list", "test-service-0") })
	// Error retrieving all services
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list") })

	// Output:
	// invalid service
	// invalid service
}

func ExampleServicedCLI_CmdServiceList_err() {
	DefaultServiceAPITest.services = nil
	defer func() { DefaultServiceAPITest.services = DefaultTestServices }()
	// Service not found
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list", "test-service-0") })
	// No Services found
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list") })

	// Output:
	// service not found
	// no services found
}

func ExampleServicedCLI_CmdServiceList_complete() {
	InitServiceAPITest("serviced", "service", "list", "--generate-bash-completion")

	DefaultServiceAPITest.errs["GetAllServiceDetails"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetAllServiceDetails"] = nil }()
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
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "add", "--parent-id", "test-service-1", "test-service", "test-image", "bash -c lsof")
	})

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceAdd_missingParentArg() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "add", "test-service", "test-pool", "test-image", "bash -c lsof")
	})

	// Output:
	// Must specify a parent service ID
}

func ExampleServicedCLI_CmdServiceAdd_parentNotFound() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "add", "--parent-id", "test-parent", NilService, "test-image", "bash -c lsof")
	})

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
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches

}

func ExampleServicedCLI_CmdServiceRemove_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "remove", "test-service-0") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceRemove_failed() {
	DefaultServiceAPITest.errs["RemoveService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["RemoveService"] = nil }()

	pipeStderr(func() { InitServiceAPITest("serviced", "service", "remove", "test-service-1") })

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
	//    --editor, -e 		Editor used to update the service definition
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceEdit_fail() {
	DefaultServiceAPITest.errs["GetAllServiceDetails"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["GetAllServiceDetails"] = nil }()
	DefaultServiceAPITest.errs["ResolveServicePath"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["ResolveServicePath"] = nil }()
	// Failed to get service
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "edit", "test-service-0") })
	// TODO: Failed to update service

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceEdit_err() {
	// Service not found
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "edit", "test-service-0") })
	// TODO: Nil Service after update

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceConfigList_usage() {
	InitServiceAPITest("serviced", "service", "config", "list")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    list - List all config files for a given service, or the contents of one named file
	//
	// USAGE:
	//    command list [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service config list SERVICEID [FILENAME]
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceConfigList() {
	InitServiceAPITest("serviced", "service", "config", "list", "test-service-1")
	// Output:
	// {
	//    "ConfigFiles": [
	//      "/etc/test.conf"
	//    ]
	//  }
}

func ExampleServicedCLI_CmdServiceConfigListSingle() {
	InitServiceAPITest("serviced", "service", "config", "list", "test-service-1", "/etc/test.conf")
	// Output:
	// #-----test.conf
	//
	// # This is a test conf file.
	//
}

func ExampleServicedCLI_CmdServiceConfigList_noservice() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "config", "list", "test-service-0") })
	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceConfigListSingle_nofile() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "config", "list", "test-service-1", "/etc/nothere.conf")
	})
	// Output:
	// Config file /etc/nothere.conf not found.
}

func ExampleServicedCLI_CmdServiceConfigEdit_usage() {
	InitServiceAPITest("serviced", "service", "config", "edit")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    edit - Edit one config file for a given service
	//
	// USAGE:
	//    command edit [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service config edit SERVICEID FILENAME
	//
	// OPTIONS:
	//    --editor, -e 		Editor used to update the config file
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceConfigEdit_noservice() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "config", "edit", "test-service-0", "/etc/nothere.conf")
	})
	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceConfigEdit_nofile() {
	// File not found
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "config", "edit", "test-service-1", "/etc/nothere.conf")
	})

	// Output:
	// Config file /etc/nothere.conf not found.
}

func ExampleServicedCLI_CmdServiceConfigEdit() {
	// Hard to test that an editor was opened.
	InitServiceAPITest("serviced", "service", "config", "edit", "test-service-1", "/etc/test.conf")
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
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceAssignIPs_fail() {
	DefaultServiceAPITest.errs["AssignIP"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["AssignIP"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "assign-ip", "test-service-3") })

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceAssignIPs_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "assign-ip", "test-service-0", "100.99.88.1") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceRemoveIPs() {
	//Remove assignments
	InitServiceAPITest("serviced", "service", "remove-ip", "test-service-1", "test-endpoint")
	// Output:
	//
}

func ExampleServicedCLI_CmdServiceRemoveIPs_usage() {
	InitServiceAPITest("serviced", "service", "remove-ip")

	// Incorrect Usage.

	// NAME:
	//    remove-ip - Remove the IP assignment of a service's endpoints

	// USAGE:
	//    ommand remove-ip [command options] [arguments...]

	// DESCRIPTION:
	//    serviced service remove-ip <SERVICEID> <ENDPOINTNAME>

	// OPTIONS:

}

func ExampleServicedCLI_CmdServiceRemoveIPs_fail() {
	DefaultServiceAPITest.errs["RemoveIP"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["RemoveIP"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "remove-ip", "test-service-3") })

	// Incorrect Usage.

	// NAME:
	//    remove-ip - Remove the IP assignment of a service's endpoints

	// USAGE:
	//    ommand remove-ip [command options] [arguments...]

	// DESCRIPTION:
	//    serviced service remove-ip <SERVICEID> <ENDPOINTNAME>

	// OPTIONS:
}

func ExampleServicedCLI_CmdServiceRemoveIPs_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "remove-ip", "test-service-0", "test-endpoint-1") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceSetIPs() {
	InitServiceAPITest("serviced", "service", "set-ip", "test-service-1", "test-endpoint", "127.0.0.1", "--port=8080", "--proto=tcp")

	// Output:
	//
}

func ExampleServicedCLI_CmdServiceSetIPs_usage() {
	InitServiceAPITest("serviced", "service", "set-ip")

	// Incorrect Usage.

	//NAME:
	//   set-ip - Setting an IP address to a service's endpoints requiring an explicit IP address. If ip is not provided it does an automatic assignment

	// USAGE:
	//    command set-ip [command options] [arguments...]

	// DESCRIPTION:
	//    serviced service set-ip <SERVICEID> <ENDPOINTNAME> [ADDRESS] [--port=PORT] [--proto=PROTOCOL]

	// OPTIONS:
	//    --port '0'   determine the port your service will use
	//    --proto      determine the port protocol your service will use

}

func ExampleServicedCLI_CmdServiceSetIPs_fail() {
	DefaultServiceAPITest.errs["SetIP"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["SetIP"] = nil }()
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "set-ip", "test-service-2", "test-endpoint", "127.0.0.1", "--port=8080", "--proto=tcp")
	})

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceSetIPs_err() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "set-ip", "test-service-0", "test-endpoint", "127.0.0.1")
	})

	// Please specify the valid port number.

	// NAME:
	//    set-ip - Setting an IP address to a service's endpoints requiring an explicit IP address. If ip is not provided it does an automatic assignment

	// USAGE:
	//    command set-ip [command options] [arguments...]

	// DESCRIPTION:
	//    serviced service set-ip <SERVICEID> <ENDPOINTNAME> [ADDRESS] [--port=PORT] [--proto=PROTOCOL]

	// OPTIONS:
	//    --port '0'   determine the port your service will use
	//    --proto      determine the port protocol your service will use
}

func ExampleServicedCLI_CmdServiceStart_usage() {
	InitServiceAPITest("serviced", "service", "start")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    start - Starts one or more services
	//
	// USAGE:
	//    command start [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service start SERVICEID ...
	//
	// OPTIONS:
	//    --auto-launch		Recursively schedules child services
	//    --sync, -s			Schedules services synchronously
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceStart_fail() {
	DefaultServiceAPITest.errs["StartService"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["StartService"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "start", "test-service-1") })

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceStart_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "start", "test-service-0") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStart() {
	InitServiceAPITest("serviced", "service", "start", "test-service-2")
	InitServiceAPITest("serviced", "service", "start", "test-service-1", "test-service-2")

	// Output:
	// Scheduled 1 service(s) to start
	// Scheduled 2 service(s) to start
}

func ExampleServicedCLI_CmdServiceRestart_usage() {
	InitServiceAPITest("serviced", "service", "restart")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    restart - Restarts one or more services
	//
	// USAGE:
	//    command restart [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service restart { SERVICEID | INSTANCEID } ...
	//
	// OPTIONS:
	//    --auto-launch		Recursively schedules child services
	//    --sync, -s			Schedules services synchronously
	//    --rebalance			Stops all instances before restarting them, instead of performing a rolling restart
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceRestart_fail() {
	DefaultServiceAPITest.errs["ResolveServicePath"] = ErrInvalidService
	defer func() { DefaultServiceAPITest.errs["ResolveServicePath"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "restart", "test-service-1") })

	// Output:
	// invalid service
}

func ExampleServicedCLI_CmdServiceRestart_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "restart", "test-service-0") })    // Non-existant service
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "restart", "test-service-3/4") })  // Non-existant instance
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "restart", "test-service-3/a") })  // Non-numeric instance number
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "restart", "test-service-3/0b") }) // Non-numeric instance number

	// Output:
	// service not found
	// service not found
	// service not found
	// service not found
}

func ExampleServicedCLI_CmdServiceRestart() {
	InitServiceAPITest("serviced", "service", "restart", "test-service-2")
	InitServiceAPITest("serviced", "service", "restart", "test-service-3/1")                     // Specific instance
	InitServiceAPITest("serviced", "service", "restart", "test-service-2", "test-service-3/1")   // Both
	InitServiceAPITest("serviced", "service", "restart", "test-service-2", "test-service-3")     // 2 services
	InitServiceAPITest("serviced", "service", "restart", "test-service-3/0", "test-service-3/1") // 2 instances

	// Output:
	// Restarting 1 service(s)
	// Restarting instance test-service-3/1
	// Restarting 1 service(s)
	// Restarting instance test-service-3/1
	// Restarting 2 service(s)
	// Restarting instance test-service-3/0
	// Restarting instance test-service-3/1

}

func ExampleServicedCLI_CmdServiceStop_usage() {
	InitServiceAPITest("serviced", "service", "stop")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    stop - Stops one or more services
	//
	// USAGE:
	//    command stop [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service stop SERVICEID ...
	//
	// OPTIONS:
	//    --auto-launch		Recursively schedules child services
	//    --sync, -s			Schedules services synchronously
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceStop_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "stop", "test-service-0") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceStop() {
	InitServiceAPITest("serviced", "service", "stop", "test-service-2")
	InitServiceAPITest("serviced", "service", "stop", "test-service-1", "test-service-2")

	// Output:
	// Scheduled 1 service(s) to stop
	// Scheduled 2 service(s) to stop
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
	//    --logstash				forward service logs via filebeat
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
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "shell", "test-service-0", "some", "command") })

	// Output:
	// service not found
	// exit code 1
}

/*func ExampleServicedCLI_CmdServiceRun_list() {
	output := captureStdout(func(){InitServiceAPITest( "serviced", "service", "run", "test-service-1")})
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
func ExampleServicedCLI_CmdServiceRun_exec_aloha() {
	InitServiceAPITest("serviced", "service", "run", "-i", "test-service-1", "aloha")

	// Output:
	// Command "aloha" not available.
	// Available commands for Zenoss:
	//     goodbye               Just says 'goodbye world'.
	//     hello                 Just says 'hello world'.
}

func ExampleServicedCLI_CmdServiceRun_exec_hello() {
	InitServiceAPITest("serviced", "service", "run", "-i", "test-service-1", "hello")

	// Output:
	// echo hello world
}

func ExampleServicedCLI_CmdServiceRun_help() {
	InitServiceAPITest("serviced", "service", "run", "-i", "test-service-1", "help")

	// Output:
	// Available commands for Zenoss:
	//     goodbye               Just says 'goodbye world'.
	//     hello                 Just says 'hello world'.
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
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "run", "test-service-0", "goodbye") })

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
	output := captureStdout(func() { InitServiceAPITest("serviced", "service", "list-snapshots", "test-service-1", "-t") })
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
	output := captureStdout(func() { InitServiceAPITest("serviced", "service", "list-snapshots", "test-service-1", "--show-tags") })
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
	//    --show-tags, -t		shows the tags associated with each snapshot
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServicedCLI_CmdServiceListSnapshots_fail() {
	DefaultServiceAPITest.errs["GetSnapshotsByServiceID"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["GetSnapshotsByServiceID"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list-snapshots", "test-service-1") })

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceListSnapshots_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "list-snapshots", "test-service-3") })

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
	//    --description, -d 		a description of the snapshot
	//    --tag, -t 			a unique tag for the snapshot
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches

}

func ExampleServicedCLI_CmdServiceSnapshot_fail() {
	DefaultServiceAPITest.errs["AddSnapshot"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["AddSnapshot"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "snapshot", "test-service-1") })

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceSnapshot_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "snapshot", "test-service-0") })

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
	//    --imports, -i		include only imported endpoints
	//    --all, -a			include all endpoints (imports and exports)
	//    --verify, -v			verify endpoints
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches

}

func ExampleServicedCLI_CmdServiceEndpoints_err() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "endpoints", "test-service-0") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceEndpoints_worksNoEndpoints() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "endpoints", "test-service-1") })

	// Output:
	// Zenoss - no endpoints defined
}

func ExampleServicedCLI_CmdServiceEndpoints_works() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "endpoints", "test-service-2") })

	// Output:
	// Name    ServiceID         Endpoint         Purpose    Host       HostIP     HostPort    ContainerID     ContainerIP     ContainerPort
	// Zope    test-service-2    endpointName1    export     hostID1    hostIP1    10          containerID1    containerIP1    100
	// Zope    test-service-2    endpointName2    import     hostID2    hostIP2    20          containerID2    containerIP2    200
}

func ExampleServicedCLI_CmdServiceClearEmergency_works() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "clear-emergency", "test-service-1") })

	// Output:
	// Cleared emergency status for 1 services
}

func ExampleServicedCLI_CmdServiceClearEmergency_errNoService() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "clear-emergency", "test-service-0") })

	// Output:
	// service not found
}

func ExampleServicedCLI_CmdServiceClearEmergency_err() {
	DefaultServiceAPITest.errs["ClearEmergency"] = ErrStub
	defer func() { DefaultServiceAPITest.errs["ClearEmergency"] = nil }()
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "clear-emergency", "test-service-1") })

	// Output:
	// stub for facade failed
}

func ExampleServicedCLI_CmdServiceClearEmergency_usage() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "clear-emergency") })

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    clear-emergency - Clears the 'emergency shutdown' state for a service and all child services
	//
	// USAGE:
	//    command clear-emergency [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service clear-emergency { SERVICEID | SERVICENAME | DEPLOYMENTID/...PARENTNAME.../SERVICENAME }
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceTune_usage() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune") })
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    tune - Adjust launch mode, instance count, RAM commitment, or RAM threshold for a service
	//
	// USAGE:
	//    command tune [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service tune SERVICEID
	//
	// OPTIONS:
	//    --launchMode 		Launch mode for this service (auto, manual)
	//    --instances '0'		Instance count for this service
	//    --ramCommitment 		RAM Commitment for this service
	//    --ramThreshold 		RAM Threshold for this service
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceTune_noservice() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune", "test-service-0") })
	// Output:
	// service not found
}

func ExampleServiceCLI_CmdServiceTune_nokwargs() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune", "test-service-1") })
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    tune - Adjust launch mode, instance count, RAM commitment, or RAM threshold for a service
	//
	// USAGE:
	//    command tune [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service tune SERVICEID
	//
	// OPTIONS:
	//    --launchMode 		Launch mode for this service (auto, manual)
	//    --instances '0'		Instance count for this service
	//    --ramCommitment 		RAM Commitment for this service
	//    --ramThreshold 		RAM Threshold for this service
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceTune_nochanges() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune", "test-service-3", "--instances=2") })
	// Output:
	// Service already reflects desired configured - no changes made
}

func ExampleServiceCLI_CmdServiceTune_toomany() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune", "test-service-2", "--instances=5") })
	// Output:
	// test-service-2
}

func ExampleServiceCLI_CmdServiceTune_invalidlaunchmode() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "tune", "test-service-1", "--launchMode=incorrect") })
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    tune - Adjust launch mode, instance count, RAM commitment, or RAM threshold for a service
	//
	// USAGE:
	//    command tune [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service tune SERVICEID
	//
	// OPTIONS:
	//    --launchMode 		Launch mode for this service (auto, manual)
	//    --instances '0'		Instance count for this service
	//    --ramCommitment 		RAM Commitment for this service
	//    --ramThreshold 		RAM Threshold for this service
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceTune_launch() {
	InitServiceAPITest("serviced", "service", "tune", "test-service-1", "--launchMode=manual")
	InitServiceAPITest("serviced", "service", "tune", "test-service-3", "--launchMode=auto")
}

func ExampleServiceCLI_CmdServiceTune_commitment() {
	InitServiceAPITest("serviced", "service", "tune", "test-service-1", "--ramCommitment=256M")
}

func ExampleServiceCLI_CmdServiceTune_threshold() {
	InitServiceAPITest("serviced", "service", "tune", "test-service-1", "--ramThreshold='80%'")
}

func ExampleServiceCLI_CmdServiceVariableList_usage() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "list") })
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    list - List one or all config variables and their values for a given service
	//
	// USAGE:
	//    command list [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable list SERVICEID
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableList_noservice() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "list", "test-service-0") })
	// Output:
	// service not found
}

func ExampleServiceCLI_CmdServiceVariableList() {
	InitServiceAPITest("serviced", "service", "variable", "list", "test-service-2")
	// Output:
	// away.name Bravos
	// away.score 12
	// home.name Alphas
	// home.score 19
}

func ExampleServiceCLI_CmdServiceVariableGet_usage() {
	InitServiceAPITest("serviced", "service", "variable", "get")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    get - Find the value of a config variable for a service
	//
	// USAGE:
	//    command get [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable get SERVICEID VARIABLE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableGet_noservice() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "get", "test-service-0", "wickets") })
	// Output:
	// service not found
}

func ExampleServiceCLI_CmdServiceVariableGet_badvariable() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "get", "test-service-2", "mallets") })
	// Output:
	// Variable mallets not found.
}

func ExampleServiceCLI_CmdServiceVariableGet() {
	InitServiceAPITest("serviced", "service", "variable", "get", "test-service-2", "home.name")
	// Output:
	// Alphas
}

func ExampleServiceCLI_CmdServiceVariableSet_usage() {
	InitServiceAPITest("serviced", "service", "variable", "set")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    set - Add or update one variable's value for a given service
	//
	// USAGE:
	//    command set [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable set SERVICEID VARIABLE VALUE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableSet_noservice() {
	pipeStderr(func() {
		InitServiceAPITest("serviced", "service", "variable", "set", "test-service-0", "wickets", "12")
	})
	// Output:
	// service not found
}

func ExampleServiceCLI_CmdServiceVariableSet_novariable() {
	InitServiceAPITest("serviced", "service", "variable", "set", "test-service-2")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    set - Add or update one variable's value for a given service
	//
	// USAGE:
	//    command set [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable set SERVICEID VARIABLE VALUE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableSet_novalue() {
	InitServiceAPITest("serviced", "service", "variable", "set", "test-service-2", "wickets")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    set - Add or update one variable's value for a given service
	//
	// USAGE:
	//    command set [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable set SERVICEID VARIABLE VALUE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableSet() {
	InitServiceAPITest("serviced", "service", "variable", "set", "test-service-2", "wickets", "9")
	// Output:
	// test-service-2
}

func ExampleServiceCLI_CmdServiceVariableUnset_usage() {
	InitServiceAPITest("serviced", "service", "variable", "unset")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    unset - Remove a variable from a given service
	//
	// USAGE:
	//    command unset [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable unset SERVICEID VARIABLE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableUnset_noservice() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "unset", "test-service-0", "wickets") })
	// Output:
	// service not found
}

func ExampleServiceCLI_CmdServiceVariableUnset_novariable() {
	InitServiceAPITest("serviced", "service", "variable", "unset", "test-service-2")
	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    unset - Remove a variable from a given service
	//
	// USAGE:
	//    command unset [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced service variable unset SERVICEID VARIABLE
	//
	// OPTIONS:
	//    --no-prefix-match, --np	Make SERVICEID matches on name strict 'ends with' matches
}

func ExampleServiceCLI_CmdServiceVariableUnset_badvariable() {
	pipeStderr(func() { InitServiceAPITest("serviced", "service", "variable", "unset", "test-service-2", "mallets") })
	// Output:
	// Variable mallets not found.
}

func ExampleServiceCLI_CmdServiceVariableUnset() {
	InitServiceAPITest("serviced", "service", "variable", "unset", "test-service-2", "home.name")
	// Output:
	// test-service-2
}
