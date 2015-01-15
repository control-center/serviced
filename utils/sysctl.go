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

package utils

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/zenoss/glog"
)

// SetSysctl tries to set sysctl settings
func SetSysctl(key string, value string) ([]byte, error) {
	command := []string{"/sbin/sysctl", "-w", fmt.Sprintf("%s=%s", key, value)}
	thecmd := exec.Command(command[0], command[1:]...)
	output, err := thecmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command:'%s' output: %s  error: %s\n", command, output, err)
		return output, err
	}
	if strings.HasPrefix(string(output), "sysctl: ") {
		glog.Errorf("Error running command:'%s' output: %s\n", command, output)
		return output, fmt.Errorf(string(output))
	}
	glog.V(1).Infof("Successfully ran command:'%s' output: %s\n", command, output)
	return output, nil
}