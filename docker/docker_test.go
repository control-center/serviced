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

	cid, err := StartContainer(cd, 600*time.Second, func(id string) error {
		started = true
		return nil
	})
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	StopContainer(cid, 30)

	if !started {
		t.Fatal("OnContainerStart handler was not triggered")
	}
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

	cid, err := StartContainer(cd, 600*time.Second, nil)
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
