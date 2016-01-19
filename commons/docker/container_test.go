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

// +build integration,!quick

package docker

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"testing"
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
)

func TestMain(m *testing.M) {
	StartKernel()
	defer func() { done <- struct{}{} }()
	os.Exit(m.Run())
}

func TestContainerCommit(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	err = ctr.Start()
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	_, err = ctr.Commit("testcontainer/commit", false)
	if err != nil {
		t.Fatal("can't commit: ", err)
	}

	ctr.Kill()
	ctr.Delete(true)

	cmd := []string{"docker", "rmi", "testcontainer/commit"}
	exec.Command(cmd[0], cmd[1:]...).Run()
}

func TestOnContainerStart(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	err = ctr.Start()
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	if !ctr.IsRunning() {
		t.Fatal("expected container to be running")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func TestOnContainerStop(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	ec := make(chan int)
	waitErrC := make(chan string)

	ctr.OnEvent(Die, func(cid string) {
		exitcode, err := ctr.Wait(1 * time.Second)
		if err != nil {
			waitErrC <- fmt.Sprintf("Error waiting for container to exit: %s", err)
			return
		}
		ec <- exitcode
	})

	ctr.Stop(30)
	defer ctr.Delete(true)

	select {
	case waitErr := <-waitErrC:
		t.Fatal(waitErr)
	case exitcode := <-ec:
		t.Logf("Received exit code: %d", exitcode)
	case <-time.After(30 * time.Second):
		t.Fatal("Timed out waiting for event")
	}
}

func TestCancelOnEvent(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	ec := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		ec <- struct{}{}
	})

	ctr.OnEvent(Stop, func(id string) {
		ec <- struct{}{}
	})

	ctr.CancelOnEvent(Start)

	ctr.Start()

	select {
	case <-ec:
		t.Fatal("OnEvent fired")
	case <-time.After(2 * time.Second):
		// success
	}

	ctr.Kill()
	ctr.Delete(true)
}

func TestRestartContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	restartch := make(chan struct{})
	diech := make(chan struct{})

	ctr.OnEvent(Die, func(id string) {
		diech <- struct{}{}
	})
	defer ctr.CancelOnEvent(Die)

	ctr.OnEvent(Restart, func(id string) {
		restartch <- struct{}{}
	})
	defer ctr.CancelOnEvent(Restart)

	ctr.Restart(10 * time.Second)

	select {
	case <-diech:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for container to stop/die")
	}

	select {
	case <-restartch:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for Start event")
	}

	ctr.CancelOnEvent(Die)
	ctr.CancelOnEvent(Restart)
	ctr.Kill()
	ctr.Delete(true)

}

func TestListContainers(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctrs := []*Container{}

	for i := 0; i < 4; i++ {
		ctr, err := NewContainer(cd, true, 300*time.Second, nil, nil)
		if err != nil {
			t.Fatal("can't create container: ", err)
		}

		ctrs = append(ctrs, ctr)
	}

	cl, err := Containers()
	if err != nil {
		t.Fatal("can't get a list of containers: ", err)
	}

	if (len(cl) - len(ctrs)) < 0 {
		t.Fatalf("expecting at least %d containers, found %d", len(ctrs), len(cl))
	}

	for _, ctr := range ctrs {
		var found bool
		for _, c := range cl {
			if ctr.ID == c.ID {
				found = true
				break
			}
		}

		if !found {
			t.Fatal("missing container: ", ctr.ID)
		}
	}

	for _, ctr := range ctrs {
		ctr.Kill()
	}

	for _, ctr := range ctrs {
		ctr.Delete(true)
	}
}

func TestWaitForContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	wc := make(chan int)

	go func(c *Container) {
		rc, err := c.Wait(600 * time.Second)
		if err != nil {
			t.Log("container wait failed: ", err)
		}

		wc <- rc
	}(ctr)

	time.Sleep(10 * time.Second)

	if err := ctr.Kill(); err != nil {
		t.Fatal("can't kill container: ", err)
	}

	select {
	case <-wc:
		// success
		ctr.Delete(true)
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for wait to finish")
	}
}

func TestInspectContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	prestart, err := ctr.Inspect()
	if err != nil {
		t.Fatal("can't pre inspect container: ", err)
	}

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	if err := ctr.Start(); err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
		break
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for container to start")
	}

	poststart, err := ctr.Inspect()
	if err != nil {
		t.Fatal("can't post inspect container: ", err)
	}

	if poststart.State.Running == prestart.State.Running {
		t.Fatal("inspected stated didn't change")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func TestRepeatedStart(t *testing.T) {
	t.Skip("skip this until the build box issues get sorted out")
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	if err := ctr.Start(); err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
		break
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for container to start")
	}

	if err := ctr.Start(); err == nil {
		t.Fatal("expecting ErrAlreadyStarted")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func TestNewContainerOnCreatedAndStartedActions(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	cc := make(chan struct{})
	sc := make(chan struct{})

	ca := func(id string) {
		cc <- struct{}{}
	}

	sa := func(id string) {
		sc <- struct{}{}
	}

	var ctr *Container
	ctrCreated := make(chan struct{})
	go func() {
		var err error

		ctr, err = NewContainer(cd, true, 300*time.Second, ca, sa)
		if err != nil {
			t.Fatal("can't create container: ", err)
		}

		ctrCreated <- struct{}{}
	}()

	select {
	case <-cc:
		break
	case <-time.After(360 * time.Second):
		t.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-sc:
		break
	case <-time.After(1 * time.Second):
		t.Fatal("timed out waiting for start action execution")
	}

	select {
	case <-ctrCreated:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func TestNewContainerOnCreatedAction(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	cc := make(chan struct{})

	ca := func(id string) {
		cc <- struct{}{}
	}

	var ctr *Container
	ctrCreated := make(chan struct{})
	go func() {
		var err error
		ctr, err = NewContainer(cd, false, 300*time.Second, ca, nil)
		if err != nil {
			t.Fatal("can't create container: ", err)
		}
		ctrCreated <- struct{}{}
	}()

	select {
	case <-cc:
		break
	case <-time.After(360 * time.Second):
		t.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-ctrCreated:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func TestNewContainerOnStartedAction(t *testing.T) {
	//t.Skip("skip this until the build box issues get sorted out")
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	sc := make(chan struct{})

	sa := func(id string) {
		sc <- struct{}{}
	}

	var ctr *Container
	ctrCreated := make(chan struct{})
	go func() {
		var err error

		ctr, err = NewContainer(cd, true, 300*time.Second, nil, sa)
		if err != nil {
			t.Fatal("can't create container: ", err)
		}

		ctrCreated <- struct{}{}
	}()

	select {
	case <-sc:
		break
	case <-time.After(360 * time.Second):
		t.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-ctrCreated:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		t.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func TestFindContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctrone, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	if ctr2, err := NewContainer(cd, false, 300*time.Second, nil, nil); err != nil {
		t.Fatal("can't create second container: ", err)
	} else {
		defer ctr2.Delete(true)
	}
	cid := ctrone.ID

	ctr, err := FindContainer(cid)
	if err != nil {
		t.Fatalf("can't find container %s: %v", cid, err)
	}

	if ctrone.ID != ctr.ID {
		t.Fatalf("container names don't match; got %s, expecting %s", ctr.Name, ctrone.Name)
	}

	if err := ctrone.Delete(true); err != nil {
		t.Fatal("can't delete container: ", err)
	}

	if _, err = FindContainer(cid); err == nil {
		t.Fatal("should not have found container: ", cid)
	}
}

// TODO: add some additional Export tests, e.g., bogus path, insufficient permissions, etc.
func TestContainerExport(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "ubuntu:latest",
				Cmd:   []string{"sleep", "3600"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctrone, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}
	defer ctrone.Delete(true)

	ctrtwo, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create second container: ", err)
	}
	defer ctrtwo.Delete(true)

	cf, err := ioutil.TempFile("/tmp", "containertest")
	if err != nil {
		t.Fatal("can't create temp file for export: ", err)
	}

	err = ctrone.Export(cf)
	if err != nil {
		t.Fatal("can't export container: ", err)
	}

	err = os.Remove(cf.Name())
	if err != nil {
		t.Fatalf("can't remove %s: %v", cf.Name(), err)
	}
}
