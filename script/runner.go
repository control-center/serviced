// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/zenoss/glog"
)

var (
	cmdEval map[string]func(*runner, node) error
)

func init() {
	cmdEval = map[string]func(*runner, node) error{
		"":          evalEmpty,
		DESCRIPTION: evalEmpty,
		VERSION:     evalEmpty,
		SNAPSHOT:    evalSnapshot,
		USE:         evalUSE,
		SVC_RUN:     evalSvcRun,
		DEPENDENCY:  evalDependency,
		REQUIRE_SVC: evalRequireSvc,
	}
}

type Config struct {
	ServiceID      string
	DockerRegistry string            //docker registry being used for tagging images
	NoOp           bool              //Should commands modify the system
	TenantLookup   TenantIDLookup    //function for looking up a service
	Snapshot       Snapshot          //function for creating snapshots
	Restore        SnapshotRestore   //function to do the rollback to a snapshot
	SvcIDFromPath  ServiceIDFromPath // function to find a service id from a path
}

type Runner interface {
	Run() error
}

func NewRunnerFromFile(fileName string, config *Config) (Runner, error) {
	f, err := os.Open(fileName)
	if err != nil {
		return nil, err
	}
	r := bufio.NewReader(f)
	return NewRunner(r, config)
}

func NewRunner(r io.Reader, config *Config) (Runner, error) {
	pctx, err := parseDescriptor(r)
	if err != nil {
		return nil, err
	}
	if len(pctx.errors) > 0 {
		//TODO: print each error
		return nil, errors.New("error parsing serviced runner file")
	}
	return newRunner(config, pctx), nil
}

func newRunner(config *Config, pctx *parseContext) *runner {
	if config.DockerRegistry == "" {
		config.DockerRegistry = "localhost:5000"
	}
	r := &runner{
		parseCtx:       pctx,
		config:         config,
		exitFunctions:  make([]func(bool), 0),
		env:            make(map[string]string),
		tenantIDLookup: config.TenantLookup,
		snapshot:       config.Snapshot,
		restore:        config.Restore,
		svcFromPath:    config.SvcIDFromPath,
		findImage:      docker.FindImage,
		pullImage:      docker.PullImage,
		execCommand:    defaultExec,
		tagImage:       defaultTagImage,
	}
	if config.NoOp {
		glog.Infof("creatng no op runner")
		r.execCommand = noOpExec
		r.tagImage = noOpTagImage
		r.restore = noOpRestore
		r.snapshot = noOpSnapshot
		r.pullImage = noOpPull
		r.findImage = noOpFindImage
	}

	return r
}

type runner struct {
	parseCtx       *parseContext
	config         *Config
	exitFunctions  []func(bool)      //each is called on exit of upgrade, bool denotes if upgrade exited with an error
	snapshotID     string            //the last snapshot taken
	env            map[string]string //context variables available to runner
	tenantIDLookup TenantIDLookup    //function for looking up a service
	snapshot       Snapshot          //function for creating snapshots
	restore        SnapshotRestore   //function to do the rollback to a snapshot
	svcFromPath    ServiceIDFromPath //function to find a service from a path and tenant
	findImage      findImage
	pullImage      pullImage
	execCommand    execCmd
	tagImage       tagImage
}

func (r *runner) Run() error {
	if err := r.evalNodes(r.parseCtx.nodes); err != nil {
		return err
	}
	return nil
}

func (r *runner) evalNodes(nodes []node) error {
	failed := true
	defer func() {
		for _, ef := range r.exitFunctions {
			ef(failed)
		}
	}()

	for i, n := range nodes {
		if f, found := cmdEval[n.cmd]; found {
			glog.Infof("executing step %d: %s", i, n.line)
			if err := f(r, n); err != nil {
				glog.Errorf("error executing step %d: %s: %s", i, n.cmd, err)
				return err
			}
		} else {
			glog.Infof("skipping step %d unknown function: %s", i, n.line)
		}
	}
	failed = false
	return nil
}
func (r *runner) addExitFunction(ef func(bool)) {
	r.exitFunctions = append(r.exitFunctions, ef)
}

