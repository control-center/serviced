// Copyright 2017 The Serviced Authors.
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

package logfilter

import (
	"github.com/control-center/serviced/datastore"
)

// LogFilter is the definition of a log filter
type LogFilter struct {
	Name                   string                 // Name of the filter
	Version                string                 // Version of the parent service
	Filter                 string                 // the filter string
	datastore.VersionedEntity
}

func (lf LogFilter) String() string {
	return lf.GetID()
}

// Equals checks the equality of two log filters
func (a *LogFilter) Equals(b *LogFilter) bool {
	if a.Name != b.Name {
		return false
	}
	if a.Version != b.Version {
		return false
	}
	if a.Filter != b.Filter {
		return false
	}
	return true
}

// GetType return the LogFilter's type
// It returns the type as a string
func GetType() string {
	return kind
}

// GetType returns the LogFilter instance's type
// It returns the type as a string
func (lf *LogFilter) GetType() string {
	return GetType()
}

// GetID return a LogFilter instance's ID
// It returns the ID as a string
func (lf *LogFilter) GetID() string {
	return buildID(lf.Name, lf.Version)
}
