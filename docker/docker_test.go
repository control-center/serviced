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

	ctr, err := CreateContainer(cd, false, 600*time.Second, nil, nil)
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

	ctr.Stop(30)
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

	ctr, err := CreateContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	select {
	case <-cs:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	ctr.Stop(30)
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

	ctr, err := CreateContainer(cd, true, 600*time.Second, nil, nil)
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
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for event")
	}
}
