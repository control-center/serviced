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
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/utils"
	"github.com/control-center/serviced/volume/btrfs"
)

const (
	NilSnapshot = "NilSnapshot"
)

var DefaultSnapshotAPITest = SnapshotAPITest{snapshots: DefaultTestSnapshots}

var DefaultTestSnapshots = []dao.SnapshotInfo{
	dao.SnapshotInfo{SnapshotID: "test-service-1-snapshot-1", TenantID: "test-service-1", Description: "description 1", Tags: []string{"tag-1"}},
	dao.SnapshotInfo{SnapshotID: "test-service-1-snapshot-2", TenantID: "test-service-1", Description: "description 2", Tags: []string{"tag-2", "tag-3"}},
	dao.SnapshotInfo{SnapshotID: "test-service-1-invalid", Invalid: true},
	dao.SnapshotInfo{SnapshotID: "test-service-2-snapshot-1", TenantID: "test-service-2", Description: "", Tags: []string{""}},
	dao.SnapshotInfo{SnapshotID: "test-service-2-invalid", Invalid: true},
}

var (
	ErrNoSnapshotFound = errors.New("no snapshot found")
	ErrInvalidSnapshot = errors.New("invalid snapshot")
	ErrGetByTagFailed  = errors.New("unable to retrieve snapshot by tag name")
)

type SnapshotAPITest struct {
	api.API
	fail         bool
	btrfsFail    bool
	getByTagFail bool
	snapshots    []dao.SnapshotInfo
}

func InitSnapshotAPITest(args ...string) {
	New(DefaultSnapshotAPITest, utils.TestConfigReader(make(map[string]string))).Run(args)
}

func (t SnapshotAPITest) hasSnapshot(id string) (bool, error) {
	if t.fail {
		return false, ErrInvalidSnapshot
	}
	for _, s := range t.snapshots {
		if s.SnapshotID == id {
			return true, nil
		}
	}
	return false, nil
}

func (t SnapshotAPITest) GetSnapshots() ([]dao.SnapshotInfo, error) {
	if t.fail {
		return nil, ErrInvalidSnapshot
	}
	return t.snapshots, nil
}

func (t SnapshotAPITest) GetSnapshotsByServiceID(serviceID string) ([]dao.SnapshotInfo, error) {
	if t.fail {
		return nil, ErrInvalidSnapshot
	}
	var snapshots []dao.SnapshotInfo
	for _, s := range t.snapshots {
		if s.TenantID == serviceID || strings.HasPrefix(s.SnapshotID, serviceID) {
			snapshots = append(snapshots, s)
		}
	}
	return snapshots, nil
}

func (t SnapshotAPITest) AddSnapshot(config api.SnapshotConfig) (string, error) {
	if t.fail {
		return "", ErrInvalidSnapshot
	} else if config.ServiceID == NilSnapshot || config.DockerID == NilSnapshot {
		return "", nil
	}

	var id string
	if id = config.ServiceID; id == "" {
		id = config.DockerID
		return fmt.Sprintf("%s-snapshot", id), nil
	}

	return fmt.Sprintf("%s-snapshot description=%q tags=%q", id, config.Message, config.Tag), nil
}

func (t SnapshotAPITest) RemoveSnapshot(id string) error {
	if ok, err := t.hasSnapshot(id); err != nil {
		return err
	} else if !ok {
		return ErrNoSnapshotFound
	}
	return nil
}

func (t SnapshotAPITest) GetSnapshotByServiceIDAndTag(serviceID string, tagName string) (string, error) {
	if t.getByTagFail {
		return "", ErrGetByTagFailed
	}
	for _, s := range t.snapshots {
		if s.TenantID == serviceID {
			for _, t := range s.Tags {
				if t == tagName {
					return s.SnapshotID, nil
				}
			}
		}
	}
	return "", nil
}

func (t SnapshotAPITest) Commit(dockerID string) (string, error) {
	if t.fail {
		return "", ErrInvalidSnapshot
	} else if dockerID == NilSnapshot {
		return "", nil
	}
	return fmt.Sprintf("%s-snapshot", dockerID), nil
}

func (t SnapshotAPITest) Rollback(id string, f bool) error {
	return t.RemoveSnapshot(id)
}

func (t SnapshotAPITest) TagSnapshot(snapshotID string, tagName string) error {
	if t.fail {
		return ErrInvalidSnapshot
	} else if t.btrfsFail {
		return btrfs.ErrBtrfsNotSupported
	}
	for _, s := range t.snapshots {
		if s.SnapshotID == snapshotID {
			return nil
		}
	}

	return ErrNoSnapshotFound
}

