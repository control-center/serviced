package docker

import (
	"fmt"
	"io"
	"os"
	"regexp"
	"time"

	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced/commons"
)

const (
	dockerep         = "unix:///var/run/docker.sock"
	snr              = "SERVICED_NOREGISTRY"
	maxStartAttempts = 24
	Wildcard         = "*"
)

const (
	pullop = iota
	pushop
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

type commitreq struct {
	request
	args struct {
		containerID string
		imageID     *commons.ImageID
	}
	respchan chan *Image
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

type exportreq struct {
	request
	args struct {
		id      string
		outfile io.Writer
	}
}

type impimgreq struct {
	request
	args struct {
		repotag  string
		filename string
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

type pushpullreq struct {
	request
	args struct {
		op       int
		uuid     string
		reponame string
		registry string
		tag      string
	}
	respchan chan *Image
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
		Commit          chan commitreq
		Create          chan createreq
		Delete          chan deletereq
		DeleteImage     chan delimgreq
		Export          chan exportreq
		ImageImport     chan impimgreq
		ImageList       chan imglistreq
		Inspect         chan inspectreq
		Kill            chan killreq
		List            chan listreq
		OnContainerStop chan onstopreq
		OnEvent         chan oneventreq
		PullImage       chan pushpullreq
		Restart         chan restartreq
		Start           chan startreq
		Stop            chan stopreq
		TagImage        chan tagimgreq
		Wait            chan waitreq
	}{
		make(chan addactionreq),
		make(chan cancelactionreq),
		make(chan commitreq),
		make(chan createreq),
		make(chan deletereq),
		make(chan delimgreq),
		make(chan exportreq),
		make(chan impimgreq),
		make(chan imglistreq),
		make(chan inspectreq),
		make(chan killreq),
		make(chan listreq),
		make(chan onstopreq),
		make(chan oneventreq),
		make(chan pushpullreq),
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

	ppi := make(chan pushpullreq)
	ppo := make(chan pushpullreq)
	go pushpullq(ppi, ppo)

	go scheduler(dc, so, co, ppo, done)

KernelLoop:
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
		case req := <-cmds.Commit:
			// TODO: this may need to be shifted to the scheduler if commits take too long
			opts := dockerclient.CommitContainerOptions{
				Container:  req.args.containerID,
				Repository: req.args.imageID.BaseName(),
			}

			glog.V(1).Infof("commit container %s (%#v)", req.args.containerID, opts)

			img, err := dc.CommitContainer(opts)
			if err != nil {
				glog.V(1).Infof("unable to commit container %s: %v", req.args.containerID, err)
				req.errchan <- err
				continue
			}

			close(req.errchan)
			req.respchan <- &Image{img.ID, *req.args.imageID}
		case req := <-cmds.Create:
			ci <- req
		case req := <-cmds.Delete:
			glog.V(1).Infof("removing container %#v", req.args.removeOptions)
			err := dc.RemoveContainer(req.args.removeOptions)
			if err != nil {
				glog.V(1).Infof("unable to remove %#v: %v", req.args.removeOptions, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.DeleteImage:
			glog.V(1).Info("removing image: ", req.args.repotag)
			err := dc.RemoveImage(req.args.repotag)
			if err != nil {
				glog.V(1).Infof("unable to remove %s: %v", req.args.repotag, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.Export:
			// TODO: this may need to be shifted to the scheduler, exporting takes some time
			glog.V(1).Info("exporting container: ", req.args.id)
			err := dc.ExportContainer(dockerclient.ExportContainerOptions{req.args.id, req.args.outfile})
			if err != nil {
				glog.V(1).Infof("unable to export container %s: %v", req.args.id, err)
				req.errchan <- err
				continue
			}

			close(req.errchan)
		case req := <-cmds.ImageImport:
			// TODO: this may need to be shifted to the scheduler, importing takes some time
			glog.V(1).Infof("importing image %s from %s", req.args.repotag, req.args.filename)
			f, err := os.Open(req.args.filename)
			if err != nil {
				req.errchan <- err
				continue
			}
			defer f.Close()

			iid, err := commons.ParseImageID(req.args.repotag)
			if err != nil {
				req.errchan <- err
				continue
			}

			opts := dockerclient.ImportImageOptions{
				Repository:  iid.BaseName(),
				Source:      "-",
				InputStream: f,
				Tag:         iid.Tag,
			}

			if err = dc.ImportImage(opts); err != nil {
				glog.V(1).Infof("unable to import %s: %v", req.args.repotag, err)
				req.errchan <- err
				continue
			}

			close(req.errchan)
		case req := <-cmds.ImageList:
			glog.V(1).Info("retrieving image list")
			imgs, err := dc.ListImages(false)
			if err != nil {
				glog.V(1).Infof("unable to retrieve image list: ", err)
				req.errchan <- err
				continue
			}

			re := regexp.MustCompile("<none>:<none>")

			resp := []*Image{}
			for _, img := range imgs {
				for _, repotag := range img.RepoTags {
					if len(re.FindString(repotag)) > 0 {
						continue
					}

					iid, err := commons.ParseImageID(repotag)
					if err != nil {
						req.errchan <- err
						continue KernelLoop
					}
					resp = append(resp, &Image{img.ID, *iid})
				}
			}

			glog.V(1).Infof("retrieved image list: %+v", resp)

			close(req.errchan)
			req.respchan <- resp
		case req := <-cmds.Inspect:
			glog.V(1).Info("inspecting container: ", req.args.id)
			ctr, err := dc.InspectContainer(req.args.id)
			if err != nil {
				glog.V(1).Infof("unable to inspect container %s: %v", req.args.id, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
			req.respchan <- ctr
		case req := <-cmds.Kill:
			glog.V(1).Info("killing container: ", req.args.id)
			err := dc.KillContainer(dockerclient.KillContainerOptions{req.args.id, dockerclient.SIGKILL})
			if err != nil {
				glog.V(1).Infof("unable to kill container %s: %v", req.args.id, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.List:
			glog.V(1).Info("retrieving list of containers")
			apictrs, err := dc.ListContainers(dockerclient.ListContainersOptions{All: true})
			if err != nil {
				glog.V(1).Infof("unable to retrieve list of containers: %v", err)
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
			glog.V(1).Infof("retreived list of containers: %+v", resp)
			close(req.errchan)
			req.respchan <- resp
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
		case req := <-cmds.PullImage:
			ppi <- req
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
		case req := <-cmds.Stop:
			glog.V(1).Info("stopping container: ", req.args.id)
			err := dc.StopContainer(req.args.id, req.args.timeout)
			if err != nil {
				glog.V(1).Infof("unable to stop container %s: %v", req.args.id, err)
				req.errchan <- err
				continue
			}
			close(req.errchan)
		case req := <-cmds.TagImage:
			glog.V(1).Infof("tagging image %s as: %s", req.args.repo, req.args.tag)
			err := dc.TagImage(req.args.name, dockerclient.TagImageOptions{Repo: req.args.repo, Tag: req.args.tag})
			if err != nil {
				glog.V(1).Infof("unable to tag image %s: %v", req.args.repo, err)
				req.errchan <- err
				continue
			}

			if !noregistry {
				ppi <- pushpullreq{
					request{req.errchan},
					struct {
						op       int
						uuid     string
						reponame string
						registry string
						tag      string
					}{pushop, req.args.uuid, req.args.repo, req.args.registry, req.args.tag},
					req.respchan,
				}
				continue
			}

			iid, err := commons.ParseImageID(fmt.Sprintf("%s:%s", req.args.repo, req.args.tag))
			if err != nil {
				req.errchan <- err
				continue
			}

			close(req.errchan)
			glog.V(1).Infof("image %s tagged: %+v", &Image{req.args.uuid, *iid})
			req.respchan <- &Image{req.args.uuid, *iid}
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
func scheduler(dc *dockerclient.Client, src <-chan startreq, crc <-chan createreq, pprc <-chan pushpullreq, done chan struct{}) {
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
					// continue
				}

				if !noregistry {
					glog.V(2).Infof("pulling image %s prior to creating a container from it", iid.String())
					err = dc.PullImage(
						dockerclient.PullImageOptions{
							Repository: iid.BaseName(),
							Registry:   iid.Registry(),
							Tag:        iid.Tag,
						},
						dockerclient.AuthConfiguration{})
					if err != nil {
						glog.V(2).Infof("unable to pull image %s: %v", iid.String(), err)
						req.errchan <- err
						return
						// continue
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
						// continue
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

					glog.V(2).Infof("post creation start of %s: %+v", ctr.ID, req.args.hostConfig)
					err = dc.StartContainer(ctr.ID, req.args.hostConfig)
					if err != nil {
						glog.V(1).Infof("post creation start of %s failed: %v", ctr.ID, err)
						req.errchan <- err
						return
						// continue
					}

					glog.V(2).Infof("======= wait for %s to start =======", ctr.ID)
					attempts := 0

				WaitForContainerStart:
					for {
						select {
						case <-sc:
							glog.V(2).Infof("update container %s state post start", ctr.ID)
							ctr, err = dc.InspectContainer(ctr.ID)
							if err != nil {
								glog.V(1).Infof("failed to update container %s state post start: %v", ctr.ID, err)
								req.errchan <- err
								return
								// continue
							}

							glog.V(2).Infof("container %s is started", ctr.ID)
							break WaitForContainerStart
						case <-time.After(5 * time.Second):
							ctr, err = dc.InspectContainer(ctr.ID)
							if err != nil {
								glog.V(2).Infof("can't inspect container %s: %v", ctr.ID, err)
								req.errchan <- err
								return
							}

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
					// continue
				}

				glog.V(2).Infof("update container %s state post start", req.args.id)
				ctr, err := dc.InspectContainer(req.args.id)
				if err != nil {
					glog.V(2).Infof("failed to update container %s state post start: %v", req.args.id, err)
					req.errchan <- err
					return
					// continue
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
		case req := <-pprc:
			switch req.args.op {
			case pullop:
				dc, err := dockerclient.NewClient(dockerep)
				if err != nil {
					panic(fmt.Errorf("can't get docker client: %v", err))
				}

				go func(req pushpullreq, dc *dockerclient.Client) {
					glog.V(2).Info("pulling image: ", req.args.reponame)
					opts := dockerclient.PullImageOptions{
						Repository: req.args.reponame,
						Registry:   req.args.registry,
						Tag:        req.args.tag,
					}

					err := dc.PullImage(opts, dockerclient.AuthConfiguration{})
					if err != nil {
						glog.V(2).Infof("failed to pull %s: %v", req.args.reponame, err)
						req.errchan <- err
						return
						// continue
					}

					close(req.errchan)
				}(req, dc)
			case pushop:
				dc, err := dockerclient.NewClient(dockerep)
				if err != nil {
					panic(fmt.Errorf("can't get docker client: %v", err))
				}

				go func(req pushpullreq, dc *dockerclient.Client) {
					glog.V(2).Info("pushing image: ", req.args.reponame)
					opts := dockerclient.PushImageOptions{
						Name:     req.args.reponame,
						Registry: req.args.registry,
						Tag:      req.args.tag,
					}

					err = dc.PushImage(opts, dockerclient.AuthConfiguration{})
					if err != nil {
						glog.V(2).Infof("failed to push %s: %v", req.args.reponame, err)
						req.errchan <- err
						return
						// continue
					}

					iid, err := commons.ParseImageID(fmt.Sprintf("%s:%s", req.args.reponame, req.args.tag))
					if err != nil {
						req.errchan <- err
						return
						// continue
					}

					close(req.errchan)
					glog.V(2).Infof("pushed %+v", &Image{req.args.uuid, *iid})
					req.respchan <- &Image{req.args.uuid, *iid}
				}(req, dc)
			}
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

// pushpullq implements an inifinite buffered channel of pushpull requests. Requests are added via the
// in channel and received on the next channel.
func pushpullq(in <-chan pushpullreq, next chan<- pushpullreq) {
	defer close(next)

	pending := []pushpullreq{}

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
				pending = []pushpullreq{}
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
