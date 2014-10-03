// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package dfs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/layer"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	DockerLatest = "latest"
)

// Commit will merge a container into existing services' image
func (dfs *DistributedFilesystem) Commit(dockerID string) (string, error) {
	// get the container and verify that it is not running
	ctr, err := docker.FindContainer(dockerID)
	if err != nil {
		glog.Errorf("Could not get container %s: %s", dockerID, err)
		return "", err
	}

	if ctr.IsRunning() {
		err := fmt.Errorf("cannot commit a running container")
		glog.Errorf("Error committing container %s: %s", ctr.ID, err)
		return "", err
	}

	// parse the image information
	imageID, err := commons.ParseImageID(ctr.Config.Image)
	if err != nil {
		glog.Errorf("Could not parse image information for %s: %s", dockerID, err)
		return "", err
	}
	tenantID := imageID.User

	// find the image that is being committed
	image, err := findImage(tenantID, ctr.Image, DockerLatest)
	if err != nil {
		glog.Errorf("Could not find image %s: %s", dockerID, err)
		return "", fmt.Errorf("cannot commit a stale container")
	}

	// check the number of image layers
	if layers, err := image.History(); err != nil {
		glog.Errorf("Could not check history for image %s: %s", image.ID, err)
		return "", err
	} else if numLayers := len(layers); numLayers >= layer.WARN_LAYER_COUNT {
		glog.Warningf("Image %s has %d layers and is approaching the maximum (%d). Please squash image layers.",
			image.ID, numLayers, layer.MAX_LAYER_COUNT)
	} else {
		glog.V(3).Infof("Image %s has %d layers", image.ID, numLayers)
	}

	// commit the container to the image and tag
	if _, err := ctr.Commit(image.ID.BaseName()); err != nil {
		glog.Errorf("Error trying to commit %s to %s: %s", dockerID, image.ID, err)
		return "", err
	}

	// desynchronize any running containers
	if err := dfs.desynchronize(image.ID, time.Now()); err != nil {
		glog.Warningf("Could not denote all desynchronized services: %s", err)
	}

	// snapshot the filesystem and images
	snapshotID, err := dfs.Snapshot(tenantID)
	if err != nil {
		glog.Errorf("Could not create a snapshot of the new image %s: %s", tenantID, err)
		return "", err
	}

	return snapshotID, nil
}

