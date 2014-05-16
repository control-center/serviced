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

// CreateContainer creates a new container and returns its id. The supplied action function, if
// any, will be executed on successful creation of the container.
func CreateContainer(cd *ContainerDefinition, start bool, timeout time.Duration, oncreate ContainerActionFunc, onstart ContainerActionFunc) (string, error) {
	ec := make(chan error)
	rc := make(chan string)

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

// InspectContainer returns information about the container specified by id.
func InspectContainer(id string) (*dockerclient.Container, error) {
	ec := make(chan error)
	rc := make(chan *dockerclient.Container)

	cmds.Inspect <- inspectreq{
		request{ec},
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

// ListContainers returns a list (slice) of known Docker container ids.
func ListContainers() ([]string, error) {
	ec := make(chan error)
	rc := make(chan []string)

	cmds.List <- listreq{
		request{ec},
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

// StartContainer uses the information provided in the container definition cd to start a new Docker
// container. If a container can't be started before the timeout expires an error is returned. After the container
// is successfully started the onstart action function is executed.
func StartContainer(id string, cd *ContainerDefinition, timeout time.Duration, onstart ContainerActionFunc) error {
	ec := make(chan error)

	cmds.Start <- startreq{
		request{ec},
		struct {
			id         string
			hostConfig *dockerclient.HostConfig
			action     ContainerActionFunc
		}{id, &cd.HostConfig, onstart},
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

// StopContainer stops the container specified by the id. If the container can't be stopped before the timeout
// expires an error is returned.
func StopContainer(id string, timeout time.Duration) error {
	ec := make(chan error)

	cmds.Stop <- stopreq{
		request{ec},
		struct {
			id      string
			timeout uint
		}{id, 10},
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

// OnContainerCreated associates a containter action with the specified container. The action will be triggered when
// that container is created; since we can't know before it's created what a containers id will be the only really
// useful id is docker.Wildcard which will cause the action to be triggered for every container docker creates.
func OnContainerCreated(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Create, id, action)
}

// OnContainerDeleted associates a container action with the specified container. The action will be triggered when
// that container is deleted.
func OnContainerDeleted(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Delete, id, action)
}

// OnContainerDestroy associates a container action with the specified container. The action will be triggered when
// that container is destroyed.
func OnContainerDestroy(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Destroy, id, action)
}

// OnContainerDeath associates a container action with the specified container. The action will be triggered when
// that container dies.
func OnContainerDeath(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Die, id, action)
}

// OnContainerExport associates a container action with the specified container. The action will be triggered when
// that container is exported.
func OnContainerExport(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Export, id, action)
}

// OnContainerKill associates a container action with the specified container. The action will be triggered when
// that container is killed.
func OnContainerKill(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Kill, id, action)
}

// OnContainerRestart associates a container action with the specified container. The action will be triggered when
// that container is restarted.
func OnContainerRestart(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Restart, id, action)
}

// OnContainerStart associates a container action with the specified container. The action will be triggered when
// that container is started.
func OnContainerStart(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Start, id, action)
}

// OnContainerStop associates a container action with the specified container. The action will be triggered when
// that container is stopped.
func OnContainerStop(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Stop, id, action)
}

// OnContainerUntagged associates a container action with the specified container. The action will be triggered when
// that container is untagged.
func OnContainerUntagged(id string, action ContainerActionFunc) error {
	return onContainerEvent(dockerclient.Untag, id, action)
}

// CancelOnContainerCreated cancels any OnContainerCreated action associated with the specified id - docker.Wildcard is
// the only id that really makes sense.
func CancelOnContainerCreated(id string) error {
	return cancelOnContainerEvent(dockerclient.Create, id)
}

// CancelOnContainerDeleted cancels any OnContainerDeleted action asssociated with the specified id.
func CancelOnContainerDeleted(id string) error {
	return cancelOnContainerEvent(dockerclient.Delete, id)
}

// CancelOnContainerDestroy cancels any OnContainerDestroy action asssociated with the specified id.
func CancelOnContainerDestroy(id string) error {
	return cancelOnContainerEvent(dockerclient.Destroy, id)
}

// CancelOnContainerDeath cancels any OnContainerDeath action associated with the specified id.
func CancelOnContainerDeath(id string) error {
	return cancelOnContainerEvent(dockerclient.Die, id)
}

// CancelOnContainerExport cancels any OnContainerExport action associated with the specified id.
func CancelOnContainerExport(id string) error {
	return cancelOnContainerEvent(dockerclient.Export, id)
}

// CancelOnContainerKill cancels any OnContainerKill action associated with the specified id.
func CancelOnContainerKill(id string) error {
	return cancelOnContainerEvent(dockerclient.Kill, id)
}

// CancelOnContainerRestart cancels any OnContainerRestart action associated with the specified id.
func CancelOnContainerRestart(id string) error {
	return cancelOnContainerEvent(dockerclient.Restart, id)
}

// CancelOnContainerStart cancels any OnContainerStart action associated with the specified id.
func CancelOnContainerStart(id string) error {
	return cancelOnContainerEvent(dockerclient.Start, id)
}

// CancelOnContainerStop cancels any OnContainerStop action associated with the specified id.
func CancelOnContainerStop(id string) error {
	return cancelOnContainerEvent(dockerclient.Stop, id)
}

// CancelOnContainerUntagged cancels any OnContainerUntagged action associated with the specified id.
func CancelOnContainerUntagged(id string) error {
	return cancelOnContainerEvent(dockerclient.Untag, id)
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
