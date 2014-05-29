package docker

import (
	"errors"
	"fmt"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
)

type Container struct {
	dockerclient.Container
	dockerclient.HostConfig
}

type ContainerDefinition struct {
	dockerclient.CreateContainerOptions
	dockerclient.HostConfig
}

type ContainerActionFunc func(id string)

const (
	Create  = dockerclient.Create
	Delete  = dockerclient.Delete
	Destroy = dockerclient.Destroy
	Die     = dockerclient.Die
	Export  = dockerclient.Export
	Kill    = dockerclient.Kill
	Restart = dockerclient.Restart
	Start   = dockerclient.Start
	Stop    = dockerclient.Stop
	Untag   = dockerclient.Untag
)

var (
	ErrRequestTimeout = errors.New("docker: request timed out")
	ErrKernelShutdown = errors.New("docker: kernel shutdown")
)

// NewContainer creates a new container and returns its id. The supplied action function, if
// any, will be executed on successful creation of the container.
func NewContainer(cd *ContainerDefinition, start bool, timeout time.Duration, oncreate ContainerActionFunc, onstart ContainerActionFunc) (*Container, error) {
	ec := make(chan error)
	rc := make(chan dockerclient.Container)

	cmds.Create <- createreq{
		request{ec},
		struct {
			containerOptions *dockerclient.CreateContainerOptions
			hostConfig       *dockerclient.HostConfig
			start            bool
			createaction     ContainerActionFunc
			startaction      ContainerActionFunc
		}{&cd.CreateContainerOptions, &cd.HostConfig, start, oncreate, onstart},
		rc,
	}

	select {
	case <-time.After(timeout):
		return nil, ErrRequestTimeout
	case <-done:
		return nil, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return &Container{<-rc, cd.HostConfig}, nil
		default:
			return nil, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// Containers retrieves a list of all the Docker containers.
func Containers() ([]*Container, error) {
	ec := make(chan error)
	rc := make(chan []*Container)

	cmds.List <- listreq{
		request{ec},
		rc,
	}

	select {
	case <-done:
		return []*Container{}, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return <-rc, nil
		default:
			return []*Container{}, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// CancelOnEvent cancels the action associated with the specified event.
func (c *Container) CancelOnEvent(event string) error {
	return cancelOnContainerEvent(event, c.ID)
}

// Delete removes the container.
func (c *Container) Delete(volumes bool) error {
	ec := make(chan error)

	cmds.Delete <- deletereq{
		request{ec},
		struct {
			removeOptions dockerclient.RemoveContainerOptions
		}{dockerclient.RemoveContainerOptions{c.ID, volumes}},
	}

	select {
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

// Kill sends a SIGKILL signal to the container. If the container is not started
// no action is taken.
func (c *Container) Kill() error {
	ec := make(chan error)

	cmds.Kill <- killreq{
		request{ec},
		struct {
			id string
		}{c.ID},
	}

	select {
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

// Inspect returns information about the container specified by id.
func (c Container) Inspect() *dockerclient.Container {
	return &c.Container
}

// OnEvent adds an action for the specified event.
func (c *Container) OnEvent(event string, action ContainerActionFunc) error {
	return onContainerEvent(event, c.ID, action)
}

// Restart stops and then restarts a container.
func (c *Container) Restart(timeout time.Duration) error {
	ec := make(chan error)

	cmds.Restart <- restartreq{
		request{ec},
		struct {
			id      string
			timeout uint
		}{c.ID, uint(timeout)},
	}

	select {
	case <-time.After(timeout):
		return nil
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

// Start uses the information provided in the container definition cd to start a new Docker
// container. If a container can't be started before the timeout expires an error is returned. After the container
// is successfully started the onstart action function is executed.
func (c *Container) Start(timeout time.Duration, onstart ContainerActionFunc) error {
	ec := make(chan error)

	cmds.Start <- startreq{
		request{ec},
		struct {
			id         string
			hostConfig *dockerclient.HostConfig
			action     ContainerActionFunc
		}{c.ID, &c.HostConfig, onstart},
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

// Stop stops the container specified by the id. If the container can't be stopped before the timeout
// expires an error is returned.
func (c *Container) Stop(timeout time.Duration) error {
	ec := make(chan error)

	cmds.Stop <- stopreq{
		request{ec},
		struct {
			id      string
			timeout uint
		}{c.ID, 10},
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

// Wait blocks until the container stops or the timeout expires and then returns its exit code.
func (c *Container) Wait(timeout time.Duration) (int, error) {
	ec := make(chan error)
	rc := make(chan int)

	cmds.Wait <- waitreq{
		request{ec},
		struct {
			id string
		}{c.ID},
		rc,
	}

	select {
	case <-time.After(timeout):
		return -127, ErrRequestTimeout
	case <-done:
		return -127, ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return <-rc, nil
		default:
			return -127, fmt.Errorf("docker: request failed: %v", err)
		}
	}
}

// OnContainerCreated associates a containter action with the specified container. The action will be triggered when
// that container is created; since we can't know before it's created what a containers id will be the only really
// useful id is docker.Wildcard which will cause the action to be triggered for every container docker creates.
func OnContainerCreated(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Create, id, action)
}

// CancelOnContainerCreated cancels any OnContainerCreated action associated with the specified id - docker.Wildcard is
// the only id that really makes sense.
func CancelOnContainerCreated(id string) error {
	return cancelOnContainerEvent(dockerclient.Create, id)
}

func onContainerEvent(event, id string, action ContainerActionFunc) error {
	ec := make(chan error)

	cmds.AddAction <- addactionreq{
		request{ec},
		struct {
			id     string
			event  string
			action ContainerActionFunc
		}{id, event, action},
	}

	select {
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

func cancelOnContainerEvent(event, id string) error {
	ec := make(chan error)

	cmds.CancelAction <- cancelactionreq{
		request{ec},
		struct {
			id    string
			event string
		}{id, event},
	}

	select {
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