func (t SnapshotAPITest) RemoveSnapshotTag(serviceID string, tagName string) (string, error) {
	if t.fail {
		return "", ErrInvalidSnapshot
	} else if t.btrfsFail {
		return "", btrfs.ErrBtrfsNotSupported
	}

	for _, s := range t.snapshots {
		if s.TenantID == serviceID {
			for _, tag := range s.Tags {
				if tag == tagName {
					return s.SnapshotID, nil
				}
			}
		}
	}

	return "", ErrNoSnapshotFound
}

func ExampleServicedCLI_CmdSnapshotList() {
	InitSnapshotAPITest("serviced", "snapshot", "list")

	// Output:
	// test-service-1-snapshot-1 description 1
	// test-service-1-snapshot-2 description 2
	// test-service-1-invalid [DEPRECATED]
	// test-service-2-snapshot-1
	// test-service-2-invalid [DEPRECATED]

}

func TestServicedCLI_CmdSnapshotList_ShowTagsShort(t *testing.T) {
	output := pipe(InitSnapshotAPITest, "serviced", "snapshot", "list", "-t")
	expected :=
		"Snapshot                                 Description        Tags" +
			"\ntest-service-1-snapshot-1                description 1      tag-1" +
			"\ntest-service-1-snapshot-2                description 2      tag-2,tag-3" +
			"\ntest-service-1-invalid [DEPRECATED]" +
			"\ntest-service-2-snapshot-1" +
			"\ntest-service-2-invalid [DEPRECATED]"

	outStr := TrimLines(fmt.Sprintf("%s", output))
	expected = TrimLines(expected)

	if expected != outStr {
		t.Fatalf("\ngot:\n%s\nwant:\n%s", outStr, expected)
	}
}

func TestServicedCLI_CmdSnapshotList_ShowTagsLong(t *testing.T) {
	output := pipe(InitSnapshotAPITest, "serviced", "snapshot", "list", "--show-tags")
	expected :=
		"Snapshot                                 Description        Tags" +
			"\ntest-service-1-snapshot-1                description 1      tag-1" +
			"\ntest-service-1-snapshot-2                description 2      tag-2,tag-3" +
			"\ntest-service-1-invalid [DEPRECATED]" +
			"\ntest-service-2-snapshot-1" +
			"\ntest-service-2-invalid [DEPRECATED]"

	outStr := TrimLines(fmt.Sprintf("%s", output))
	expected = TrimLines(expected)

	if expected != outStr {
		t.Fatalf("\ngot:\n%s\nwant:\n%s", outStr, expected)
	}
}

func ExampleServicedCLI_CmdSnapshotList_byServiceID() {
	InitSnapshotAPITest("serviced", "snapshot", "list", "test-service-1")

	// Output:
	// test-service-1-snapshot-1 description 1
	// test-service-1-snapshot-2 description 2
	// test-service-1-invalid [DEPRECATED]
}

func TestServicedCLI_CmdSnapshotList_byServiceID_ShowTagsShort(t *testing.T) {
	output := pipe(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1", "-t")
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

func TestervicedCLI_CmdSnapshotList_byServiceID_ShowTagsLong(t *testing.T) {
	output := pipe(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1", "--show-tags")
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

func ExampleServicedCLI_CmdSnapshotList_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	// failed to retrieve all snapshots
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list")
	// failed to retrieve all snapshots by service id
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1")
	// failed to retrieve all snapshots with tags
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "-t")
	// failed to retrieve all snapshots with tags by service id
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1", "-t")

	// Output:
	// invalid snapshot
	// invalid snapshot
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
	// no snapshots found with tags
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "-t")
	// no snapshots found with tags for service id
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "list", "test-service-1", "-t")

	// Output:
	// no snapshots found
	// no snapshots found
	// no snapshots found
	// no snapshots found
}

func ExampleServicedCLI_CmdSnapshotAdd() {
	InitSnapshotAPITest("serviced", "snapshot", "add", "test-service-99")

	// Output:
	// test-service-99-snapshot description="" tags=""
}

func ExampleServicedCLI_CmdSnapshotAdd_withDescription() {
	InitSnapshotAPITest("serviced", "snapshot", "add", "test-service-99", "-d", "some description")

	// Output:
	// test-service-99-snapshot description="some description" tags=""
}

func ExampleServicedCLI_CmdSnapshotAdd_withDescriptionAndTags() {
	InitSnapshotAPITest("serviced", "snapshot", "add", "test-service-99", "-d", "some description", "-t", "unusedTag")

	// Output:
	// test-service-99-snapshot description="some description" tags="unusedTag"
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
	//    --description, -d 	a description of the snapshot
	//    --tag, -t 		a unique tag for the snapshot

}

/*
this command exits 1 which fails the test runner
func ExampleServicedCLI_CmdSnapshotAdd_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "add", "test-service-2")

	// Output:
	// invalid snapshot
}
*/

/*
this command exits 1 which fails the test runner
func ExampleServicedCLI_CmdSnapshotAdd_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "add", NilSnapshot)

	// Output:
	// received nil snapshot
}
*/

