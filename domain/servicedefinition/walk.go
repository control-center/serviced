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

package servicedefinition

//Visit function called when walking a ServiceDefinition hierarchy
type Visit func(sd *ServiceDefinition) error

//Walk a ServiceDefinition hierarchy and call the Visit function at each ServiceDefinition. Returns error if any Visit
// returns an error
func Walk(sd *ServiceDefinition, visit Visit) error {
	if err := visit(sd); err != nil {
		return err
	}
	for _, def := range sd.Services {
		if err := Walk(&def, visit); err != nil {
			return err
		}
	}
	return nil
}
