/*******************************************************************************
* Copyright (C) Zenoss, Inc. 2013, 2014, all rights reserved.
*
* This content is made available according to terms specified in
* License.zenoss under the directory where your Zenoss product is installed.
*
*******************************************************************************/

package isvcs

import (
	"testing"
	"time"
)

func TestManager(t *testing.T) {
	testManager := NewManager("unix:///var/run/docker.sock", "/tmp")

	if err := testManager.Stop(); err != ErrManagerNotRunning {
		t.Logf("expected an error got %s", err)
		t.Fail()
	}

	if err := testManager.Start(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fail()
	}

	cd1 := ContainerDescription{
		Name:    "test1",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: `/bin/sh -c "while true; do echo hello world; sleep 1; done"`,
	}
	container, err := NewContainer(cd1)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fail()
	}

	cd2 := ContainerDescription{
		Name:    "test2",
		Repo:    "ubuntu",
		Tag:     "latest",
		Command: `/bin/sh -c "while true; do echo hello world; sleep 1; done"`,
	}
	container2, err := NewContainer(cd2)
	if err != nil {
		t.Logf("could not create container: %s", err)
		t.Fail()
	}

	if err := testManager.Stop(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fail()
	}

	if err := testManager.Register(container); err != nil {
		t.Logf("expected nil got %s", err)
		t.Fatal()
	}

	if err := testManager.Register(container2); err != nil {
		t.Logf("expected nil got %s", err)
		t.Fatal()
	}

	if err := testManager.Start(); err != nil {
		t.Logf("expected no error got %s", err)
		t.Fail()
	}
	time.Sleep(time.Second * 10)

	testManager.Stop()
}
