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

package datastore

import (
	"fmt"
)

// ErrNoSuchEntity is returned when no entity was found for a given key.
type ErrNoSuchEntity struct {
	Key Key
}

func (e ErrNoSuchEntity) Error() string {
	return fmt.Sprintf("No such entity {kind:%s, id:%s}", e.Key.Kind(), e.Key.ID())
}

//IsErrNoSuchEntity check see if error param is of type ErrNoSuchEntity
func IsErrNoSuchEntity(err error) bool {
	switch err.(type) {
	case ErrNoSuchEntity:
		return true
	}
	return false
}