func (dfs *DistributedFilesystem) desynchronize(imageID commons.ImageID, commit time.Time) error {
	svcs, err := dfs.facade.GetServices(datastore.Get(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not get all services", err)
		return err
	}

	for _, svc := range svcs {
		conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
		if err != nil {
			glog.Errorf("Could not acquire connection to coordinator (%s): %s", svc.PoolID, err)
			return err
		}

		// figure out which services use the provided image
		img, err := commons.ParseImageID(svc.ImageID)
		if err != nil {
			glog.Errorf("Error while parsing image %s for %s (%s): %s", svc.ImageID, svc.Name, svc.ID)
			return err
		}

		if !img.Equals(imageID) {
			continue
		}

		states, err := zkservice.GetServiceStates(conn, svc.ID)
		if err != nil {
			glog.Errorf("Could not get running services for %s (%s): %s", svc.Name, svc.ID)
			return err
		}

		for _, state := range states {
			// check if the instance has been running since before the commit
			if state.IsRunning() && state.Started.Before(commit) {
				state.InSync = false
				if err := zkservice.UpdateServiceState(conn, &state); err != nil {
					glog.Errorf("Could not update service state %s for %s (%s) as out of sync: %s", state.ID, svc.Name, svc.ID, err)
					return err
				}
			}
		}
	}
	return nil
}

func (dfs *DistributedFilesystem) exportImages(dirpath string, templates map[string]servicetemplate.ServiceTemplate, services []service.Service) ([][]string, error) {
	tRepos, sRepos := getImageRefs(templates, services)
	imageTags, err := getImageTags(tRepos, sRepos)
	if err != nil {
		return nil, err
	}

	registry := fmt.Sprintf("%s:%d", dfs.dockerHost, dfs.dockerPort)
	i := 0
	var result [][]string
	for id, tags := range imageTags {
		filename := filepath.Join(dirpath, fmt.Sprintf("%d.tar", i))
		// Try to find the tag referring to the local registry, so we don't
		// make a call to Docker Hub potentially with invalid auth
		// Default to the first tag in the list
		if len(tags) == 0 {
			continue
		}

		tag := tags[0]
		for _, t := range tags {
			if strings.HasPrefix(t, registry) {
				tag = t
				break
			}
		}

		if err := saveImage(tag, filename); err == dockerclient.ErrNoSuchImage {
			glog.Warningf("Docker image %s was referenced, but does not exist. Skipping.", id)
			continue
		} else if err != nil {
			glog.Errorf("Could not export %s: %s", id, err)
			return nil, err
		}
		result = append(result, tags)
		i++
	}
	return result, nil
}

func (dfs *DistributedFilesystem) importImages(dirpath string, images [][]string, tenants map[string]struct{}) error {
	for i, tags := range images {
		filename := filepath.Join(dirpath, fmt.Sprintf("%d.tar", i))

		// Make sure all images that refer to a local registry are named with the local registry
		var imgs []string
		for _, id := range tags {
			image, err := commons.ParseImageID(id)
			if err != nil {
				glog.Errorf("Could not parse %s: %s", id, err)
				return err
			}
			if _, ok := tenants[image.User]; ok {
				image.Host, image.Port = dfs.dockerHost, dfs.dockerPort
			}
			imgs = append(imgs, image.String())
		}

		if err := loadImage(filename, imgs); err != nil {
			glog.Errorf("Error loading %s: %s", filename, err)
			return err
		}
	}
	return nil
}

func findImage(tenantID, uuid, tag string) (*docker.Image, error) {
	images, err := docker.Images()
	if err != nil {
		return nil, err
	}

	for _, image := range images {
		if image.ID.User == tenantID && image.UUID == uuid && image.ID.Tag == tag {
			return image, nil
		}
	}

	return nil, fmt.Errorf("image not found")
}

func findImages(tenantID, tag string) ([]*docker.Image, error) {
	images, err := docker.Images()
	if err != nil {
		return nil, err
	}

	var result []*docker.Image
	for _, image := range images {
		if image.ID.Tag == tag && image.ID.User == tenantID {
			result = append(result, image)
		}
	}
	return result, nil
}

func tag(tenantID, oldtag, newtag string) error {
	images, err := findImages(tenantID, oldtag)
	if err != nil {
		return err
	}

	var tagged []*docker.Image
	for _, image := range images {
		t, err := image.Tag(fmt.Sprintf("%s:%s", image.ID.BaseName(), newtag))
		if err != nil {
			glog.Errorf("Error while adding tags; rolling back: %s", err)
			for _, t := range tagged {
				if err := t.Delete(); err != nil {
					glog.Errorf("Could not untag image %s: %s", t.ID, err)
				}
			}
			return err
		}
		tagged = append(tagged, t)
	}
	return nil
}

func getImageTags(templateRepos []string, serviceRepos []string) (map[string][]string, error) {
	// make a map of all docker images
	images, err := docker.Images()
	if err != nil {
		return nil, err
	}

	// TODO: enable tagmap if we are storing all snapshots in a backup
	// tagmap := make(map[string][]string)
	imap := make(map[string]string)

	for _, image := range images {

		if image.ID.Tag == DockerLatest {
			image.ID.Tag = ""
		}
		// repo := image.ID.BaseName()
		// tagmap[repo] = append(tagmap[repo], image.ID.String())
		imap[image.ID.String()] = image.UUID
	}

	repos := append(templateRepos, serviceRepos...)

	// TODO: Enable this if we are storing all snapshots in a backup
	// Get all the tags related to a service
	/*
		repos := templateRepos
		for _, repo := range serviceRepos {
			imageID, err := commons.ParseImageID(repo)
			if err != nil {
				glog.Errorf("Invalid image %s: %s", repo, err)
				return nil, err
			}
			repos = append(repos, tagmap[imageID.BaseName()]...)
		}
	*/

	// Organize repos by UUID
	result := make(map[string][]string)
	for _, repo := range repos {
		if imageID, ok := imap[repo]; ok {
			result[imageID] = append(result[imageID], repo)
		} else {
			err := fmt.Errorf("not found: %s", repo)
			return nil, err
		}
	}

	return result, nil
}

func getImageRefs(templates map[string]servicetemplate.ServiceTemplate, services []service.Service) (t []string, s []string) {
	tmap := make(map[string]struct{})
	smap := make(map[string]struct{})

	var visit func(*[]servicedefinition.ServiceDefinition)
	visit = func(sds *[]servicedefinition.ServiceDefinition) {
		for _, sd := range *sds {
			if sd.ImageID != "" {
				tmap[sd.ImageID] = struct{}{}
			}
			visit(&sd.Services)
		}
	}

	for _, template := range templates {
		visit(&template.Services)
	}
	for _, service := range services {
		if service.ImageID != "" {
			smap[service.ImageID] = struct{}{}
		}
	}

	for r := range tmap {
		t = append(t, r)
	}

	for r := range smap {
		s = append(s, r)
	}

	return t, s
}

func saveImage(imageID, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		glog.Errorf("Could not create file %s: %s", filename, err)
		return err
	}

	defer func() {
		if err := file.Close(); err != nil {
			glog.Warningf("Could not close file %s: %s", filename, err)
		}
	}()

	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Cmd:   []string{"echo"},
				Image: imageID,
			},
		},
		dockerclient.HostConfig{},
	}

	ctr, err := docker.NewContainer(cd, false, 10*time.Second, nil, nil)
	if err != nil {
		glog.Errorf("Could not create container from image %s.  Have you synced lately?  (serviced docker sync): %s", imageID, err)
		return err
	}

	glog.V(1).Infof("Created container %s based on image %s", ctr.ID, imageID)
	defer func() {
		if err := ctr.Delete(true); err != nil {
			glog.Errorf("Could not remove container %s (%s): %s", ctr.ID, imageID, err)
		}
	}()

	if err := ctr.Export(file); err != nil {
		glog.Errorf("Could not export container %s (%s): %v", ctr.ID, imageID, err)
		return err
	}

	glog.Infof("Exported container %s (based on image %s) to %s", ctr.ID, imageID, filename)
	return nil
}

