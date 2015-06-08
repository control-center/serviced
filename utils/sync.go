// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package utils

import (
	"errors"

	"github.com/zenoss/glog"
)

var (
	// Handles unexpected use cases
	ErrBadSync = errors.New("bad data cannot sync")
)

// Synchronizer is an interface type for synchronizing data
type Synchronizer interface {
	// Returns the list of data ids to the receiving db
	IDs() ([]string, error)
	// Create creates a new object in the receieving db
	Create(data interface{}) error
	// Update updates and existing object in the receieving db
	Update(data interface{}) error
	// Delete deletes an object in the receiving db
	Delete(id string) error
}

// Sync does the actual data sync given a map of id :=> data
func Sync(s Synchronizer, data map[string]interface{}) error {
	ids, err := s.IDs()
	if err != nil {
		glog.Errorf("Could not look up ids: %s", err)
		return err
	}

	for _, id := range ids {
		if obj, ok := data[id]; !ok {
			// this node has no data
			if err := s.Delete(id); err != nil {
				glog.Errorf("Could not delete %s: %s", id, err)
				return err
			}
		} else {
			// if object is nil, then there are no updates to this node
			if obj != nil {
				// this node has updates
				if err := s.Update(obj); err != nil {
					glog.Errorf("Could not update %s: %s", id, err)
					return err
				}
			}
			delete(data, id)
		}
	}

	for id, obj := range data {
		if obj != nil {
			if err := s.Create(obj); err != nil {
				glog.Errorf("Could not create %s: %s", id, err)
				return err
			}
		} else {
			// User is trying to sync data that isn't available?  In any case,
			// this should not happen.
			glog.Errorf("Could not sync %s: %s", id, ErrBadSync)
			return ErrBadSync
		}
	}

	return nil
}