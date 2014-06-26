package docker

import (
	"fmt"
	"os"
	"time"

	dockerclient "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced/commons"
)

const (
	dockerep = "unix:///var/run/docker.sock"
	snr      = "SERVICED_NOREGISTRY"
	Wildcard = "*"
)

var noregistry bool

type request struct {
	errchan chan error
}

type addactionreq struct {
	request
	args struct {
		id     string
		event  string
		action ContainerActionFunc
	}
}

type cancelactionreq struct {
	request
	args struct {
		id    string
		event string
	}
}

type createreq struct {
	request
	args struct {
		containerOptions *dockerclient.CreateContainerOptions
		hostConfig       *dockerclient.HostConfig
		start            bool
		createaction     ContainerActionFunc
		startaction      ContainerActionFunc
	}
	respchan chan *dockerclient.Container
}

type deletereq struct {
	request
	args struct {
		removeOptions dockerclient.RemoveContainerOptions
	}
}

type delimgreq struct {
	request
	args struct {
		repotag string
	}
}

type imglistreq struct {
	request
	respchan chan []*Image
}

type inspectreq struct {
	request
	args struct {
		id string
	}
	respchan chan *dockerclient.Container
}

type killreq struct {
	request
	args struct {
		id string
	}
}

type listreq struct {
	request
	respchan chan []*Container
}

type oneventreq struct {
	request
	args struct {
		id    string
		event string
	}
}

type onstopreq struct {
	request
	args struct {
		id     string
		action ContainerActionFunc
	}
}

type pullimgreq struct {
	request
	args struct {
		repo     string
		registry string
		tag      string
	}
}

type restartreq struct {
	request
	args struct {
		id      string
		timeout uint
	}
}

type startreq struct {
	request
	args struct {
		id         string
		hostConfig *dockerclient.HostConfig
		action     ContainerActionFunc
	}
}

type stopreq struct {
	request
	args struct {
		id      string
		timeout uint
	}
}

type tagimgreq struct {
	request
	args struct {
		uuid     string
		name     string
		repo     string
		registry string
		tag      string
	}
	respchan chan *Image
}

type waitreq struct {
	request
	args struct {
		id string
	}
	respchan chan int
}

var (
	cmds = struct {
		AddAction       chan addactionreq
		CancelAction    chan cancelactionreq
		Create          chan createreq
		Delete          chan deletereq
		DeleteImage     chan delimgreq
		ImageList       chan imglistreq
		Inspect         chan inspectreq
		Kill            chan killreq
		List            chan listreq
		OnContainerStop chan onstopreq
		OnEvent         chan oneventreq
		PullImage       chan pullimgreq
		Restart         chan restartreq
		Start           chan startreq
		Stop            chan stopreq
		TagImage        chan tagimgreq
		Wait            chan waitreq
	}{
		make(chan addactionreq),
		make(chan cancelactionreq),
		make(chan createreq),
		make(chan deletereq),
		make(chan delimgreq),
		make(chan imglistreq),
		make(chan inspectreq),
		make(chan killreq),
		make(chan listreq),
		make(chan onstopreq),
		make(chan oneventreq),
		make(chan pullimgreq),
		make(chan restartreq),
		make(chan startreq),
		make(chan stopreq),
		make(chan tagimgreq),
		make(chan waitreq),
	}
	dockerevents = []string{
		dockerclient.Create,
		dockerclient.Delete,
		dockerclient.Destroy,
		dockerclient.Die,
		dockerclient.Export,
		dockerclient.Kill,
		dockerclient.Restart,
		dockerclient.Start,
		dockerclient.Stop,
		dockerclient.Untag,
	}
	done = make(chan struct{})
)

// init starts up the kernel loop that is responsible for handling all the API calls
// in a goroutine.
func init() {
	if os.Getenv(snr) != "" {
		noregistry = true
	}

	client, err := dockerclient.NewClient(dockerep)
	if err != nil {
		panic(fmt.Sprintf("can't create Docker client: %v", err))
	}

	go kernel(client, done)
}

