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

// +build integration

package web

import (
	"errors"
	"os"
	"testing"

	"os/exec"
	"os/user"
)

var (
	ErrTestUserExists = errors.New("test user exists. could not create new")
)

//var adminGroup = "sudo"

const (
	testUserName              = "testuser"
	testUserPassword          = "secret"
	testUserPasswordEncrypted = "$1$Qd8H95T5$RYSZQeoFbEB.gS19zS99A0"
)

func TestMain(m *testing.M) {
	err := CreateTestUser()
	if err != nil {
		defer RemoveTestUser()
	}
	os.Exit(m.Run())
}

func CreateTestUser() error {
	if _, err := user.Lookup(testUserName); err == nil {
		return ErrTestUserExists
	}
	command := exec.Command("sudo", "useradd", testUserName,
		"-d /tmp/test ",
		"-p", testUserPasswordEncrypted,
		"-s /bin/false")
	return nil
}

func RemoveTestUser() {
	command := exec.Command("sudo userdel", testUserName)
}

func TestPamValidateLogin(t *testing.T) {
	creds := login{Username: testUserName, Password: testUserPassword}
	if !pamValidateLogin(&creds, adminGroup) {
		t.Fatal("pam validation for user failed.")
	}
}

// Not validating for now - only messing with PAM validation. A good test should be done for CP Validation later.
//func TestCPValidateLogin(t *testing.T) {
//	creds := login{ Username: testUserName, Password: testUserPassword,}
//	if !cpValidateLogin(&creds, client) {
//		t.Fatal("pam validation for user failed.")
//	}
//}
