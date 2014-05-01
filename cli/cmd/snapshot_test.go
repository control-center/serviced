package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/zenoss/serviced/cli/api"
)

var DefaultSnapshotAPITest = SnapshotAPITest{snapshots: DefaultTestSnapshots}

var DefaultTestSnapshots = []string{
	"test-service-1-snapshot-1",
	"test-service-1-snapshot-2",
	"test-service-2-snapshot-1",
}

var (
	ErrNoSnapshotFound = errors.New("no snapshot found")
	ErrInvalidSnapshot = errors.New("invalid snapshot")
)

type SnapshotAPITest struct {
	api.API
	snapshots []string
}

func InitSnapshotAPITest(args ...string) {
	New(DefaultSnapshotAPITest).Run(args)
}

func (t SnapshotAPITest) GetSnapshots() ([]string, error) {
	return t.snapshots, nil
}

func (t SnapshotAPITest) GetSnapshotsByServiceID(serviceID string) ([]string, error) {
	var snapshots []string
	for _, s := range t.snapshots {
		if strings.HasPrefix(s, serviceID) {
			snapshots = append(snapshots, s)
		}
	}

	return snapshots, nil
}

func (t SnapshotAPITest) AddSnapshot(serviceID string) (string, error) {
	return fmt.Sprintf("%s-snapshot", serviceID), nil
}

func (t SnapshotAPITest) RemoveSnapshot(id string) error {
	for _, s := range t.snapshots {
		if s == id {
			return nil
		}
	}

	return ErrNoSnapshotFound
}

func (t SnapshotAPITest) Commit(dockerID string) (string, error) {
	return fmt.Sprintf("%s-snapshot", dockerID), nil
}

func (t SnapshotAPITest) Rollback(id string) error {
	for _, s := range t.snapshots {
		if s == id {
			return nil
		}
	}

	return ErrNoSnapshotFound
}

func ExampleServicedCli_cmdSnapshotList() {
	InitSnapshotAPITest("serviced", "snapshot", "list")

	// Output:
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
	// test-service-2-snapshot-1
}

func ExampleServicedCli_cmdSnapshotList_ByServiced() {
	InitSnapshotAPITest("serviced", "snapshot", "list", "test-service-1")

	// Output
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
}

func ExampleServicedCli_cmdSnapshotAdd() {
	InitSnapshotAPITest("serviced", "snapshot", "add", "test-service-99")

	// Output:
	// test-service-99-snapshot
}

func ExampleServicedCli_cmdSnapshotRemove() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "test-service-2-snapshot-1", "test-service-0-snapshot-4")

	// Output:
	// test-service-2-snapshot-1
}

func ExampleServiceCLI_CmdSnapshotRemove_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "rm")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    remove - Removes an existing snapshot
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot remove SNAPSHOTID ...
	//
	// OPTIONS:
}

func ExampleServicedCli_cmdSnapshotCommit() {
	InitSnapshotAPITest("serviced", "snapshot", "commit", "ABC123")

	// Output:
	// ABC123-snapshot
}

func ExampleServicedCli_cmdSnapshotRollback() {
	InitSnapshotAPITest("serviced", "snapshot", "rollback", "test-service-1-snapshot-1")

	// Output:
	// test-service-1-snapshot-1
}
