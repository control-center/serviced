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
package zookeeper

import (
	"fmt"

	zklib "github.com/control-center/go-zookeeper/zk"
	"github.com/zenoss/glog"
)

type zkLogger struct {}

// Assert that our logging shim implements the Logger interface
var _ zklib.Logger =  zkLogger{}

// Register a logger for the Zookeeper library which will messages from their library to our application log
func RegisterZKLogger() {
	zklib.DefaultLogger = zkLogger{}
}

func (logger zkLogger) Printf(format string, args ...interface{}) {
	glog.V(1).Infof("zklib: %s", fmt.Sprintf(format, args))
}
