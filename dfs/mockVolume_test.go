// Copyright 2015 The Serviced Authors.
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

package dfs

import (
	"github.com/control-center/serviced/volume"

	"github.com/stretchr/testify/mock"
)

// assert the interface
var _ volume.Volume = &mockVolume{}

type mockVolume struct {
	mock.Mock
}

func (mv *mockVolume) Name() string {
	return mv.Mock.Called().String(0)
}

func (mv *mockVolume) Path() string {
	return mv.Mock.Called().String(0)
}

func (mv *mockVolume) SnapshotPath(label string) string {
	return mv.Mock.Called(label).String(0)
}

func (mv *mockVolume) Snapshot(label string) error {
	return mv.Mock.Called(label).Error(0)
}

func (mv *mockVolume) Snapshots() ([]string, error) {
	args := mv.Mock.Called()

	var snapshots []string
	if arg0 := args.Get(0); arg0 != nil {
		snapshots = arg0.([]string)
	}
	return snapshots, args.Error(1)
}

func (mv *mockVolume) RemoveSnapshot(label string) error {
	return mv.Mock.Called(label).Error(0)
}

func (mv *mockVolume) Rollback(label string) error {
	return mv.Mock.Called(label).Error(0)
}

func (mv *mockVolume) Unmount() error {
	return mv.Mock.Called().Error(0)
}

func (mv *mockVolume) Export(label, parent, filename string) error {
	return mv.Mock.Called(label, parent, filename).Error(0)
}

func (mv *mockVolume) Import(label, filename string) error {
	return mv.Mock.Called(label, filename).Error(0)
}
