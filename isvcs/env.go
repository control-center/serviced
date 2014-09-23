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

package isvcs

import (
	"fmt"
	"regexp"
	"strings"
)


var envPerService = make(map[string]map[string] string)

// Add a string of the form SERVICE:KEY=VAL to the isvcs environment map
func AddEnv(env_str string) error {
	result := regexp.MustCompile("([^:]+):([^=]+)=(.+)").FindStringSubmatch(env_str)
	if result==nil {
		return fmt.Errorf("Error parsing internal service environment variable: %s", env_str)
	}
	service := result[1]
	key     := result[2]
	val     := result[3]

	if serviceMap, ok := envPerService[service]; ok {
		serviceMap[key] = val
		return nil
	}
	svcNames := []string{}
	for key := range envPerService {
		svcNames = append(svcNames, key)
	}
	return fmt.Errorf("Error setting internal service environment:'%s'  Service '%s' must be one of [%s]'",
		env_str, service, strings.Join(svcNames, ", "))
}