// kernel is responsible for executing all the Docker client commands.
func kernel(dc *dockerclient.Client, done chan struct{}) error {
	routeEventsToKernel(dc)

	eventactions := mkEventActionTable()

	si := make(chan startreq)
	so := make(chan startreq)
	go startq(si, so)

	ci := make(chan createreq)
	co := make(chan createreq)
	go createq(ci, co)

	pi := make(chan pullimgreq)
	po := make(chan pullimgreq)
	go pullq(pi, po)

	go scheduler(dc, so, co, po, done)

	for {
		select {
		case req := <-cmds.AddAction:
			event := req.args.event

			if _, ok := eventactions[event]; !ok {
				req.errchan <- fmt.Errorf("docker: unknown event type: %s", event)
				continue
			}

			if _, ok := eventactions[event]; !ok {
				eventactions[event] = make(map[string]ContainerActionFunc)
			}

			eventactions[event][req.args.id] = req.args.action
			close(req.errchan)
		case req := <-cmds.CancelAction:
			event := req.args.event

			if _, ok := eventactions[event]; !ok {
				req.errchan <- fmt.Errorf("docker: unknown event type: %s", event)
				continue
			}

			delete(eventactions[event], req.args.id)
			close(req.errchan)
		case req := <-cmds.Create:
			ci <- req
		case req := <-cmds.Delete:
			err := dc.RemoveContainer(req.args.removeOptions)
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.DeleteImage:
			err := dc.RemoveImage(req.args.repotag)
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.ImageList:
			imgs, err := dc.ListImages(false)
			if err != nil {
				req.errchan <- err
				continue
			}

			resp := []*Image{}
			for _, img := range imgs {
				for _, repotag := range img.RepoTags {
					iid, err := commons.ParseImageID(repotag)
					if err != nil {
						req.errchan <- err
						continue
					}
					resp = append(resp, &Image{img.ID, *iid})
				}
			}

			close(req.errchan)
			req.respchan <- resp
		case req := <-cmds.Inspect:
			ctr, err := dc.InspectContainer(req.args.id)
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
			req.respchan <- ctr
		case req := <-cmds.Kill:
			err := dc.KillContainer(dockerclient.KillContainerOptions{req.args.id, dockerclient.SIGINT})
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.List:
			apictrs, err := dc.ListContainers(dockerclient.ListContainersOptions{All: true})
			if err != nil {
				req.errchan <- err
				continue
			}
			resp := []*Container{}
			for _, apictr := range apictrs {
				ctr, err := dc.InspectContainer(apictr.ID)
				if err != nil {
					continue
				}
				resp = append(resp, &Container{ctr, dockerclient.HostConfig{}})
			}
			close(req.errchan)
			req.respchan <- resp
		case req := <-cmds.OnEvent:
			if wcaction, ok := eventactions[req.args.event][Wildcard]; ok {
				go wcaction(req.args.id)
			}
			if action, ok := eventactions[req.args.event][req.args.id]; ok {
				go action(req.args.id)
			}
			close(req.errchan)
		case req := <-cmds.PullImage:
			req.errchan <- fmt.Errorf("kernel does not implement PullImage")
			continue
		case req := <-cmds.Restart:
			// FIXME: this should really be done by the scheduler since the timeout could be long.
			err := dc.RestartContainer(req.args.id, req.args.timeout)
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.Start:
			// check to see if the container is already running
			ctr, err := dc.InspectContainer(req.args.id)
			if err != nil {
				req.errchan <- err
				continue
			}

			if ctr.State.Running {
				req.errchan <- ErrAlreadyStarted
				continue
			}

			// schedule the start only if the container is not running
			si <- req
		case req := <-cmds.Stop:
			err := dc.StopContainer(req.args.id, req.args.timeout)
			if err != nil {
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.TagImage:
			err := dc.TagImage(req.args.name, dockerclient.TagImageOptions{Repo: req.args.repo, Tag: req.args.tag})
			if err != nil {
				req.errchan <- err
				continue
			}

			if !noregistry {
				opts := dockerclient.PushImageOptions{
					Name:     req.args.repo,
					Registry: req.args.registry,
					Tag:      req.args.tag,
				}

				// FIXME: pushing can be slow and should be moved to the scheduler
				err = dc.PushImage(opts, dockerclient.AuthConfiguration{})
				if err != nil {
					req.errchan <- err
					continue
				}
			}

			iid, err := commons.ParseImageID(fmt.Sprintf("%s:%s", req.args.repo, req.args.tag))
			if err != nil {
				req.errchan <- err
				continue
			}

			close(req.errchan)
			req.respchan <- &Image{req.args.uuid, *iid}
		case req := <-cmds.Wait:
			go func(req waitreq) {
				rc, err := dc.WaitContainer(req.args.id)
				if err != nil {
					req.errchan <- err
				}
				close(req.errchan)
				req.respchan <- rc
			}(req)
		case <-done:
			close(si)
			close(ci)
			return nil
		}
	}
}

// scheduler handles creating and starting up containers and pulling images. Those operations can take a long time so
// the scheduler runs in its own goroutine and pulls requests off of the create, start, and pull queues.
func scheduler(dc *dockerclient.Client, src <-chan startreq, crc <-chan createreq, prc <-chan pullimgreq, done chan struct{}) {
	em, err := dc.MonitorEvents()
	if err != nil {
		panic(fmt.Sprintf("scheduler can't monitor Docker events: %v", err))
	}

	for {
		select {
		case req := <-crc:
			// always pull the image to make sure we have the latest version
			iid, err := commons.ParseImageID(req.args.containerOptions.Config.Image)
			if err != nil {
				req.errchan <- err
				continue
			}

			err = dc.PullImage(
				dockerclient.PullImageOptions{
					Repository: iid.BaseName(),
					Registry:   iid.Registry(),
					Tag:        iid.Tag,
				},
				dockerclient.AuthConfiguration{})
			if err != nil {
				req.errchan <- err
				continue
			}

			ctr, err := dc.CreateContainer(*req.args.containerOptions)
			if err != nil {
				req.errchan <- err
			}

			if req.args.createaction != nil {
				req.args.createaction(ctr.ID)
			}

			if req.args.start {
				ss, err := em.Subscribe(ctr.ID)
				if err != nil {
					req.errchan <- err
					continue
				}

				sc := make(chan struct{})

				ss.Handle(Start, func(e dockerclient.Event) error {
					if req.args.startaction != nil {
						req.args.startaction(ctr.ID)
					}
					sc <- struct{}{}
					return nil
				})

				err = dc.StartContainer(ctr.ID, req.args.hostConfig)
				if err != nil {
					req.errchan <- err
					continue
				}

				select {
				case <-sc:
				case <-time.After(60 * time.Second):
					req.errchan <- fmt.Errorf("timed out starting container: %s", ctr.ID)
					continue
				}
			}

			close(req.errchan)

			// don't hang around forever waiting for the caller to get the result
			select {
			case req.respchan <- ctr:
				break
			case <-time.After(10 * time.Millisecond):
				break
			}
		case req := <-src:
			err := dc.StartContainer(req.args.id, req.args.hostConfig)
			if err != nil {
				req.errchan <- err
				continue
			}

			if req.args.action != nil {
				req.args.action(req.args.id)
			}

			close(req.errchan)
		case req := <-prc:
			opts := dockerclient.PullImageOptions{
				Repository: req.args.repo,
				Registry:   req.args.registry,
				Tag:        req.args.tag,
			}

			err := dc.PullImage(opts, dockerclient.AuthConfiguration{})
			if err != nil {
				req.errchan <- err
				continue
			}

			close(req.errchan)
		case <-done:
			return
		}
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

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []startreq{}
			}
		case next <- pending[0]:
			pending = pending[1:]
		}
	}

	for _, v := range pending {
		next <- v
	}
}