func evalEmpty(r *runner, n node) error {
	glog.V(1).Infof("nothing to eval: %s", n.line)
	return nil
}
func evalSnapshot(r *runner, n node) error {
	glog.V(0).Info("performing snapshot")

	if r.snapshot == nil {
		return fmt.Errorf("no snapshot function provided for %s", SNAPSHOT)
	}
	if r.restore == nil {
		return fmt.Errorf("no restore function provided for %s", SNAPSHOT)
	}

	svcID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SNAPSHOT)
	}

	mySnapshotID, err := r.snapshot(svcID)
	if err != nil {
		return err
	}
	r.snapshotID = mySnapshotID //keep track of the latest snapshot to rollback to
	glog.V(0).Infof("snapshot id: %s", mySnapshotID)

	exitFunc := func(failed bool) {
		if failed && r.snapshotID == mySnapshotID {
			glog.Infof("restoring snapshot %s", mySnapshotID)
			if err := r.restore(mySnapshotID); err != nil {
				glog.Errorf("failed restoring snapshot %s: %v", mySnapshotID, err)

			}
		}
	}
	r.addExitFunction(exitFunc)
	return nil
}

func evalUSE(r *runner, n node) error {
	imageName := n.args[0]
	glog.V(0).Infof("preparing to use image: %s", imageName)
	svcID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", USE)
	}

	imageID, err := commons.ParseImageID(imageName)
	if err != nil {
		return err
	}
	if imageID.Tag == "" {
		imageID.Tag = "latest"
	}
	glog.Infof("pulling image %s, this may take a while...", imageID)
	if err := r.pullImage(imageID.String()); err != nil {
		return fmt.Errorf("unable to pull image %s", imageID)
	}

	//verify image has been pulled
	img, err := r.findImage(imageID.String(), false)
	if err != nil {
		err = fmt.Errorf("could not look up image %s: %s. Check your docker login and retry service deployment.", imageID, err)
		return err
	}

	//Tag images to latest all images
	var newTag *commons.ImageID

	newTag, err = renameImageID(r.config.DockerRegistry, svcID, imageID.String(), "latest")
	if err != nil {
		return err
	}
	glog.Infof("tagging image %s to %s ", imageName, newTag)
	if _, err = r.tagImage(img, newTag.String()); err != nil {
		glog.Errorf("could not tag image: %s (%v)", imageName, err)
		return err
	}
	return nil
}

func evalSvcRun(r *runner, n node) error {
	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_RUN)
	}

	svcPath := n.args[0]
	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SVC_RUN)
	}
	svcID, err := r.svcFromPath(tenantID, svcPath)
	if err != nil {
		return err
	}
	if svcID == "" {
		return fmt.Errorf("no service id found for %s", svcPath)
	}

	n.args[0] = svcID

	glog.V(0).Infof("running: serviced service run %s", strings.Join(n.args, " "))
	args := []string{"service", "run"}
	args = append(args, n.args...)
	if err := r.execCommand("serviced", args...); err != nil {
		return err
	}

	return nil
}
func evalDependency(r *runner, n node) error {
	glog.V(0).Infof("checking serviced dependency: %s", n.args[0])
	glog.V(0).Info("dependency check for serviced not implemented, skipping...")
	return nil
}

func evalRequireSvc(r *runner, n node) error {
	if r.tenantIDLookup == nil {
		return fmt.Errorf("no tenant lookup function provided for %s", REQUIRE_SVC)
	}
	glog.V(0).Infof("checking service requirement")
	if r.config.ServiceID == "" {
		return errors.New("no service id specified")
	}
	glog.V(0).Infof("verifying service %s", r.config.ServiceID)
	//lookup tenant id for service
	tID, err := r.tenantIDLookup(r.config.ServiceID)
	if err != nil {
		return err
	}
	glog.V(0).Infof("found %s tenant id for service %s", tID, r.config.ServiceID)
	r.env["TENANT_ID"] = tID
	return nil
}
