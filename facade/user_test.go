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

// +build integration

package facade

import (
	userdomain "github.com/control-center/serviced/domain/user"
	. "gopkg.in/check.v1"
)

func (ft *IntegrationTest) TestUser_UserOperations(t *C) {
	user := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	err := ft.Facade.AddUser(ft.CTX, user)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}

	var newUser userdomain.User
	newUser, err = ft.Facade.GetUser(ft.CTX, "Pepe")
	if err != nil {
		t.Fatalf("Failure getting user %s", err)
	}

	// make sure they are the same user
	if newUser.Name != user.Name {
		t.Fatalf("Retrieved an unexpected user %v", newUser)
	}

	// make sure the password was hashed
	if newUser.Password == "Pepe" {
		t.Fatalf("Did not hash the password %+v", user)
	}

	err = ft.Facade.RemoveUser(ft.CTX, "Pepe")
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}
}

func (ft *IntegrationTest) TestUser_ValidateCredentials(t *C) {
	user := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	err := ft.Facade.AddUser(ft.CTX, user)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	var isValid bool
	attemptUser := userdomain.User{
		Name:     "Pepe",
		Password: "Pepe",
	}
	isValid, err = ft.Facade.ValidateCredentials(ft.CTX, attemptUser)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}

	if !isValid {
		t.Fatalf("Unable to authenticate user credentials")
	}

	err = ft.Facade.RemoveUser(ft.CTX, "Pepe")
	if err != nil {
		t.Fatalf("Failure removing user %s", err)
	}

	// update the user
	user.Password = "pepe2"
	err = ft.Facade.UpdateUser(ft.CTX, user)
	if err != nil {
		t.Fatalf("Failure creating a user %s", err)
	}
	attemptUser.Password = "Pepe2"
	// make sure we can validate against the updated credentials
	isValid, err = ft.Facade.ValidateCredentials(ft.CTX, attemptUser)

	if err != nil {
		t.Fatalf("Failure authenticating credentials %s", err)
	}
}
