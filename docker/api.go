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

type ContainerActionFunc func(cid string) error

const (
	emptystr = ""
)

var (
	ErrRequestTimeout = errors.New("docker: request timed out")
	ErrKernelShutdown = errors.New("docker: kernel shutdown")
)

// InspectContainer returns information about the container specified by id.
func InspectContainer(id string) (*dockerclient.Container, error) {
	ec := make(chan error)
	rc := make(chan *dockerclient.Container)

	cmds.Inspect <- inspectreq{
		request{ec, 0 * time.Second},
		struct {
			id string
		}{id},
		rc,
	}

	select {
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

// StartContainer uses the information provided in the container definition cd to create and start a new Docker
// container. If a container can't be started before the timeout expires an error is returned. After the container
// is successfully started the onstart action function is executed.
func StartContainer(cd *ContainerDefinition, timeout time.Duration, onstart ContainerActionFunc) (string, error) {
	ec := make(chan error)
	rc := make(chan string)

	cmds.Start <- startreq{
		request{ec, timeout},
		struct {
			ContainerOptions *dockerclient.CreateContainerOptions
			HostConfig       *dockerclient.HostConfig
			ActionFunc       ContainerActionFunc
		}{&cd.CreateContainerOptions, &cd.HostConfig, onstart},
		rc,
	}

	select {
	case <-time.After(timeout):
		return emptystr, ErrRequestTimeout
	case <-done:
		return emptystr, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return <-rc, nil
		default:
			return emptystr, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// StopContainer stops the container specified by the id. If the container can't be stopped before the timeout
// expires an error is returned.
func StopContainer(id string, timeout time.Duration) error {
	ec := make(chan error)

	cmds.Stop <- stopreq{
		request{ec, timeout},
		struct {
			id string
		}{id},
	}

	select {
	case <-time.After(timeout):
		return ErrRequestTimeout
	case <-done:
		return ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: request failed: %v", err)
		}
	}
}
