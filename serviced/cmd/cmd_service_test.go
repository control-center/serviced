package cmd

import (
	"errors"
	"io"

	service "github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/host"
	"github.com/zenoss/serviced/serviced/api"
)

var DefaultServiceAPITest = ServiceAPITest{services: DefaultTestServices}

var DefaultTestServices = []*service.Service{
	{
		Id:           "test-service-id-1",
		Name:         "Zenoss",
		Startup:      "startup command 1",
		Instances:    0,
		ImageId:      "quay.io/zenossinc/tenantid1-core5x",
		PoolId:       "default",
		DesiredState: 1,
		Launch:       "auto",
		DeploymentId: "Zenoss-resmgr",
		Runs: map[string]string{
			"hello":   "echo hello world",
			"goodbye": "echo goodbye world",
		},
	}, {
		Id:           "test-service-id-2",
		Name:         "Zope",
		Startup:      "startup command 2",
		Instances:    1,
		ImageId:      "quay.io/zenossinc/tenantid2-core5x",
		PoolId:       "default",
		DesiredState: 1,
		Launch:       "auto",
		DeploymentId: "Zenoss-core",
	}, {
		Id:           "test-service-id-3",
		Name:         "zencommand",
		Startup:      "startup command 3",
		Instances:    2,
		ImageId:      "quay.io/zenossinc/tenantid1-opentsdb",
		PoolId:       "remote",
		DesiredState: 1,
		Launch:       "manual",
		DeploymentId: "Zenoss-core",
	},
}

var (
	ErrNoServiceFound = errors.New("no service found")
	ErrInvalidService = errors.New("invalid service")
)

type ServiceAPITest struct {
	api.API
	services []*service.Service
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

func (t ServiceAPITest) RemoveService(id string) error {
	return nil
}

func (t ServiceAPITest) UpdateService(reader io.Reader) (*service.Service, error) {
	return nil, nil
}

func (t ServiceAPITest) StartService(id string) (*host.Host, error) {
	return nil, nil
}

func (t ServiceAPITest) StopService(id string) error {
	return nil
}

func (t ServiceAPITest) AssignIP(config api.IPConfig) ([]service.AddressAssignment, error) {
	return nil, nil
}

func (t ServiceAPITest) StartProxy(config api.ProxyConfig) error {
	return nil
}

func (t ServiceAPITest) StartShell(config api.ShellConfig) error {
	return nil
}

func (t ServiceAPITest) RunShell(config api.ShellConfig) error {
	return nil
}

func (t ServiceAPITest) GetSnapshots() ([]string, error) {
	return nil, nil
}

func (t ServiceAPITest) GetSnapshotsByServiceID(id string) ([]string, error) {
	return nil, nil
}

func (t ServiceAPITest) AddSnapshot(id string) (string, error) {
	return "", nil
}

func ExampleServicedCli_cmdServiceList() {
}

func ExampleServicedCli_cmdServiceAdd() {
}

func ExampleServicedCli_cmdServiceRemove() {
}

func ExampleServicedCli_cmdServiceEdit() {
}

func ExampleServicedCli_cmdServiceAutoIPs() {
}

func ExampleServicedCli_cmdServiceStart() {
}

func ExampleServicedCli_cmdServiceStop() {
}

func ExampleServicedCli_cmdServiceProxy() {
}

func ExampleServicedCli_cmdServiceShell() {
}

func ExampleServicedCli_cmdServiceRun_list() {
}

func ExampleServicedCli_cmdServiceRun_exec() {
}
