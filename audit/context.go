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

package audit

// Context information necessary for audit logging.
type Context interface {

	// User returns the user performing the action.
	User() string
}

// NewContext returns default implementation of the Context interface that has the
// provided string for a user.
func NewContext(user string) Context {
	return &context{user: user}
}

type context struct {
	user string
}

func (c *context) User() string {
	return c.user
}
