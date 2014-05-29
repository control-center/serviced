package docker

import (
	"testing"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
)

func TestOnContainerStart(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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

	err = ctr.Start(30*time.Second, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	ctr.Kill()
}

func TestOnContainerCreated(t *testing.T) {
	cs := make(chan string)

	OnContainerCreated(Wildcard, func(id string) {
		cs <- id
	})

	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	select {
	case <-cs:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	ctr.Kill()
}

func TestOnContainerStop(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := NewContainer(cd, true, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	ec := make(chan struct{})

	ctr.OnEvent(Stop, func(cid string) {
		ec <- struct{}{}
	})

	ctr.Stop(30)

	select {
	case <-ec:
	case <-time.After(30 * time.Second):
		t.Fatal("Timed out waiting for event")
	}
}

func TestCancelOnEvent(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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

	ctr.Start(1*time.Second, nil)

	select {
	case <-ec:
		t.Fatal("OnEvent fired")
	case <-time.After(2 * time.Second):
		// success
	}

	ctr.Kill()
}

func TestRestartContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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

	ctr.OnEvent(Restart, func(id string) {
		restartch <- struct{}{}
	})

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

	ctr.Kill()
}

func TestListContainers(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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
	case <-time.After(5 * time.Second):
		t.Fatal("timed out waiting for wait to finish")
	}
}

func TestInspectContainer(t *testing.T) {
	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
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

	if err := ctr.Start(1*time.Second, nil); err != nil {
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
}
