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

package volume_test

import (
	"testing"

	. "github.com/control-center/serviced/volume"
)

type TestDriver struct {
	root string
}

func (d TestDriver) Create(volumeName string) (Volume, error) {
	return TestVolume{volumeName, d.root}, nil
}

func (d TestDriver) Remove(volumeName string) error {
	return nil
}

func (d TestDriver) List() []string {
	return nil
}

func (d TestDriver) Root() string {
	return d.root
}

func (d TestDriver) Get(volumeName string) (Volume, error) {
	return d.Create(volumeName)
}

func (d TestDriver) Release(volumeName string) error {
	return nil
}

func (d TestDriver) Exists(volumeName string) bool {
	return true
}

func (d TestDriver) Cleanup() error {
	return nil
}

type TestVolume struct {
	name string
	path string
}

func (c TestVolume) Name() string {
	return c.name
}

func (c TestVolume) Path() string {
	return c.path
}

func (v TestVolume) Tenant() string {
	return ""
}

func (c TestVolume) Snapshot(label string) error {
	return nil
}

func (c TestVolume) Snapshots() ([]string, error) {
	return []string{}, nil
}

func (c TestVolume) RemoveSnapshot(label string) error {
	return nil
}

func (c TestVolume) Rollback(label string) error {
	return nil
}

func (c TestVolume) Export(label, parent, filename string) error {
	return nil
}

func (c TestVolume) Import(label, filename string) error {
	return nil
}

func TestNilRegistration(t *testing.T) {
	if err := Register("nilregistration", nil); err == nil {
		t.Fatal("nil driver registration didn't produce an error")
	}
}

func newTestDriver(root string) (Driver, error) {
	return &TestDriver{root}, nil
}

func TestRedundantRegistration(t *testing.T) {
	Register("redundant", newTestDriver)
	err := Register("redundant", newTestDriver)
	if err == nil {
		t.Fatal("Redundant driver registration did not produce an error")
	}
}

func TestRegistration(t *testing.T) {
	Register("registration", newTestDriver)
	if _, err := GetDriver("registration", ""); err != nil {
		t.Fatal("Test driver was not registered")
	}
}

func TestUnregistered(t *testing.T) {
	if _, err := GetDriver("unregistered", ""); err != ErrDriverNotSupported {
		t.Fatal("Retrieval of unsupported driver did not produce an error")
	}
}

func TestMount(t *testing.T) {
	Register("testmount", newTestDriver)
	v, err := Mount("testmount", "testmount", "/opt/testmount")
	switch {
	case err != nil:
		t.Fatalf("Mount failed: %v", err)
	case v == nil:
		t.Fatal("nil volume")
	}
}

func TestBadMount(t *testing.T) {
	if _, err := Mount("badmount", "badmount", "/opt/badmount"); err != ErrDriverNotSupported {
		t.Fatal("bad mount should not succeed")
	}
}
