package docker

import (
	"fmt"
	"os"
	"syscall"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	dockerep = "unix:///var/run/docker.sock"
)

type request struct {
	errchan chan error
	timeout time.Duration
}

type inspectreq struct {
	request
	args struct {
		id string
	}
	respchan *dockerclient.Container
}

type listreq struct {
	request
	respchan chan []string
}

type startreq struct {
	request
	args struct {
		ContainerOptions *dockerclient.CreateContainerOptions
		HostConfig       *dockerclient.HostConfig
	}
	respchan chan string
}

var (
	cmds = struct {
		Inspect chan inspectreq
		List    chan listreq
		Start   chan startreq
	}{
		make(chan inspectreq),
		make(chan listreq),
		make(chan startreq),
	}
	done = make(chan struct{})
)

func init() {
	client, err := dockerclient.NewClient(dockerep)
	if err != nil {
		panic(fmt.Sprintf("can't create Docker client: %v", err))
	}

	go kernel(client, done)
}

func kernel(dc *dockerclient.Client, done chan struct{}) error {
	em, err := dc.MonitorEvents()
	if err != nil {
		panic(fmt.Sprintf("can't monitor Docker events: %v", err))
	}

	for {
		select {
		case ir := <-cmds.Inspect:
			ctr, err := dc.InspectContainer(ir.args.id)
			if err != nil {
				ir.errchan <- err
				continue
			}
			close(ir.errchan)
			ir.respchan <- ctr
		case lr := <-cmds.List:
			apictrs, err := dc.ListContainers(dockerclient.ListContainersOptions{All: true})
			if err != nil {
				lr.errchan <- err
				continue
			}
			resp := []string{}
			for _, apictr := range apictrs {
				resp = append(resp, apictr.ID)
			}
			close(lr.errchan)
			lr.respchan <- resp
		case sr := <-cmds.Start:
			ctr, err := dc.CreateContainer(*sr.args.ContainerOptions)
			switch {
			case err == dockerclient.ErrNoSuchImage:
				if pullerr := dc.PullImage(dockerclient.PullImageOptions{
					Repository:   sr.args.ContainerOptions.Config.Image,
					OutputStream: os.NewFile(uintptr(syscall.Stdout), "/def/stdout"),
				}, dockerclient.AuthConfiguration{}); pullerr != nil {
					sr.errchan <- err
					continue
				}

				ctr, err = dc.CreateContainer(*sr.args.ContainerOptions)
				if err != nil {
					sr.errchan <- err
					continue
				}
			case err != nil:
				sr.errchan <- err
				continue
			}

			go func(cid string, hc *dockerclient.HostConfig, rc chan string, ec chan error, timeout time.Duration) {
				s, err := em.Subscribe(cid)
				if err != nil {
					ec <- fmt.Errorf("can't subscribe to Docker events on container %s: %v", cid, err)
				}

				emc := make(chan struct{})

				s.Handle(dockerclient.Start, func(e dockerclient.Event) error {
					emc <- struct{}{}
					return nil
				})

				err = dc.StartContainer(cid, hc)
				if err != nil {
					ec <- err
				} else {
					select {
					case <-emc:
						close(ec)
						rc <- cid
					case <-time.After(timeout):
						ec <- fmt.Errorf("container start timed out for Docker container: %s", cid)
					}
				}
			}(ctr.ID, sr.args.HostConfig, sr.respchan, sr.errchan, sr.request.timeout)
		case <-done:
			return nil
		}
	}
}