func loadImage(filename string, imageIDs []string) error {
	var images []string

	var image *docker.Image
	for _, id := range imageIDs {
		img, err := docker.FindImage(id, false)

		if err == docker.ErrNoSuchImage {
			images = append(images, id)
			continue
		} else if err != nil {
			glog.Errorf("Could not look up docker image %s: %s", id, err)
			return err
		}

		// verify the tag belongs to the right image
		if image != nil && img.UUID != image.UUID {
			err := fmt.Errorf("image conflict")
			glog.Errorf("Error checking docker image %s (%s) does not equal %s: %s", id, img.UUID, image.UUID, err)
			return err
		}
		image = img
	}

	// image not found so import
	if image == nil {
		// TODO: If the docker registry changes, do we need to update the tag?
		if err := docker.ImportImage(images[0], filename); err != nil {
			glog.Errorf("Could not import image from file %s: %s", filename, err)
			return err
		} else if image, err = docker.FindImage(images[0], false); err != nil {
			glog.Errorf("Could not look up docker image %s: %s", images[0], err)
			return err
		}
		images = images[1:]
	}

	// tag remaining images
	for _, id := range images {
		if _, err := image.Tag(id); err != nil {
			glog.Errorf("Could not tag image %s as %s: %s", image.UUID, id, err)
			return err
		}
	}

	return nil
}
