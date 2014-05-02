package cmd

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/zenoss/serviced/cli/api"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain"
	"github.com/zenoss/serviced/domain/host"
)

const (
	IPAddressNotFound = "IP ADDRESS NOT FOUND"
)

var DefaultServiceAPITest = ServiceAPITest{
	services:  DefaultTestServices,
	snapshots: DefaultTestSnapshots,
}

var DefaultTestServices = []*service.Service{
	{
		Id:             "test-service-1",
		Name:           "Zenoss",
		Startup:        "startup command 1",
		Instances:      0,
		InstanceLimits: domain.MinMax{0, 0},
		ImageId:        "quay.io/zenossinc/tenantid1-core5x",
		PoolId:         "default",
		DesiredState:   1,
		Launch:         "auto",
		DeploymentId:   "Zenoss-resmgr",
		Runs: map[string]string{
			"hello":   "echo hello world",
			"goodbye": "echo goodbye world",
		},
	}, {
		Id:             "test-service-2",
		Name:           "Zope",
		Startup:        "startup command 2",
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1},
		ImageId:        "quay.io/zenossinc/tenantid2-core5x",
		PoolId:         "default",
		DesiredState:   1,
		Launch:         "auto",
		DeploymentId:   "Zenoss-core",
	}, {
		Id:             "test-service-3",
		Name:           "zencommand",
		Startup:        "startup command 3",
		Instances:      2,
		InstanceLimits: domain.MinMax{2, 2},
		ImageId:        "quay.io/zenossinc/tenantid1-opentsdb",
		PoolId:         "remote",
		DesiredState:   1,
		Launch:         "manual",
		DeploymentId:   "Zenoss-core",
	},
}

var (
	ErrNoServiceFound = errors.New("no service found")
	ErrInvalidService = errors.New("invalid service")
	ErrCmdNotFound    = errors.New("command not found")
)

type ServiceAPITest struct {
	api.API
	services  []*service.Service
	snapshots []string
}

func InitServiceAPITest(args ...string) {
	New(DefaultServiceAPITest).Run(args)
}

func (t ServiceAPITest) GetServices() ([]*service.Service, error) {
	return t.services, nil
}

func (t ServiceAPITest) GetService(id string) (*service.Service, error) {
	for i, s := range t.services {
		if s.Id == id {
			return t.services[i], nil
		}
	}

	return nil, ErrNoServiceFound
}

func (t ServiceAPITest) AddService(config api.ServiceConfig) (*service.Service, error) {
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
		Id:             fmt.Sprintf("%s-%s-%s", config.Name, config.PoolID, config.ImageID),
		Name:           config.Name,
		PoolId:         config.PoolID,
		ImageId:        config.ImageID,
		Endpoints:      endpoints,
		Startup:        config.Command,
		Instances:      1,
		InstanceLimits: domain.MinMax{1, 1},
	}

	return &s, nil
}

func (t ServiceAPITest) RemoveService(id string) error {
	_, err := t.GetService(id)
	return err
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.Service, error) {
	var s service.Service

	if err := json.NewDecoder(reader).Decode(&s); err != nil {
		return nil, ErrInvalidService
	}

	if _, err := t.GetService(s.Id); err != nil {
		return nil, err
	}

	return &s, nil
}

func (t ServiceAPITest) StartService(id string) (*host.Host, error) {
	if _, err := t.GetService(id); err != nil {
		return nil, err
	}

	h := host.Host{
		ID: fmt.Sprintf("%s-host", id),
	}

	return &h, nil
}

func (t ServiceAPITest) StopService(id string) error {
	if _, err := t.GetService(id); err != nil {
		return err
	}

	return nil
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) (string, error) {
	if _, err := t.GetService(config.ServiceID); err != nil {
		return "", err
	}

	switch config.IPAddress {
	case IPAddressNotFound:
		return "", nil
	case "":
		return "0.0.0.0", nil
	default:
		return config.IPAddress, nil
	}
}

func (t ServiceAPITest) StartProxy(config api.ProxyConfig) error {
	return nil
}

func (t ServiceAPITest) StartShell(config api.ShellConfig) error {
	return nil
}

func (t ServiceAPITest) RunShell(config api.ShellConfig) error {
	s, err := t.GetService(config.ServiceID)
	if err != nil {
		return err
	}

	command, ok := s.Runs[config.Command]
	if !ok {
		return ErrCmdNotFound
	}

	fmt.Printf("%s %s\n", command, strings.Join(config.Args, " "))
	return nil
}

func (t ServiceAPITest) GetSnapshots() ([]string, error) {
	return t.snapshots, nil
}

func (t ServiceAPITest) GetSnapshotsByServiceID(id string) ([]string, error) {
	var snapshots []string
	for _, s := range t.snapshots {
		if strings.HasPrefix(s, id) {
			snapshots = append(snapshots, s)
		}
	}

	return snapshots, nil
}

func (t ServiceAPITest) AddSnapshot(id string) (string, error) {
	return fmt.Sprintf("%s-snapshot", id), nil
}

