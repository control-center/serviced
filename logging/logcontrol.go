// Copyright 2016 The Serviced Authors.
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
package logging

import (
	"github.com/Sirupsen/logrus"
	"github.com/zenoss/glog"
	"github.com/zenoss/logri"
)

type logControl struct{}

// assert interface
var _ LogControl = &logControl{}

func NewLogControl() LogControl {
	return logControl{}
}

func (l logControl) SetLevel(level logrus.Level) {
	logri.SetLevel(level)
	logrus.SetLevel(level)
}

func (l logControl) ApplyConfigFromFile(file string) error {
	return logri.ApplyConfigFromFile(file)
}

func (l logControl) WatchConfigFile(file string) {
	logri.WatchConfigFile(file)
}

func (l logControl) SetVerbosity(value int) {
	glog.SetVerbosity(value)
}

func (l logControl) GetVerbosity() int {
	return int(glog.GetVerbosity())
}

func (l logControl) SetToStderr(value bool) {
	glog.SetToStderr(value)
}

func (l logControl) SetAlsoToStderr(value bool) {
	glog.SetAlsoToStderr(value)
}

func (l logControl) SetStderrThreshold(value string) error {
	return glog.SetStderrThreshold(value)
}

func (l logControl) SetVModule(value string) error {
	return glog.SetVModule(value)
}

func (l logControl) SetTraceLocation(value string) error {
	return glog.SetTraceLocation(value)
}
