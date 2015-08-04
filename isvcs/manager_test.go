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

// +build integration

// Package agent implements a service that runs on a serviced node. It is
// responsible for ensuring that a particular node is running the correct services
// and reporting the state and health of those services back to the master
// serviced.

package isvcs

import (
	"github.com/control-center/serviced/utils"

	"testing"
	"time"
)

func TestManager(t *testing.T) {
	testManager := NewManager(utils.LocalDir("images"), "/tmp")

	if err := testManager.Start(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fail()
	}

	cd1 := IServiceDefinition{
		Name:    "test1",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: func() string { return `while true; do echo hello world; sleep 1; done` },
	}
	container, err := NewIService(cd1)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fail()
	}

	cd2 := IServiceDefinition{
		Name:    "test2",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: func() string { return `while true; do echo hello world; sleep 1; done` },
	}
	container2, err := NewIService(cd2)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fail()
	}

	if err := testManager.Register(container); err != nil {
		t.Fatalf("expected nil got %s", err)
	}

	if err := testManager.Register(container2); err != nil {
		t.Fatalf("expected nil got %s", err)
	}

	if err := testManager.Start(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fail()
	}
	time.Sleep(time.Second * 10)

	testManager.Stop()
}
