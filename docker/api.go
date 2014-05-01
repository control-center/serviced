package docker

import (
	"errors"
	"fmt"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
)

type ContainerDefinition struct {
	dockerclient.CreateContainerOptions
	dockerclient.HostConfig
}

type HandlerFunc func()

const (
	nilstr = ""
)

var (
	ErrRequestTimeout = errors.New("docker: request timed out")
	ErrKernelShutdown = errors.New("docker: kernel shutdown")
)

func InspectContainer(id string) (*dockerclient.Container, error) {
	ec := make(chan error)
	rc := make(chan *dockerclient.Container)

	cmds.Inspect <- inspectreq{
		request{ec, 1 * time.Second},
		struct {
			id string
		}{id},
		rc,
	}

	select {
	case <-time.After(1 * time.Second):
		return nil, ErrRequestTimeout
	case <-done:
		return nil, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return <-rc, nil
		default:
			return nil, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

func StartContainer(cd *ContainerDefinition) (string, error) {
	ec := make(chan error)
	rc := make(chan string)

	cmds.Start <- startreq{
		request{ec, 30 * time.Second},
		struct {
			ContainerOptions *dockerclient.CreateContainerOptions
			HostConfig       *dockerclient.HostConfig
		}{&cd.CreateContainerOptions, &cd.HostConfig},
		rc,
	}

	select {
	case <-time.After(10 * time.Second):
		return nilstr, ErrRequestTimeout
	case <-done:
		return nilstr, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return <-rc, nil
		default:
			return nilstr, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

func OnContainerStart(props map[string]string, h HandlerFunc) {
}
