// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package script

import (
	"errors"
	"fmt"
	"os/exec"
	"regexp"

	"github.com/docker/docker/pkg/parsers"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
)

//Lookup a tenant ID given a service (name, id, or path)
type TenantIDLookup func(service string) (string, error)

// Snapshot an application
type Snapshot func(serviceID string) (string, error)

// SnapshotRestore restore a given tenant ID and snapshot ID.
type SnapshotRestore func(tenantID, snapshotID string) error

// ServiceIDFromPath get a service id of a service given the tenant id the path to the services
type ServiceIDFromPath func(tenantID string, path string) (string, error)

// ServiceStart starts a service
type ServiceStart func(serviceID string) error

type execCmd func(string, ...string) error

type findImage func(string, bool) (*docker.Image, error)

type pullImage func(string) error

type tagImage func(*docker.Image, string) (*docker.Image, error)

type findTenant func(string) (string, error)

func defaultExec(name string, args ...string) error {
	cmd := exec.Command(name, args...)
	return cmd.Run()
}

func defaultTagImage(image *docker.Image, newTag string) (*docker.Image, error) {
	return image.Tag(newTag)
}

func renameImageID(dockerRegistry, tenantId string, imgID string, tag string) (*commons.ImageID, error) {
	repo, _ := parsers.ParseRepositoryTag(imgID)
	re := regexp.MustCompile("/?([^/]+)\\z")
	matches := re.FindStringSubmatch(repo)
	if matches == nil {
		return nil, errors.New("malformed imageid")
	}
	name := matches[1]
	newImageID := fmt.Sprintf("%s/%s/%s:%s", dockerRegistry, tenantId, name, tag)
	return commons.ParseImageID(newImageID)
}

func noOpExec(name string, args ...string) error {
	return nil
}

func noOpServiceStart(serviceID string) error {
	return nil
}

func noOpTagImage(image *docker.Image, newTag string) (*docker.Image, error) {
	return image, nil
}

func noOpRestore(tenantID, snapshotID string) error {
	return nil
}
func noOpSnapshot(serviceID string) (string, error) {
	return "no_op_snapshot", nil
}

func noOpPull(image string) error {
	return nil
}

func noOpFindImage(image string, pull bool) (*docker.Image, error) {
	id, err := commons.ParseImageID(image)
	if err != nil {
		return nil, err
	}
	return &docker.Image{"123456789", *id}, nil
}
