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
	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/logging"
)

type MockLogControl struct{}

var _ logging.LogControl = &MockLogControl{}

func (l MockLogControl) SetLevel(level logrus.Level) {
}
func (l MockLogControl) ApplyConfigFromFile(file string) error {
	return nil
}
func (l MockLogControl) WatchConfigFile(file string) {
}
func (l MockLogControl) SetVerbosity(value int) {
}
func (l MockLogControl) GetVerbosity() int {
	return 0
}
func (l MockLogControl) SetToStderr(value bool) {
}
func (l MockLogControl) SetAlsoToStderr(value bool) {
}
func (l MockLogControl) SetStderrThreshold(value string) error {
	return nil
}
func (l MockLogControl) SetVModule(value string) error {
	return nil
}
func (l MockLogControl) SetTraceLocation(value string) error {
	return nil
}
