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
	"time"

	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

const (
	dockerep         = "unix:///var/run/docker.sock"
	maxStartAttempts = 24
	Wildcard         = "*"
)

const (
	pullop = iota
	pushop
)

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

var (
	cmds = struct {
		AddAction       chan addactionreq
		CancelAction    chan cancelactionreq
		OnContainerStop chan onstopreq
		OnEvent         chan oneventreq
	}{
		make(chan addactionreq),
		make(chan cancelactionreq),
		make(chan onstopreq),
		make(chan oneventreq),
	}
	dockerevents = []string{
		Create,
		Delete,
		Destroy,
		Die,
		Export,
		Kill,
		Restart,
		Start,
		Stop,
		Untag,
	}
	done = make(chan struct{})
)

// StartKernel starts up the kernel loop that is responsible for handling all the API calls
// in a goroutine.
func StartKernel() {
	client, err := getDockerClient()
	if err != nil {
		panic(fmt.Sprintf("can't create Docker client: %v", err))
	}
	go kernel(client, done)
}

// kernel is responsible for executing all the Docker client commands.
func kernel(dc ClientInterface, done <-chan struct{}) error {
	routeEventsToKernel(dc)

	eventactions := mkEventActionTable()

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
		case <-done:
			return nil
		}
	}
}

func mkEventActionTable() map[string]map[string]ContainerActionFunc {
	eat := make(map[string]map[string]ContainerActionFunc)

	for _, de := range dockerevents {
		eat[de] = make(map[string]ContainerActionFunc)
	}

	return eat
}

func routeEventsToKernel(dc ClientInterface) {
	em, err := dc.MonitorEvents()
	if err != nil {
		panic(fmt.Sprintf("can't monitor Docker events: %v", err))
	}

	s, err := em.Subscribe(AllThingsDocker)
	if err != nil {
		panic(fmt.Sprintf("can't subscribe to Docker events: %v", err))
	}

	for _, de := range dockerevents {
		s.Handle(de, eventToKernel)
	}
}

func eventToKernel(e *dockerclient.APIEvents) error {
	glog.V(2).Infof("sending %+v to kernel", e)
	ec := make(chan error)

	cmds.OnEvent <- oneventreq{
		request{ec},
		struct {
			id    string
			event string
		}{e.ID, e.Status},
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