func ExampleServicedCli_cmdServiceList() {
	InitServiceAPITest("serviced", "service", "list", "-v")

	// Output:
	// [
	//    {
	//      "Id": "test-service-1",
	//      "Name": "Zenoss",
	//      "Context": "",
	//      "Startup": "startup command 1",
	//      "Description": "",
	//      "Tags": null,
	//      "ConfigFiles": null,
	//      "Instances": 0,
	//      "InstanceLimits": {
	//        "Min": 0,
	//        "Max": 0
	//      },
	//      "ImageId": "quay.io/zenossinc/tenantid1-core5x",
	//      "PoolId": "default",
	//      "DesiredState": 1,
	//      "HostPolicy": "",
	//      "Hostname": "",
	//      "Privileged": false,
	//      "Launch": "auto",
	//      "Endpoints": null,
	//      "Tasks": null,
	//      "ParentServiceId": "",
	//      "Volumes": null,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "DeploymentId": "Zenoss-resmgr",
	//      "DisableImage": false,
	//      "LogConfigs": null,
	//      "Snapshot": {
	//        "Pause": "",
	//        "Resume": ""
	//      },
	//      "Runs": {
	//        "goodbye": "echo goodbye world",
	//        "hello": "echo hello world"
	//      },
	//      "RAMCommitment": 0,
	//      "Actions": null,
	//      "HealthChecks": null
	//    },
	//    {
	//      "Id": "test-service-2",
	//      "Name": "Zope",
	//      "Context": "",
	//      "Startup": "startup command 2",
	//      "Description": "",
	//      "Tags": null,
	//      "ConfigFiles": null,
	//      "Instances": 1,
	//      "InstanceLimits": {
	//        "Min": 1,
	//        "Max": 1
	//      },
	//      "ImageId": "quay.io/zenossinc/tenantid2-core5x",
	//      "PoolId": "default",
	//      "DesiredState": 1,
	//      "HostPolicy": "",
	//      "Hostname": "",
	//      "Privileged": false,
	//      "Launch": "auto",
	//      "Endpoints": null,
	//      "Tasks": null,
	//      "ParentServiceId": "",
	//      "Volumes": null,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "DeploymentId": "Zenoss-core",
	//      "DisableImage": false,
	//      "LogConfigs": null,
	//      "Snapshot": {
	//        "Pause": "",
	//        "Resume": ""
	//      },
	//      "Runs": null,
	//      "RAMCommitment": 0,
	//      "Actions": null,
	//      "HealthChecks": null
	//    },
	//    {
	//      "Id": "test-service-3",
	//      "Name": "zencommand",
	//      "Context": "",
	//      "Startup": "startup command 3",
	//      "Description": "",
	//      "Tags": null,
	//      "ConfigFiles": null,
	//      "Instances": 2,
	//      "InstanceLimits": {
	//        "Min": 2,
	//        "Max": 2
	//      },
	//      "ImageId": "quay.io/zenossinc/tenantid1-opentsdb",
	//      "PoolId": "remote",
	//      "DesiredState": 1,
	//      "HostPolicy": "",
	//      "Hostname": "",
	//      "Privileged": false,
	//      "Launch": "manual",
	//      "Endpoints": null,
	//      "Tasks": null,
	//      "ParentServiceId": "",
	//      "Volumes": null,
	//      "CreatedAt": "0001-01-01T00:00:00Z",
	//      "UpdatedAt": "0001-01-01T00:00:00Z",
	//      "DeploymentId": "Zenoss-core",
	//      "DisableImage": false,
	//      "LogConfigs": null,
	//      "Snapshot": {
	//        "Pause": "",
	//        "Resume": ""
	//      },
	//      "Runs": null,
	//      "RAMCommitment": 0,
	//      "Actions": null,
	//      "HealthChecks": null
	//    }
	//  ]
}

func ExampleServicedCli_cmdServiceAdd() {
	InitServiceAPITest("serviced", "service", "add", "test-service", "test-pool", "test-image", "bash")

	// Output:
	// test-service-test-pool-test-image
}

func ExampleServicedCli_cmdServiceRemove() {
	InitServiceAPITest("serviced", "service", "rm", "test-service-1", "test-service-0")

	// Output:
	// test-service-1
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
}

func ExampleServicedCli_cmdServiceEdit() {
	InitServiceAPITest("serviced", "service", "edit", "test-service-1")
}

func ExampleServicedCli_cmdServiceAssignIPs() {
	// Service does not exist
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-0")
	// IP Address not returned
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-3", IPAddressNotFound)
	// Auto-assignment
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-1")
	// Manual-assignment
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-2", "127.0.0.1")

	// Output:
	// 0.0.0.0
	// 127.0.0.1
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

func ExampleServicedCli_cmdServiceStart() {
	InitServiceAPITest("serviced", "service", "start", "test-service-1")

	// Output:
	// Service scheduled to start on host: test-service-1-host
}

func ExampleServicedCli_cmdServiceStop() {
	InitServiceAPITest("serviced", "service", "stop", "test-service-2")

	// Output:
	// Service scheduled to stop.
}

func ExampleServicedCli_cmdServiceProxy() {
}

func ExampleServicedCli_cmdServiceShell() {
}

func ExampleServicedCli_cmdServiceRun_list() {
	InitServiceAPITest("serviced", "service", "run", "test-service-0")
	InitServiceAPITest("serviced", "service", "run", "test-service-1")

	// Output:
	// hello
	// goodbye
}

func ExampleServicedCli_cmdServiceRun_exec() {
	InitServiceAPITest("serviced", "service", "run", "test-service-1", "notfound")
	InitServiceAPITest("serviced", "service", "run", "test-service-1", "hello", "-i")

	// Output:
	// echo hello world -i
}

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
	//    serviced service run SERVICEID [COMMAND]
	//
	// OPTIONS:
	//    --saveas, -s 	saves the service instance with the given name
	//    --interactive, -i	runs the service instance as a tty
}
