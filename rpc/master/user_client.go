// Copyright 2016 The Serviced Authors.
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

package master

import (
	"github.com/control-center/serviced/domain/user"
)

// Get the system user record
func (c *Client) GetSystemUser() (user.User, error) {
	systemUser := user.User{}
	err := c.call("GetSystemUser", empty, &systemUser)
	return systemUser, err
}

// Validate the credentials of the specified user
func (c *Client) ValidateCredentials(user user.User) (bool, error) {
	result := false
	err := c.call("ValidateCredentials", user, &result)
	return result, err
}
