// Copyright 2015 The Serviced Authors.
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

package domain

import "fmt"

// EmptyNotAllowed is an error type for validating args in a datastore search
type EmptyNotAllowed string

// Error returns the error string
func (err EmptyNotAllowed) Error() string {
	return fmt.Sprintf("empty %s not allowed", err)
}