// createq implements an inifinite buffered channel of create requests. Requests are added via the
// in channel and received on the next channel.
func createq(in <-chan createreq, next chan<- createreq) {
	defer close(next)

	pending := []createreq{}

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

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []createreq{}
			}
		case next <- pending[0]:
			pending = pending[1:]
		}
	}

	for _, v := range pending {
		next <- v
	}
}

// pullq implements an inifinite buffered channel of pull requests. Requests are added via the
// in channel and received on the next channel.
func pullq(in <-chan pullimgreq, next chan<- pullimgreq) {
	defer close(next)

	pending := []pullimgreq{}

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

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []pullimgreq{}
			}
		case next <- pending[0]:
			pending = pending[1:]
		}
	}

	for _, v := range pending {
		next <- v
	}
}

func mkEventActionTable() map[string]map[string]ContainerActionFunc {
	eat := make(map[string]map[string]ContainerActionFunc)

	for _, de := range dockerevents {
		eat[de] = make(map[string]ContainerActionFunc)
	}

	return eat
}

func routeEventsToKernel(dc *dockerclient.Client) {
	em, err := dc.MonitorEvents()
	if err != nil {
		panic(fmt.Sprintf("can't monitor Docker events: %v", err))
	}

	s, err := em.Subscribe(dockerclient.AllThingsDocker)
	if err != nil {
		panic(fmt.Sprintf("can't subscribe to Docker events: %v", err))
	}

	for _, de := range dockerevents {
		s.Handle(de, eventToKernel)
	}
}

func eventToKernel(e dockerclient.Event) error {
	ec := make(chan error)

	cmds.OnEvent <- oneventreq{
		request{ec},
		struct {
			id    string
			event string
		}{e["id"].(string), e["status"].(string)},
	}

	select {
	case <-time.After(1 * time.Second):
		return ErrRequestTimeout
	case <-done:
		return ErrKernelShutdown
	default:
		switch err, ok := <-ec; {
		case !ok:
			return nil
		default:
			return fmt.Errorf("docker: event handler failed: %v", err)
		}
	}
}
