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

package cmd

import (
	"errors"
	"fmt"
	"strings"

	"github.com/control-center/serviced/cli/api"
)

const (
	NilSnapshot = "NilSnapshot"
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
	fail      bool
	snapshots []string
}

func InitSnapshotAPITest(args ...string) {
	New(DefaultSnapshotAPITest).Run(args)
}

func (t SnapshotAPITest) hasSnapshot(id string) (bool, error) {
	if t.fail {
		return false, ErrInvalidSnapshot
	}
	for _, s := range t.snapshots {
		if s == id {
			return true, nil
		}
	}
	return false, nil
}

func (t SnapshotAPITest) GetSnapshots() ([]string, error) {
	if t.fail {
		return nil, ErrInvalidSnapshot
	}
	return t.snapshots, nil
}

func (t SnapshotAPITest) GetSnapshotsByServiceID(serviceID string) ([]string, error) {
	if t.fail {
		return nil, ErrInvalidSnapshot
	}
	var snapshots []string
	for _, s := range t.snapshots {
		if strings.HasPrefix(s, serviceID) {
			snapshots = append(snapshots, s)
		}
	}
	return snapshots, nil
}

func (t SnapshotAPITest) AddSnapshot(id string) (string, error) {
	if t.fail {
		return "", ErrInvalidSnapshot
	} else if id == NilSnapshot {
		return "", nil
	}
	return fmt.Sprintf("%s-snapshot", id), nil
}

func (t SnapshotAPITest) RemoveSnapshot(id string) error {
	if ok, err := t.hasSnapshot(id); err != nil {
		return err
	} else if !ok {
		return ErrNoSnapshotFound
	}
	return nil
}

func (t SnapshotAPITest) Commit(dockerID string) (string, error) {
	return t.AddSnapshot(dockerID)
}

func (t SnapshotAPITest) Rollback(id string) error {
	return t.RemoveSnapshot(id)
}

func ExampleServicedCLI_CmdSnapshotList() {
	InitSnapshotAPITest("serviced", "snapshot", "list")

	// Output:
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
	// test-service-2-snapshot-1
}

func ExampleServicedCLI_CmdSnapshotList_byServiceID() {
	InitSnapshotAPITest("serviced", "snapshot", "list", "test-service-1")

	// Output:
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
}

func ExampleServicedCLI_CmdSnapshotList_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	// failed to retrieve all snapshots
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list")
	// failed to retrieve all snapshots by service id
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1")

	// Output:
	// invalid snapshot
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotList_err() {
	DefaultSnapshotAPITest.snapshots = nil
	defer func() { DefaultSnapshotAPITest.snapshots = DefaultTestSnapshots }()
	// no snapshots found
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list")
	// no snapshots found for service id
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1")

	// Output:
	// no snapshots found
	// no snapshots found
}

func ExampleServicedCLI_CmdSnapshotAdd() {
	InitSnapshotAPITest("serviced", "snapshot", "add", "test-service-99")

	// Output:
	// test-service-99-snapshot
}

func ExampleServicedCLI_CmdSnapshotAdd_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "add")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    add - Take a snapshot of an existing service
	//
	// USAGE:
	//    command add [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot add SERVICEID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdSnapshotAdd_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "add", "test-service-2")

	// Output:
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotAdd_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "add", NilSnapshot)

	// Output:
	// received nil snapshot
}

func ExampleServicedCLI_CmdSnapshotRemove() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "test-service-2-snapshot-1")

	// Output:
	// test-service-2-snapshot-1
}

func ExampleServicedCLI_CmdSnapshotRemove_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "remove")

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

func ExampleServicedCLI_CmdSnapshotRemove_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "remove", "test-service-0-snapshot")

	// Output:
	// test-service-0-snapshot: no snapshot found
}

func ExampleServicedCLI_CmdSnapshotCommit() {
	InitSnapshotAPITest("serviced", "snapshot", "commit", "ABC123")

	// Output:
	// ABC123-snapshot
}

func ExampleServicedCLI_CmdSnapshotCommit_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "commit")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    commit - Snapshots and commits a given service instance
	//
	// USAGE:
	//    command commit [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot commit DOCKERID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdSnapshotCommit_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "commit", "ABC123")

	// Output:
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotCommit_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "commit", NilSnapshot)

	// Output:
	// received nil snapshot
}

func ExampleServicedCLI_CmdSnapshotRollback() {
	InitSnapshotAPITest("serviced", "snapshot", "rollback", "test-service-1-snapshot-1")

	// Output:
	// test-service-1-snapshot-1
}

func ExampleServicedCLI_CmdSnapshotRollback_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "rollback")

	// Output:
	// Incorrect Usage.
	//
	// NAME:
	//    rollback - Restores the environment to the state of the given snapshot
	//
	// USAGE:
	//    command rollback [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot rollback SNAPSHOTID
	//
	// OPTIONS:
}

func ExampleServicedCLI_CmdSnapshotRollback_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "rollback", "test-service-0-snapshot")

	// Output:
	// no snapshot found
}
