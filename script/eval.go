// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/zenoss/glog"
)

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

	tID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SNAPSHOT)
	}

	mySnapshotID, err := r.snapshot(tID)
	if err != nil {
		return err
	}
	r.snapshotID = mySnapshotID //keep track of the latest snapshot to rollback to
	glog.V(0).Infof("snapshot id: %s", mySnapshotID)

	exitFunc := func(failed bool) {
		if failed && r.snapshotID == mySnapshotID {
			glog.Infof("restoring snapshot %s", mySnapshotID)
			if err := r.restore(mySnapshotID, true); err != nil {
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
		glog.Warningf("unable to pull image %s", imageID)
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

func evalSvcWait(r *runner, n node) error {

	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_WAIT)
	}

	if r.svcWait == nil {
		return fmt.Errorf("no service wait function for %s", SVC_WAIT)
	}

	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SVC_WAIT)
	}

	var svcIDs []string
	var stateIdx = -1
	for i, arg := range n.args {
		if arg == "started" || arg == "stopped" || arg == "paused" {
			stateIdx = i
			break
		}
		svcPath := arg
		svcID, err := r.svcFromPath(tenantID, arg)
		if err != nil {
			return err
		}
		if svcID == "" {
			return fmt.Errorf("no service id found for %s", svcPath)
		}
		svcIDs = append(svcIDs, svcID)
	}

	state := ServiceState(n.args[stateIdx])

	timeout := uint32(0)
	hasTimeout := len(n.args) == stateIdx+2
	if hasTimeout {
		var timeout64 uint64
		var err error
		lastArg := n.args[len(n.args)-1]
		if timeout64, err = strconv.ParseUint(lastArg, 10, 32); err != nil {
			return fmt.Errorf("Unable to parse timeout value %s: %s", lastArg, err)
		}
		timeout = uint32(timeout64)
	}

	plural := ""
	if stateIdx > 1 {
		plural = "s"
	}
	glog.Infof("waiting %d for service%s %s to be %s", timeout, plural, strings.Join(n.args[:stateIdx], ", "), state)
	if err := r.svcWait(svcIDs, state, timeout); err != nil {
		return err
	}

	return nil
}

func evalSvcStart(r *runner, n node) error {

	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_START)
	}

	if r.svcStart == nil {
		return fmt.Errorf("no service start function for %s", SVC_START)
	}

	svcPath := n.args[0]
	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SVC_START)
	}
	svcID, err := r.svcFromPath(tenantID, svcPath)
	if err != nil {
		return err
	}
	if svcID == "" {
		return fmt.Errorf("no service id found for %s", svcPath)
	}

	recursive := false
	if len(n.args) > 1 {
		recursive = true
	}

	glog.Infof("starting service %s %s", svcPath, svcID)
	if err := r.svcStart(svcID, recursive); err != nil {
		return err
	}

	return nil
}

func evalSvcStop(r *runner, n node) error {

	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_STOP)
	}

	if r.svcStop == nil {
		return fmt.Errorf("no service stop function for %s", SVC_STOP)
	}

	svcPath := n.args[0]
	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SVC_STOP)
	}
	svcID, err := r.svcFromPath(tenantID, svcPath)
	if err != nil {
		return err
	}
	if svcID == "" {
		return fmt.Errorf("no service id found for %s", svcPath)
	}

	recursive := false
	if len(n.args) > 1 {
		recursive = true
	}

	glog.Infof("stopping service %s %s", svcPath, svcID)
	if err := r.svcStop(svcID, recursive); err != nil {
		return err
	}

	return nil
}

func evalSvcRestart(r *runner, n node) error {

	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_RESTART)
	}

	if r.svcRestart == nil {
		return fmt.Errorf("no service restart function for %s", SVC_RESTART)
	}

	svcPath := n.args[0]
	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", SVC_RESTART)
	}
	svcID, err := r.svcFromPath(tenantID, svcPath)
	if err != nil {
		return err
	}
	if svcID == "" {
		return fmt.Errorf("no service id found for %s", svcPath)
	}

	recursive := false
	if len(n.args) > 1 {
		recursive = true
	}

	glog.Infof("restarting service %s %s", svcPath, svcID)
	if err := r.svcRestart(svcID, recursive); err != nil {
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

func evalSvcExec(r *runner, n node) error {
	if r.svcFromPath == nil {
		return fmt.Errorf("no service id lookup function for %s", SVC_RUN)
	}

	svcPath := n.args[1]
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

	n.args[1] = svcID
	// "HMS-YMD_svcID" will be the name of the container
	containerName := time.Now().Format("150405-20060102") + "_" + svcID

	glog.V(0).Infof("running: serviced service shell -s %s %s", containerName, strings.Join(n.args[1:], " "))
	args := []string{"service", "shell", "-s", containerName}
	args = append(args, n.args[1:]...)
	if err := r.execCommand("serviced", args...); err != nil {
		return err
	}

	// Now commit the container (if 'COMMIT' was specified)
	switch n.args[0] {
	case "COMMIT":
		glog.V(0).Infof("committing container %s", containerName)
		var snapshotID string
		if snapshotID, err = r.commitContainer(containerName); err != nil {
			return err
		}
		exitFunc := func(failed bool) {
			if !failed {
				glog.V(1).Infof("cleaning up snapshot %s", snapshotID)
				if err := r.execCommand("serviced", "snapshot", "remove", snapshotID); err != nil {
					glog.Errorf("failed deleting snapshot %s: %v", snapshotID, err)
				}
			}
		}
		r.addExitFunction(exitFunc)
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
