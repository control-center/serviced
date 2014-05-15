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
}

type inspectreq struct {
	request
	args struct {
		id string
	}
	respchan chan *dockerclient.Container
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
		ActionFunc       ContainerActionFunc
	}
	respchan chan string
}

type stopreq struct {
	request
	args struct {
		id string
	}
}

var (
	cmds = struct {
		Inspect chan inspectreq
		List    chan listreq
		Start   chan startreq
		Stop    chan stopreq
	}{
		make(chan inspectreq),
		make(chan listreq),
		make(chan startreq),
		make(chan stopreq),
	}
	done  = make(chan struct{})
	srin  = make(chan startreq)
	srout = make(chan startreq)
)

// init starts up the kernel loop that is responsible for handling all the API calls
// in a goroutine.
func init() {
	client, err := dockerclient.NewClient(dockerep)
	if err != nil {
		panic(fmt.Sprintf("can't create Docker client: %v", err))
	}

	go kernel(client, done)
}

// kernel is responsible for executing all the Docker client commands.
func kernel(dc *dockerclient.Client, done chan struct{}) error {
	em, err := dc.MonitorEvents()
	if err != nil {
		panic(fmt.Sprintf("can't monitor Docker events: %v", err))
	}

	go startq(srin, srout)
	go scheduler(srout)

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
			srin <- sr
		case stop := <-cmds.Stop:
			err := dc.StopContainer(stop.args.id, uint(stop.timeout))
			if err != nil {
				stop.errchan <- err
				continue
			}

			close(stop.errchan)
		case <-done:
			return nil
		}
	}
}

// scheduler handles starting up containers. Container startup can take a long time so
// the scheduler runs in its own goroutine and pulls requests off of the start queue.
func scheduler(rc <-chan startreq, done chan struct{}) {
	select {
	case sr <- rc:
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

		err = dc.StartContainer(cid, hc)
		if err != nil {
			ec <- err
		}

		close(ec)

		if sr.args.ActionFunc != nil {
			sr.args.ActionFunc(cid)
		}

		rc <- cid
	case <-done:
		return
	}
}

// startq implements an inifinite buffered channel of start requests. Requests are added via the
// in channel and received on the next channel.
func startq(in <-chan startreq, next chan<- startreq) {
	defer close(next)

	pending := []startreq{}

restart:
	for {
		if len(pending) == 0 {
			v, ok := <-in
			if !ok {
				break
			}

			pending = append(pending, v)
		}

		select {
		case v, ok := <-in:
			if !ok {
				break restart
			}

			pending = append(pending, v)
		case next <- pending[0]:
			pending = pending[1:]
		}
	}

	for _, v := range pending {
		next <- v
	}
}
