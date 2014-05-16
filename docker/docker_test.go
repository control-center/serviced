package docker

import (
	"testing"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
)

func TestOnContainerStart(t *testing.T) {
	var started bool

	cd := &ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Image: "base",
				Cmd:   []string{"/bin/sh", "-c", "while true; do echo hello world; sleep 1; done"},
			},
		},
		dockerclient.HostConfig{},
	}

	cid, err := CreateContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	sc := make(chan struct{})

	OnContainerStart(cid, func(id string) error {
		sc <- struct{}{}
		return nil
	})

	err = StartContainer(cid, cd, 30*time.Second, func(id string) error {
		started = true
		return nil
	})
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	select {
	case <-sc:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	StopContainer(cid, 30)
}

func TestOnContainerCreated(t *testing.T) {
	cs := make(chan string)

	OnContainerCreated(Wildcard, func(id string) error {
		cs <- id
		return nil
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

	cid, err := CreateContainer(cd, false, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't create container: ", err)
	}

	select {
	case <-cs:
	case <-time.After(10 * time.Second):
		t.Fatal("Timed out waiting for event")
	}

	StopContainer(cid, 30)
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

	cid, err := CreateContainer(cd, true, 600*time.Second, nil, nil)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	ec := make(chan struct{})

	OnContainerStop(cid, func(cid string) error {
		ec <- struct{}{}
		return nil
	})

	StopContainer(cid, 30)

	select {
	case <-ec:
	case <-time.After(1 * time.Second):
		t.Fatal("Timed out waiting for event")
	}
}
