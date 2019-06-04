// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"strconv"
	"strings"
	"time"
)

func evalEmpty(r *runner, n node) error {
	plog.WithField("line", n.line).Debug("nothing to eval")
	return nil
}
func evalSnapshot(r *runner, n node) error {
	tagName := ""

	if len(n.args) == 1 {
		tagName = n.args[0]
	}

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

	mySnapshotID, err := r.snapshot(tID, "", tagName)
	if err != nil {
		return err
	}
	r.snapshotID = mySnapshotID //keep track of the latest snapshot to rollback to
	logger := plog.WithField("snapshotid", mySnapshotID)
	logger.Debug("performing snapshot")

	exitFunc := func(failed bool) {
		if failed && r.snapshotID == mySnapshotID {
			if err := r.restore(mySnapshotID, true); err != nil {
				logger.WithError(err).Error("Unable to restore snapshot")

			}
			logger.Info("Restored snapshot")
		}
	}
	r.addExitFunction(exitFunc)
	return nil
}

// override images with new one for services under a tenant or only for a particular service if it is specified
// possible usages:
// set image for all services under top level tenant
//SVC_USE <new image>
//override old images with new for all services under top level tenant
//SVC_USE <new image> <image to replace> replace image with new for all services under top level tenant
// override old images with new for a specific service
// SVC_USE <new image> <image to replace> service Zenoss.resmgr/Infrastructure/mariadb-model
func evalUSE(r *runner, n node) error {

	imageName := n.args[0]
	replaceImgs := make([]string, 0)
	var svcPath, serviceID string

	tenantID, found := r.env["TENANT_ID"]
	if !found {
		return fmt.Errorf("no service tenant id specified for %s", USE)
	}

	if len(n.args) > 1 {
		for i, arg := range n.args[1:] {
			if arg == "service" {
				if i == 0 {
					svcPath = n.args[i+2]
					break
				} else {
					svcPath = n.args[i+2]
					replaceImgs = make([]string, len(n.args[1:i+1]))
					replaceImgs = n.args[1 : i+1]
					break
				}
			} else {
				replaceImgs = make([]string, len(n.args[1:]))
				replaceImgs = n.args[1:len(n.args)]
			}
		}
	}
	logger := plog.WithField("imagename", imageName)
	logger.Debug("Preparing to use image")

	if svcPath != "" {
		var err error
		serviceID, err = r.svcFromPath(tenantID, svcPath)
		if err != nil {
			return err
		}

	}

	_, err := r.svcUse(tenantID, serviceID, imageName, r.config.DockerRegistry, replaceImgs, r.config.NoOp)
	if err != nil {
		return err
	}
	logger.Info("Successfully pulled and tagged new image")
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
	var stateIdx int
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
	recursive := false

	if len(n.args) > stateIdx+1 {
		// check optional first arg
		if n.args[stateIdx+1] == "recursive" {
			recursive = true
		} else {
			var timeout64 uint64
			var err error
			lastArg := n.args[len(n.args)-1]
			if timeout64, err = strconv.ParseUint(n.args[stateIdx+1], 10, 32); err != nil {
				return fmt.Errorf("Unable to parse timeout value %s: %s", lastArg, err)
			}
			timeout = uint32(timeout64)
		}
		// check optional second arg (data should be sanitized prior to here)
		if len(n.args) == stateIdx+3 {
			recursive = true
		}

	}

	plog.WithFields(log.Fields{
		"timeout":     timeout,
		"services":    strings.Join(n.args[:stateIdx], ", "),
		"targetstate": state,
	}).Info("Waiting for service(s) to reach target state")
	if err := r.svcWait(svcIDs, state, timeout, recursive); err != nil {
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

	plog.WithFields(log.Fields{
		"servicepath": svcPath,
		"serviceid":   svcID,
	}).Info("Starting service")
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

	plog.WithFields(log.Fields{
		"servicepath": svcPath,
		"serviceid":   svcID,
	}).Info("Stopping service")
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

	plog.WithFields(log.Fields{
		"servicepath": svcPath,
		"serviceid":   svcID,
	}).Info("Restarting service")
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

	plog.Debugf("Running: serviced service run %s", strings.Join(n.args, " "))
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
	logger := plog.WithFields(log.Fields{
		"serviceid":     svcID,
		"containername": containerName,
	})
	logger.Debugf("Running: serviced service shell %s", strings.Join(n.args[1:], " "))
	args := []string{"service", "shell", "-s", containerName}
	args = append(args, n.args[1:]...)
	if err := r.execCommand("serviced", args...); err != nil {
		return err
	}

	// Now commit the container (if 'COMMIT' was specified)
	switch n.args[0] {
	case "COMMIT":
		logger.Debug("committing container")
		var snapshotID string
		if snapshotID, err = r.commitContainer(containerName); err != nil {
			return err
		}
		exitFunc := func(failed bool) {
			if !failed {
				if err := r.execCommand("serviced", "snapshot", "remove", snapshotID); err != nil {
					logger.WithError(err).
						WithField("snapshotid", snapshotID).
						Warning("Unable to delete snapshot")
				}
			}
		}
		r.addExitFunction(exitFunc)
	}
	return nil
}

func evalDependency(r *runner, n node) error {
	plog.Debug("Dependency check for serviced not implemented, skipping...")
	return nil
}

func evalRequireSvc(r *runner, n node) error {
	if r.tenantIDLookup == nil {
		return fmt.Errorf("no tenant lookup function provided for %s", REQUIRE_SVC)
	}
	if r.config.ServiceID == "" {
		return errors.New("no service id specified")
	}
	//lookup tenant id for service
	tID, err := r.tenantIDLookup(r.config.ServiceID)
	if err != nil {
		return err
	}
	plog.WithFields(log.Fields{
		"tenantid":  tID,
		"serviceid": r.config.ServiceID,
	}).Debug("found tenant for service")
	r.env["TENANT_ID"] = tID
	return nil
}
