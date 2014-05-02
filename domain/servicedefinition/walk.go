// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

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
