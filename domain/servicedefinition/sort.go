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

// ServiceDefinitionByName implements sort.Interface for []ServiceDefinition based on Name field
type ServiceDefinitionByName []ServiceDefinition

func (a ServiceDefinitionByName) Len() int           { return len(a) }
func (a ServiceDefinitionByName) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ServiceDefinitionByName) Less(i, j int) bool { return a[i].Name < a[j].Name }

// AddressResourceConfigByPort implements sort.Interface for []AddressResourceConfig based on the Port field
type AddressResourceConfigByPort []AddressResourceConfig

func (a AddressResourceConfigByPort) Len() int           { return len(a) }
func (a AddressResourceConfigByPort) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a AddressResourceConfigByPort) Less(i, j int) bool { return a[i].Port < a[j].Port }