func ExampleServicedCLI_CmdSnapshotRemove() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "test-service-2-snapshot-1")

	// Output:
	// test-service-2-snapshot-1
}

func ExampleServicedCLI_CmdSnapshotRemove_All() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "-f")

	// Output:
	// test-service-1-snapshot-1
	// test-service-1-snapshot-2
	// test-service-1-invalid
	// test-service-2-snapshot-1
	// test-service-2-invalid

}

func ExampleServicedCLI_CmdSnapshotRemove_Tag() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "test-service-1", "tag-1")

	// Output:
	// test-service-1-snapshot-1
}

func ExampleServicedCLI_CmdSnapshotRemove_All_NoForce() {
	InitSnapshotAPITest("serviced", "snapshot", "remove")

	// Output:
	// Incorrect Usage.
	// Use
	//    serviced snapshot remove -f
	// to delete all snapshots, or
	//    serviced snapshot remove -h
	// for help with this command.
}

func ExampleServicedCLI_CmdSnapshotRemove_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "--help")

	// Output:
	// NAME:
	//    remove - Removes an existing snapshot
	//
	// USAGE:
	//    command remove [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot remove [SNAPSHOTID | SERVICED TAG-NAME]
	//
	// OPTIONS:
	//    --force, -f	required for deleting all snapshots

}

func ExampleServicedCLI_CmdSnapshotRemove_BadID() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "remove", "test-service-0-snapshot")

	// Output:
	// no snapshot found
}

func ExampleServicedCLI_CmdSnapshotRemove_nomatch_Tag() {
	InitSnapshotAPITest("serviced", "snapshot", "remove", "test-service-2", "Tag-1")

	// Output:
	// No matching snapshots found.
}

func ExampleServicedCLI_CmdSnapshotRemove_error() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "remove", "test-service-1-snapshot-1")

	// Output:
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotRemove_errorGetByTag() {
	DefaultSnapshotAPITest.getByTagFail = true
	defer func() { DefaultSnapshotAPITest.getByTagFail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "remove", "test-service-1", "tag-1")

	// Output:
	// unable to retrieve snapshot by tag name
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
	//    --force-restart	restarts running services during rollback
}

/*
this command exits 1 which fails the test runner
func ExampleServicedCLI_CmdSnapshotRollback_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "rollback", "test-service-0-snapshot")

	// Output:
	// no snapshot found
}
*/

func ExampleServicedCLI_CmdSnapshotTag_usage() {
	InitSnapshotAPITest("serviced", "snapshot", "tag")

	// Output:
	// 	Incorrect Usage.
	//
	// NAME:
	//    tag - Tags an existing snapshot with TAG-NAME
	//
	// USAGE:
	//    command tag [command options] [arguments...]
	//
	// DESCRIPTION:
	//    serviced snapshot tag SNAPSHOTID TAG-NAME
	//
	// OPTIONS:

}

func ExampleServicedCLI_CmdSnapshotTag() {
	InitSnapshotAPITest("serviced", "snapshot", "tag", "test-service-1-snapshot-1", "tag-A")

	// Output:
	//
}

func ExampleServicedCLI_CmdSnapshotTag_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "tag", "test-service-0-snapshot", "tag-A")

	// Output:
	// no snapshot found
}

func ExampleServicedCLI_CmdSnapshotTag_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "tag", "test-service-1-snapshot-1", "tag-A")

	// Output:
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotTag_btrfsFail() {
	DefaultSnapshotAPITest.btrfsFail = true
	defer func() { DefaultSnapshotAPITest.btrfsFail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "tag", "test-service-1-snapshot-1", "tag-A")

	// Output:
	// operation not supported on btrfs driver
}

func ExampleServicedCLI_CmdSnapshotUntag() {
	InitSnapshotAPITest("serviced", "snapshot", "untag", "test-service-1", "tag-2")

	// Output:
	// test-service-1-snapshot-2
}

func ExampleServicedCLI_CmdSnapshotUntag_err() {
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "untag", "test-service-0", "tag-A")

	// Output:
	// no snapshot found
}

func ExampleServicedCLI_CmdSnapshotUntag_fail() {
	DefaultSnapshotAPITest.fail = true
	defer func() { DefaultSnapshotAPITest.fail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "untag", "test-service-1", "tag-2")

	// Output:
	// invalid snapshot
}

func ExampleServicedCLI_CmdSnapshotUntag_btrfsFail() {
	DefaultSnapshotAPITest.btrfsFail = true
	defer func() { DefaultSnapshotAPITest.btrfsFail = false }()
	pipeStderr(InitSnapshotAPITest, "serviced", "snapshot", "untag", "test-service-1", "tag-A")

	// Output:
	// operation not supported on btrfs driver
}
