package cmd

import (
	"errors"

	"github.com/zenoss/serviced/serviced/api"
)

var DefaultSnapshotAPITest = SnapshotAPITest{snapshots: DefaultTestSnapshots}

var DefaultTestSnapshots = []string{
	"test-service-1-snapshot-1",
	"test-service-1-snapshot-2",
	"test-service-2-snapshot-1",
}

var (
	ErrNoServiceFound  = errors.New("no service found")
	ErrNoSnapshotFound = errors.New("no snapshot found")
)

type SnapshotAPITest struct {
	api.API
	snapshots []string
}

func InitSnapshotAPITest(args ...string) {
	New(DefaultSnapshotAPITest).Run(args)
}

func (t SnapshotAPITest) GetSnapshots() ([]string, error) {
	return nil, nil
}

func (t SnapshotAPITest) GetSnapshotsByServiceID(serviceID string) ([]string, error) {
	return nil, nil
}

func (t SnapshotAPITest) AddSnapshot(serviceID string) (string, error) {
	return "", nil
}

func (t SnapshotAPITest) RemoveSnapshot(id string) error {
	return nil
}

func (t SnapshotAPITest) Commit(dockerID string) (string, error) {
	return "", nil
}

func (t SnapshotAPITest) Rollback(id string) error {
	return nil
}

func ExampleServicedCli_cmdSnapshotList() {
	InitSnapshotAPITest("serviced", "snapshot", "list")
}

func ExampleServicedCli_cmdSnapshotAdd() {
	InitSnapshotAPITest("serviced", "snapshot", "add")
}

func ExampleServicedCli_cmdSnapshotRemove() {
	InitSnapshotAPITest("serviced", "snapshot", "remove")
	InitSnapshotAPITest("serviced", "snapshot", "rm")
}

func ExampleServicedCli_cmdSnapshotCommit() {
	InitSnapshotAPITest("serviced", "snapshot", "commit")
}

func ExampleServicedCli_cmdSnapshotRollback() {
	InitSnapshotAPITest("serviced", "snapshot", "rollback")
}
