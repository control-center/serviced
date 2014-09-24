// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package docker

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	dockerep         = "unix:///var/run/docker.sock"
	sdr              = "SERVICED_REGISTRY"
	maxStartAttempts = 24
	Wildcard         = "*"
)

const (
	pullop = iota
	pushop
)

var useRegistry = false

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

type impimgreq struct {
	request
	args struct {
		repotag  string
		filename string
	}
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
	respchan chan *dockerclient.Container
}

type stopreq struct {
	request
	args struct {
		id      string
		timeout uint
	}
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
		OnContainerStop chan onstopreq
		OnEvent         chan oneventreq
		Restart         chan restartreq
		Start           chan startreq
		Wait            chan waitreq
	}{
		make(chan addactionreq),
		make(chan cancelactionreq),
		make(chan createreq),
		make(chan onstopreq),
		make(chan oneventreq),
		make(chan restartreq),
		make(chan startreq),
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
	trues := []string{"1", "true", "t", "yes"}
	if v := strings.ToLower(os.Getenv(sdr)); v != "" {
		for _, t := range trues {
			if v == t {
				useRegistry = true
			}
		}
	}

	client, err := dockerclient.NewClient(dockerep)
	if err != nil {
		panic(fmt.Sprintf("can't create Docker client: %v", err))
	}

	go kernel(client, done)
}

// SetUseRegistry sets the value of useRegistry
func SetUseRegistry(ur bool) {
	useRegistry = ur
}

// UseRegistry returns the value of useRegistry
func UseRegistry() bool {
	return useRegistry
}

