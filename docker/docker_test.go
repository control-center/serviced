package docker

import (
	"testing"

	dockerclient "github.com/zenoss/go-dockerclient"
)

func TestOnContainerStart(t *testing.T) {
	props := map[string]string{"Image": "base"}
	var started bool

	OnContainerStart(props, func() {
		started = true
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

	cid, err := StartContainer(cd)
	if err != nil {
		t.Fatal("can't start container: ", err)
	}

	StopContainer(cid)

	if !started {
		t.Fatal("OnContainerStart handler was not triggered")
	}
}
