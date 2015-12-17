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

package isvcs

import (
	"os"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/utils"

	"testing"
	"time"
)


func TestMain(m *testing.M) {
	docker.StartKernel()
	os.Exit(m.Run())
}

func TestManager(t *testing.T) {
	testManager := NewManager(utils.LocalDir("images"), "/tmp", defaultTestDockerLogDriver, defaultTestDockerLogOptions)

	if err := testManager.Start(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fatal()
	}

	cd1 := IServiceDefinition{
		ID:      "test1",
		Name:    "test1",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: func() string { return `while true; do echo hello world; sleep 1; done` },
	}
	container, err := NewIService(cd1)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fatal()
	}

	cd2 := IServiceDefinition{
		ID:      "test2",
		Name:    "test2",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: func() string { return `while true; do echo hello world; sleep 1; done` },
	}
	container2, err := NewIService(cd2)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fatal()
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