// kernel is responsible for executing all the Docker client commands.
func kernel(dc *dockerclient.Client, done <-chan struct{}) error {
	routeEventsToKernel(dc)

	eventactions := mkEventActionTable()

	si := make(chan startreq)
	so := make(chan startreq)
	go startq(si, so)

	ci := make(chan createreq)
	co := make(chan createreq)
	go createq(ci, co)

	go scheduler(dc, so, co, done)

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

			glog.V(1).Info("added action for: ", event)
			close(req.errchan)
		case req := <-cmds.CancelAction:
			event := req.args.event

			if _, ok := eventactions[event]; !ok {
				req.errchan <- fmt.Errorf("docker: unknown event type: %s", event)
				continue
			}

			delete(eventactions[event], req.args.id)

			glog.V(1).Info("removed action for: ", event)
			close(req.errchan)
		case req := <-cmds.Create:
			ci <- req
		case req := <-cmds.OnEvent:
			if wcaction, ok := eventactions[req.args.event][Wildcard]; ok {
				glog.V(1).Info("executing wildcard action for event: ", req.args.event)
				go wcaction(req.args.id)
			}
			if action, ok := eventactions[req.args.event][req.args.id]; ok {
				glog.V(1).Infof("executing action for %s on %s", req.args.event, req.args.id)
				go action(req.args.id)
			}
			close(req.errchan)
		case req := <-cmds.Restart:
			// FIXME: this should really be done by the scheduler since the timeout could be long.
			glog.V(1).Info("restarting container: ", req.args.id)
			err := dc.RestartContainer(req.args.id, req.args.timeout)
			if err != nil {
				glog.V(1).Infof("unable to restart container %s: %v", req.args.id, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.Start:
			// check to see if the container is already running
			ctr, err := dc.InspectContainer(req.args.id)
			if err != nil {
				glog.V(1).Infof("unable to inspect container %s prior to starting it: %v", req.args.id, err)
				req.errchan <- err
				continue
			}

			if ctr.State.Running {
				req.errchan <- ErrAlreadyStarted
				continue
			}

			// schedule the start only if the container is not running
			si <- req
		case req := <-cmds.Wait:
			go func(req waitreq) {
				glog.V(1).Infof("waiting for container %s to finish", req.args.id)
				rc, err := dc.WaitContainer(req.args.id)
				if err != nil {
					glog.V(1).Infof("wait for container %s failed: %v", req.args.id, err)
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
func scheduler(dc *dockerclient.Client, src <-chan startreq, crc <-chan createreq, done <-chan struct{}) {
	// em, err := dc.MonitorEvents()
	// if err != nil {
	// 	panic(fmt.Sprintf("scheduler can't monitor Docker events: %v", err))
	// }

	for {
		select {
		case req := <-crc:
			dc, err := dockerclient.NewClient(dockerep)
			if err != nil {
				panic(fmt.Errorf("can't get docker client: %v", err))
			}

			go func(req createreq, dc *dockerclient.Client) {
				em, err := dc.MonitorEvents()
				if err != nil {
					panic(fmt.Sprintf("scheduler can't monitor Docker events: %v", err))
				}

				iid, err := commons.ParseImageID(req.args.containerOptions.Config.Image)
				if err != nil {
					req.errchan <- err
					return
				}

				if useRegistry {
					err = pullImage(iid.BaseName(), iid.Registry(), iid.Tag)
					if err != nil {
						glog.V(2).Infof("unable to pull image %s: %v", iid.String(), err)
						req.errchan <- err
						return
					}
				}

				glog.V(2).Infof("creating container: %#v", *req.args.containerOptions)
				ctr, err := dc.CreateContainer(*req.args.containerOptions)
				switch {
				case err == dockerclient.ErrNoSuchImage:
					pullerr := dc.PullImage(
						dockerclient.PullImageOptions{
							Repository: iid.BaseName(),
							Registry:   iid.Registry(),
							Tag:        iid.Tag,
						},
						dockerclient.AuthConfiguration{})
					if pullerr != nil {
						glog.V(2).Infof("unable to pull image %s: %v", iid.String(), err)
						req.errchan <- err
						return
					}
					ctr, err = dc.CreateContainer(*req.args.containerOptions)
					if err != nil {
						glog.V(2).Infof("container creation failed %+v: %v", *req.args.containerOptions, err)
						req.errchan <- err
						return
					}
				case err != nil:
					glog.V(2).Infof("container creation failed %+v: %v", *req.args.containerOptions, err)
					req.errchan <- err
					return
				}

				glog.V(2).Infof("created container: %+v", *ctr)

				if req.args.createaction != nil {
					req.args.createaction(ctr.ID)
				}

				if req.args.start {
					ss, err := em.Subscribe(ctr.ID)
					if err != nil {
						req.errchan <- err
						return
					}

					sc := make(chan struct{})

					ss.Handle(Start, func(e dockerclient.Event) error {
						if req.args.startaction != nil {
							req.args.startaction(ctr.ID)
						}
						glog.V(2).Infof("handling event: %+v for %s", e, ctr.ID)
						sc <- struct{}{}
						return nil
					})
					defer ss.Cancel()

					glog.V(2).Infof("post creation start of %s: %+v", ctr.ID, req.args.hostConfig)
					err = dc.StartContainer(ctr.ID, req.args.hostConfig)
					if err != nil {
						glog.V(1).Infof("post creation start of %s failed: %v", ctr.ID, err)
						req.errchan <- err
						return
					}

					glog.V(2).Infof("======= wait for %s to start =======", ctr.ID)
					attempts := 0

				WaitForContainerStart:
					for {
						select {
						case <-sc:
							glog.V(2).Infof("update container %s state post start", ctr.ID)
							ctrID := ctr.ID
							ctr, err = dc.InspectContainer(ctrID)
							if err != nil {
								glog.V(1).Infof("failed to update container %s state post start: %v", ctrID, err)
								req.errchan <- err
								return
							}

							glog.V(2).Infof("container %s is started", ctr.ID)
							break WaitForContainerStart
						case <-time.After(5 * time.Second):
							nctr, err := dc.InspectContainer(ctr.ID)
							if err != nil {
								glog.V(2).Infof("can't inspect container %s: %v", ctr.ID, err)
								req.errchan <- err
								return
							}
							ctr = nctr

							switch {
							case !ctr.State.Running && attempts > maxStartAttempts:
								glog.V(2).Infof("timed out starting container")
								req.errchan <- fmt.Errorf("timed out starting container: %s", ctr.ID)
								return
							case !ctr.State.Running:
								attempts = attempts + 1
								continue WaitForContainerStart
							default:
								glog.V(2).Infof("container %s is running", ctr.ID)
								break WaitForContainerStart
							}
						}
					}
				}

				glog.V(2).Infof("success. closing errchan")
				close(req.errchan)

				// don't hang around forever waiting for the caller to get the result
				select {
				case req.respchan <- ctr:
					glog.V(2).Infof("sent %s to caller", ctr)
					break
				case <-time.After(100 * time.Millisecond):
					glog.V(2).Infof("timed out waiting for call to get result")
					break
				}
			}(req, dc)
		case req := <-src:
			dc, err := dockerclient.NewClient(dockerep)
			if err != nil {
				panic(fmt.Errorf("can't get docker client: %v", err))
			}

			go func(req startreq, dc *dockerclient.Client) {
				glog.V(2).Infof("starting container %s: %+v", req.args.id, req.args.hostConfig)
				err := dc.StartContainer(req.args.id, req.args.hostConfig)
				if err != nil {
					glog.V(2).Infof("unable to start %s: %v", req.args.id, err)
					req.errchan <- err
					return
				}

				glog.V(2).Infof("update container %s state post start", req.args.id)
				ctr, err := dc.InspectContainer(req.args.id)
				if err != nil {
					glog.V(2).Infof("failed to update container %s state post start: %v", req.args.id, err)
					req.errchan <- err
					return
				}

				if req.args.action != nil {
					req.args.action(req.args.id)
				}

				close(req.errchan)

				// don't hang around forever waiting for the caller to get the result
				select {
				case req.respchan <- ctr:
					break
				case <-time.After(100 * time.Millisecond):
					break
				}
			}(req, dc)
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

			glog.V(2).Infof("received first start request: %+v", v)
			pending = append(pending, v)
		}

		select {
		case v, ok := <-in:
			if !ok {
				break restart
			}

			glog.V(2).Infof("received start request %+v", v)
			pending = append(pending, v)

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []startreq{}
			}
		case next <- pending[0]:
			glog.V(2).Infof("delivered start request %+v", pending[0])
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

			glog.V(2).Infof("received first create request: %+v", v)
			pending = append(pending, v)
		}

		select {
		case v, ok := <-in:
			if !ok {
				break restart
			}

			glog.V(2).Infof("received create request: %+v", v)
			pending = append(pending, v)

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []createreq{}
			}
		case next <- pending[0]:
			glog.V(2).Infof("delivered create request: %+v", pending[0])
			pending = pending[1:]
		}
	}

	for _, v := range pending {
		next <- v
	}
}

// stopq implements an inifinite buffered channel of create requests. Requests are added via the
// in channel and received on the next channel.
func stopq(in <-chan stopreq, next chan<- stopreq) {
	defer close(next)

	pending := []stopreq{}

restart:
	for {
		if len(pending) == 0 {
			v, ok := <-in
			if !ok {
				break
			}

			glog.V(2).Infof("received first create request: %+v", v)
			pending = append(pending, v)
		}

		select {
		case v, ok := <-in:
			if !ok {
				break restart
			}

			glog.V(2).Infof("received create request: %+v", v)
			pending = append(pending, v)

			// don't let a burst of requests starve the outgoing channel
			if len(pending) > 8 {
				for _, v := range pending {
					next <- v
				}
				pending = []stopreq{}
			}
		case next <- pending[0]:
			glog.V(2).Infof("delivered create request: %+v", pending[0])
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
	glog.V(2).Infof("sending %+v to kernel", e)
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
