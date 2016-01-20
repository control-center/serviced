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

package api

import (
	"fmt"
	"strings"

	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
)

const (
	minTimeout     = 30
	defaultTimeout = 600
)

var (
	empty     interface{}
	unusedInt int
)

// GetAgentIP returns the agent ip address
func GetAgentIP(defaultRPCPort int) string {
	if options.Endpoint != "" {
		return options.Endpoint
	}
	agentIP, err := utils.GetIPAddress()
	if err != nil {
		glog.Fatalf("Failed to get IP address: %s", err)
	}
	return agentIP + fmt.Sprintf(":%d", defaultRPCPort)
}

type version []int

func (a version) String() string {
	var format = make([]string, len(a))
	for idx, value := range a {
		format[idx] = fmt.Sprintf("%d", value)
	}
	return strings.Join(format, ".")
}

func (a version) Compare(b version) int {
	astr := ""
	for _, s := range a {
		astr += fmt.Sprintf("%12d", s)
	}
	bstr := ""
	for _, s := range b {
		bstr += fmt.Sprintf("%12d", s)
	}

	if astr > bstr {
		return -1
	} else if astr < bstr {
		return 1
	} else {
		return 0
	}
}

func convertStringSliceToMap(list []string) map[string]string {
	mapValues := make(map[string]string)
	for i, keyValuePair := range list {
		if keyValuePair == "" {
			glog.Warningf("Skipping empty key=value pair at index %d from list %v", i, list)
			continue
		}
		keyValue := strings.SplitN(keyValuePair, "=", 2)
		if len(keyValue) != 2 || keyValue[0] == "" {
			glog.Warningf("Skipping invalid key=value pair %q at index %d from list %v", keyValuePair, i, list)
			continue
		}
		mapValues[keyValue[0]] = keyValue[1]
	}
	return mapValues
}
