// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"os"
	"os/exec"

	"github.com/control-center/serviced/commons/docker"
)

//Lookup a tenant ID given a service (name, id, or path)
type TenantIDLookup func(service string) (string, error)

// Snapshot an application
type Snapshot func(serviceID string, description string) (string, error)

// Commit a container
type ContainerCommit func(containerID string) (string, error)

// SnapshotRestore restore a given a snapshot ID.
type SnapshotRestore func(snapshotID string, forceRestart bool) error

// ServiceIDFromPath get a service id of a service given the tenant id the path to the services
type ServiceIDFromPath func(tenantID string, path string) (string, error)

// ServiceControl is a func used to control the state of a service
type ServiceControl func(serviceID string, recursive bool) error

// ServiceMigrate is a func used to migrate a service
type ServiceMigrate func(serviceID string, migrationScript string, sdkVersion string) error

// ServiceUse is a func used to control the state of a service
type ServiceUse func(serviceID string, imageID string, registry string, noOp bool) (string, error)

type ServiceState string

// Wait for a service to be in a particular state
type ServiceWait func(serviceID []string, serviceState ServiceState, timeout uint32) error

type execCmd func(string, ...string) error

type findTenant func(string) (string, error)

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

func noOpServiceMigrate(serviceID string, migrationScript string, sdkVersion string) error {
	return nil
}

func noOpServiceWait(serviceID []string, serviceState ServiceState, timeout uint32) error {
	return nil
}

func noOpRestore(snapshotID string, forceRestart bool) error {
	return nil
}

func noOpSnapshot(serviceID string, description string) (string, error) {
	return "no_op_snapshot", nil
}

func noOpCommit(containerID string) (string, error) {
	return "no_op_commit", nil
}
