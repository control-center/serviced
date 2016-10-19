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
)

type LogControl interface {
	SetLevel(level logrus.Level)
	ApplyConfigFromFile(file string) error
	WatchConfigFile(file string)

	// Legacy glog interface
	SetVerbosity(value int)
	GetVerbosity() int
	SetToStderr(value bool)
	SetAlsoToStderr(value bool)
	SetStderrThreshold(value string) error
	SetVModule(value string) error
	SetTraceLocation(value string) error
}
