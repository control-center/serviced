package cmd

import (
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/serviced/api"
)

var DefaultServiceAPITest = ServiceAPITest{services: DefaultTestServices}

var DefaultTestServices = []service.Service{
	{
		ID:           "test-service-id-1",
		Name:         "Zenoss",
		StartUp:      "startup command 1",
		Instances:    0,
		ImageID:      "quay.io/zenossinc/tenantid1-core5x",
		PoolID:       "default",
		DesiredState: 1,
		Launch:       "auto",
		DeploymentID: "Zenoss-resmgr",
		Runs: map[string]string{
			"hello":   "echo hello world",
			"goodbye": "echo goodbye world",
		},
	}, {
		ID:           "test-service-id-2",
		Name:         "Zope",
		StartUp:      "startup command 2",
		Instances:    1,
		ImageID:      "quay.io/zenossinc/tenantid2-core5x",
		PoolID:       "default",
		DesiredState: 1,
		Launch:       "auto",
		DeploymentID: "Zenoss-core",
	}, {
		ID:           "test-service-id-3",
		Name:         "zencommand",
		StartUp:      "startup command 3",
		Instances:    2,
		ImageID:      "quay.io/zenossinc/tenantid1-opentsdb",
		PoolID:       "remote",
		DesiredState: 1,
		Launch:       "manual",
		DeploymentID: "Zenoss-core",
	},
}

var (
	ErrNoServiceFound = errors.New("no service found")
	ErrInvalidService = errors.New("invalid service")
)

type ServiceAPITest struct {
	api.API
	services service.Service
}

func InitServiceAPITest(args ...string) {
	New(DefaultServiceAPITest).Run(args)
}

func (t ServiceAPITest) ListServices() ([]service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) GetService(id string) (*service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) AddService(config api.ServiceConfig) (*service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) RemoveService(id string) (*service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) StartService(id string) error {
	return nil
}

func (t ServiceAPITest) StopService(id string) error {
	return nil
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) (*host.HostIPResource, error) {
	return nil, nil
}

func (t ServiceAPITest) StartProxy(config api.ProxyConfig) {
	return
}

func (t ServiceAPITest) StartShell(config api.ShellConfig) (*shell.Command, error) {
	return nil, nil
}

func (t ServiceAPITest) ListSnapshots(id string) ([]string, error) {
	return nil, nil
}

func (t ServiceAPITest) AddSnapshot(id string) (string, error) {
	return "", nil
}

func ExampleServicedCli_cmdServiceList() {
	InitServiceAPITest("serviced", "service", "list", "--verbose")

	// Output:
	//
}

func ExampleServicedCli_cmdServiceAdd() {
	InitServiceAPITest("serviced", "service", "add", "test-service", "someimage", "somecommand")

	// Output:
	//
}

func ExampleServicedCli_cmdServiceRemove() {
	InitServiceAPITest("serviced", "service", "remove", "test-service-id-1")

	// Output:
	// test-service-id-1
}

func ExampleServicedCli_cmdServiceEdit() {
	InitServiceAPITest("serviced", "service", "edit", "test-service-id-2")
}

func ExampleServicedCli_cmdServiceAutoIPs() {
	InitServiceAPITest("serviced", "service", "assign-ip", "test-service-id-1", "10.0.0.1")

	// Output:
	// 10.0.0.1
}

func ExampleServicedCli_cmdServiceStart() {
	InitServiceAPITest("serviced", "service", "start", "test-service-id-2")
}

func ExampleServicedCli_cmdServiceStop() {
	InitServiceAPITest("serviced", "service", "stop", "test-service-id-2")
}

func ExampleServicedCli_cmdServiceProxy() {
	InitServiceAPITest("serviced", "service", "proxy", "test-service-id-1", "greet")

	// Output:
	// greet
}

func ExampleServicedCli_cmdServiceShell() {
	InitServiceAPITest("serviced", "service", "shell", "test-service-id-1", "echo", "hello world")

	// Output:
	// hello world
}

func ExampleServicedCli_cmdServiceRun_list() {
	InitServiceAPITest("serviced", "service", "run", "test-service-id-1")

	// Output:
	// hello
	// goodbye
}

func ExampleServicedCli_cmdServiceRun_exec() {
	InitServiceAPITest("serviced", "service", "run", "test-service-id-1", "hello")

	// Output:
	// hello world
}
