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
	. "gopkg.in/check.v1"
)

func TestMain(m *testing.M) {
	StartKernel()
	defer func() { done <- struct{}{} }()
	os.Exit(m.Run())
}

func (s *TestDockerSuite) TestContainerCommit(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	c.Assert(err, IsNil)

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	err = ctr.Start()
	c.Assert(err, IsNil)

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for event")
	}

	_, err = ctr.Commit("testcontainer/commit", false)
	c.Assert(err, IsNil)

	ctr.Kill()
	ctr.Delete(true)

	cmd := []string{"docker", "rmi", "testcontainer/commit"}
	exec.Command(cmd[0], cmd[1:]...).Run()
}

func (s *TestDockerSuite) TestOnContainerStart(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	c.Assert(err, IsNil)

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	err = ctr.Start()
	c.Assert(err, IsNil)

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for event")
	}

	if !ctr.IsRunning() {
		c.Fatal("expected container to be running")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func (s *TestDockerSuite) TestOnContainerStop(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 600*time.Second, nil, nil)
	c.Assert(err, IsNil)

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
		c.Fatal(waitErr)
	case exitcode := <-ec:
		c.Logf("Received exit code: %d", exitcode)
	case <-time.After(30 * time.Second):
		c.Fatal("Timed out waiting for event")
	}
}

func (s *TestDockerSuite) TestCancelOnEvent(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	c.Assert(err, IsNil)

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
		c.Fatal("OnEvent fired")
	case <-time.After(2 * time.Second):
		// success
	}

	ctr.Kill()
	ctr.Delete(true)
}

func (s *TestDockerSuite) TestRestartContainer(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 600*time.Second, nil, nil)
	c.Assert(err, IsNil)

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
		c.Fatal("Timed out waiting for container to stop/die")
	}

	select {
	case <-restartch:
	case <-time.After(10 * time.Second):
		c.Fatal("Timed out waiting for Start event")
	}

	ctr.CancelOnEvent(Die)
	ctr.CancelOnEvent(Restart)
	ctr.Kill()
	ctr.Delete(true)

}

func (s *TestDockerSuite) TestListContainers(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctrs := []*Container{}

	for i := 0; i < 4; i++ {
		ctr, err := NewContainer(cd, true, 300*time.Second, nil, nil)
		c.Assert(err, IsNil)

		ctrs = append(ctrs, ctr)
	}

	cl, err := Containers()
	c.Assert(err, IsNil)

	if (len(cl) - len(ctrs)) < 0 {
		c.Fatalf("expecting at least %d containers, found %d", len(ctrs), len(cl))
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
			c.Fatal("missing container: ", ctr.ID)
		}
	}

	for _, ctr := range ctrs {
		ctr.Kill()
	}

	for _, ctr := range ctrs {
		ctr.Delete(true)
	}
}

func (s *TestDockerSuite) TestWaitForContainer(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)

	wc := make(chan int)

	go func(ctr *Container) {
		rc, err := ctr.Wait(600 * time.Second)
		c.Assert(err, IsNil)

		wc <- rc
	}(ctr)

	time.Sleep(10 * time.Second)

	if err := ctr.Kill(); err != nil {
		c.Fatal("can't kill container: ", err)
	}

	select {
	case <-wc:
		// success
		ctr.Delete(true)
	case <-time.After(5 * time.Second):
		c.Fatal("timed out waiting for wait to finish")
	}
}

func (s *TestDockerSuite) TestInspectContainer(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)

	prestart, err := ctr.Inspect()
	c.Assert(err, IsNil)

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	if err := ctr.Start(); err != nil {
		c.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
		break
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for container to start")
	}

	poststart, err := ctr.Inspect()
	c.Assert(err, IsNil)

	if poststart.State.Running == prestart.State.Running {
		c.Fatal("inspected stated didn't change")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func (s *TestDockerSuite) TestRepeatedStart(c *C) {
	c.Skip("skip this until the build box issues get sorted out")
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)

	sc := make(chan struct{})

	ctr.OnEvent(Start, func(id string) {
		sc <- struct{}{}
	})

	if err := ctr.Start(); err != nil {
		c.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
		break
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for container to start")
	}

	if err := ctr.Start(); err == nil {
		c.Fatal("expecting ErrAlreadyStarted")
	}

	ctr.Kill()
	ctr.Delete(true)
}

func (s *TestDockerSuite) TestNewContainerOnCreatedAndStartedActions(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	createActionChan := make(chan struct{})
	startActionChan := make(chan struct{})

	createAction := func(id string) {
		createActionChan <- struct{}{}
	}

	startAction := func(id string) {
		startActionChan <- struct{}{}
	}

	var ctr *Container
	createdChan := make(chan struct{})
	go func() {
		var err error

		ctr, err = NewContainer(cd, true, 300*time.Second, createAction, startAction)
		c.Assert(err, IsNil)

		createdChan <- struct{}{}
	}()

	select {
	case <-createActionChan:
		break
	case <-time.After(360 * time.Second):
		c.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-startActionChan:
		break
	case <-time.After(30 * time.Second):
		c.Fatal("timed out waiting for start action execution")
	}

	select {
	case <-createdChan:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func (s *TestDockerSuite) TestNewContainerOnCreatedAction(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
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
		c.Assert(err, IsNil)
		ctrCreated <- struct{}{}
	}()

	select {
	case <-cc:
		break
	case <-time.After(360 * time.Second):
		c.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-ctrCreated:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func (s *TestDockerSuite) TestNewContainerOnStartedAction(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
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
		c.Assert(err, IsNil)

		ctrCreated <- struct{}{}
	}()

	select {
	case <-sc:
		break
	case <-time.After(360 * time.Second):
		c.Fatal("timed out waiting for create action execution")
	}

	select {
	case <-ctrCreated:
		ctr.Kill()
		ctr.Delete(true)
		break
	case <-time.After(10 * time.Second):
		c.Fatal("timed out waiting for NewContainer to return a ctr")
	}
}

func (s *TestDockerSuite) TestFindContainer(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctrone, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)

	if ctr2, err := NewContainer(cd, false, 300*time.Second, nil, nil); err != nil {
		c.Fatal("can't create second container: ", err)
	} else {
		defer ctr2.Delete(true)
	}
	cid := ctrone.ID

	ctr, err := FindContainer(cid)
	c.Assert(err, IsNil)
	c.Assert(ctr.ID, Equals, ctrone.ID)

	if err := ctrone.Delete(true); err != nil {
		c.Fatal("can't delete container: ", err)
	}

	if _, err = FindContainer(cid); err == nil {
		c.Fatal("should not have found container: ", cid)
	}
}

// TODO: add some additional Export tests, e.g., bogus path, insufficient permissions, etc.
func (s *TestDockerSuite) TestContainerExport(c *C) {
	cd := &dockerclient.CreateContainerOptions{
		Config: &dockerclient.Config{
			Image: "ubuntu:latest",
			Cmd:   []string{"sleep", "3600"},
		},
		HostConfig: &dockerclient.HostConfig{},
	}

	ctrone, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)
	defer ctrone.Delete(true)

	ctrtwo, err := NewContainer(cd, false, 300*time.Second, nil, nil)
	c.Assert(err, IsNil)
	defer ctrtwo.Delete(true)

	cf, err := ioutil.TempFile("/tmp", "containertest")
	c.Assert(err, IsNil)

	err = ctrone.Export(cf)
	c.Assert(err, IsNil)

	err = os.Remove(cf.Name())
	c.Assert(err, IsNil)
}
