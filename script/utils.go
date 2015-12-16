// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/domain/service"
)

//Lookup a tenant ID given a service (name, id, or path)
type TenantIDLookup func(service string) (string, error)

// Snapshot an application
type Snapshot func(serviceID string, description string, tag string) (string, error)

// Commit a container
type ContainerCommit func(containerID string) (string, error)

// SnapshotRestore restore a given a snapshot ID.
type SnapshotRestore func(snapshotID string, forceRestart bool) error

// ServiceIDFromPath get a service id of a service given the tenant id the path to the services
type ServiceIDFromPath func(tenantID string, path string) (string, error)

// ServiceControl is a func used to control the state of a service
type ServiceControl func(serviceID string, recursive bool) error

// ServiceUse is a func used to control the state of a service
type ServiceUse func(serviceID string, imageID string, registry string, replaceImgs []string, noOp bool) (string, error)

type ServiceState string

// Wait for a service to be in a particular state
type ServiceWait func(serviceID []string, serviceState ServiceState, timeout uint32, recursive bool) error

type execCmd func(string, ...string) error

type findTenant func(string) (string, error)

func ScriptStateToDesiredState(state ServiceState) (service.DesiredState, error) {
	switch state {
	case "stopped":
		return service.SVCStop, nil
	case "started":
		return service.SVCRun, nil
	case "paused":
		return service.SVCPause, nil
	}
	return service.DesiredState(-99), fmt.Errorf("service state %s unknown", state)
}

func defaultExec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func defaultTagImage(image *docker.Image, newTag string) (*docker.Image, error) {
	return image.Tag(newTag, true)
}

func noOpExec(name string, args ...string) error {
	return nil
}

func noOpServiceStart(serviceID string, recursive bool) error {
	return nil
}

func noOpServiceStop(serviceID string, recursive bool) error {
	return nil
}

func noOpServiceRestart(serviceID string, recursive bool) error {
	return nil
}

func noOpServiceUse(serviceID string, imageID string, replaceImg string, replaceImgs []string, noOp bool) (string, error) {
	return "no_op_image", nil
}

func noOpServiceWait(serviceID []string, serviceState ServiceState, timeout uint32, recursive bool) error {
	return nil
}

func noOpRestore(snapshotID string, forceRestart bool) error {
	return nil
}

func noOpSnapshot(serviceID string, description string, tag string) (string, error) {
	return "no_op_snapshot", nil
}

func noOpCommit(containerID string) (string, error) {
	return "no_op_commit", nil
}
