// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
