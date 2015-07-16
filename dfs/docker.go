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
	"time"

	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/layer"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/zzk"
	zkservice "github.com/control-center/serviced/zzk/service"
	dockerclient "github.com/fsouza/go-dockerclient"
	"github.com/zenoss/glog"
)

type imagemeta struct {
	UUID     string
	Tags     []string
	Filename string
}

// ResetRegistry will update the host:port of the docker registry
func (dfs *DistributedFilesystem) ResetRegistry() error {
	// get all the services in the system
	svcs, err := dfs.facade.GetServices(dfs.datastoreGet(), dao.ServiceRequest{})
	if err != nil {
		glog.Errorf("Could not get services for updating the registry")
		return err
	}

	imagemap := make(map[string]struct{})
	for _, svc := range svcs {
		imageID, err := commons.ParseImageID(svc.ImageID)
		if err != nil {
			glog.Errorf("Could not parse image ID (%s) for service %s (%s): %s", svc.ImageID, svc.Name, svc.ID, err)
			return err
		}

		if imageID.Host == dfs.dockerHost && imageID.Port == dfs.dockerPort {
			continue
		}

		if _, ok := imagemap[imageID.BaseName()]; !ok {
			if err := dfs.registerImages(imageID.BaseName()); err != nil {
				glog.Errorf("Could not reregister image %s: %s", imageID.BaseName(), err)
				return err
			}
			imagemap[imageID.BaseName()] = struct{}{}
		}

		imageID.Host, imageID.Port = dfs.dockerHost, dfs.dockerPort
		svc.ImageID = imageID.String()
		if err := dfs.facade.UpdateService(dfs.datastoreGet(), svc); err != nil {
			glog.Errorf("Could not update service %s (%s) with image %s", svc.Name, svc.ID, svc.ImageID)
			return err
		}
	}

	return nil
}

// Commit will merge a container into existing services' image
func (dfs *DistributedFilesystem) Commit(dockerID string) (string, error) {
	// get the container
	ctr, err := docker.FindContainer(dockerID)
	if err != nil {
		glog.Errorf("Could not get container %s: %s", dockerID, err)
		return "", err
	}

	// verify the container is not currently running
	if ctr.IsRunning() {
		err := fmt.Errorf("cannot commit a running container")
		glog.Errorf("Error committing container %s: %s", ctr.ID, err)
		return "", err
	}

	// find the image to commit (ctr.Config.Image is the repotag)
	image, err := docker.FindImage(ctr.Config.Image, false)
	if err != nil {
		glog.Errorf("Could not find image %s from %s: %s", ctr.Config.Image, dockerID, err)
		return "", err
	}

	// verify the container is not stale (ctr.Image is the UUID)
	if !image.ID.IsLatest() || image.UUID != ctr.Image {
		err := fmt.Errorf("cannot commit a stale container")
		glog.Errorf("Could not commit %s (%s): %s", dockerID, image.ID, err)
		return "", err
	}

	// verify the tenantID
	tenantID, err := dfs.facade.GetTenantID(dfs.datastoreGet(), image.ID.User)
	if err != nil {
		glog.Errorf("Could not look up tenant %s from image %s for container %s: %s", image.ID.User, image.ID, dockerID, err)
		return "", err
	} else if tenantID != image.ID.User {
		err := fmt.Errorf("service is not the tenant")
		glog.Errorf("Could not commit %s (%s): %s", dockerID, image.ID, err)
		return "", err
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
	newImage, err := ctr.Commit(image.ID.BaseName())
	if err != nil {
		glog.Errorf("Could not commit %s (%s): %s", dockerID, image.ID, err)
		return "", err
	}

	// desynchronize any running containers
	if err := dfs.desynchronize(newImage); err != nil {
		glog.Warningf("Could not denote all desynchronized services: %s", err)
	}

	// snapshot the filesystem and images
	snapshotID, err := dfs.Snapshot(tenantID, "")
	if err != nil {
		glog.Errorf("Could not create a snapshot of the new image %s: %s", tenantID, err)
		return "", err
	}

	return snapshotID, nil
}

func (dfs *DistributedFilesystem) desynchronize(image *docker.Image) error {
	// inspect the image
	dImg, err := image.Inspect()
	if err != nil {
		glog.Errorf("Could not inspect image %s (%s): %s", image.ID, image.UUID, err)
		return err
	}

	// look up services for that tenant
	svcs, err := dfs.facade.GetServices(dfs.datastoreGet(), dao.ServiceRequest{TenantID: image.ID.User})
	if err != nil {
		glog.Errorf("Could not get services for tenant %s from %s (%s): %s", image.ID.User, image.ID, image.UUID, err)
		return err
	}

	for _, svc := range svcs {
		// figure out which services are using the provided image
		svcImageID, err := commons.ParseImageID(svc.ImageID)
		if err != nil {
			glog.Warningf("Could not parse image %s for %s (%s): %s", svc.ImageID, svc.Name, svc.ID)
			continue
		} else if !svcImageID.Equals(image.ID) {
			continue
		}

		// TODO: we need to switch to using dao.ControlPlane
		conn, err := zzk.GetLocalConnection(zzk.GeneratePoolPath(svc.PoolID))
		if err != nil {
			glog.Warningf("Could not acquire connection to the coordinator (%s): %s", svc.PoolID, err)
			continue
		}

		states, err := zkservice.GetServiceStates(conn, svc.ID)
		if err != nil {
			glog.Warningf("Could not get running services for %s (%s): %s", svc.Name, svc.ID)
			continue
		}

		for _, state := range states {
			// check if the instance has been running since before the commit
			if state.IsRunning() && state.Started.Before(dImg.Created) {
				state.InSync = false
				if err := zkservice.UpdateServiceState(conn, &state); err != nil {
					glog.Warningf("Could not update service state %s for %s (%s) as out of sync: %s", state.ID, svc.Name, svc.ID, err)
					continue
				}
			}
		}
	}
	return nil
}

func (dfs *DistributedFilesystem) exportImages(dirpath string, templates map[string]servicetemplate.ServiceTemplate, services []service.Service, labels []string) ([]imagemeta, error) {
	tRepos, sRepos := getImageRefs(templates, services)
	imageTags, err := getImageTags(tRepos, sRepos, labels)
	if err != nil {
		return nil, err
	}

	filename := filepath.Join(dirpath, "images.tar")
	file, err := os.Create(filename)
	if err != nil {
		glog.Errorf("Could not create file %s: %s", filename, err)
		return nil, err
	}

	defer func() {
		if err := file.Close(); err != nil {
			glog.Warningf("Could not close file %s: %s", filename, err)
		}
	}()

	var result []imagemeta
	for uuid, tags := range imageTags {
		metadata := imagemeta{Filename: "images.tar", UUID: uuid, Tags: tags}
		result = append(result, metadata)
	}

	if err := docker.SaveImages(file, append(tRepos, sRepos...)...); err != nil {
		glog.Errorf("Could not export: %s", err)
		return nil, err
	}

	return result, nil
}

func (dfs *DistributedFilesystem) importImages(dirpath string, images []imagemeta, tenants map[string]struct{}) error {
	for _, metadata := range images {
		filename := filepath.Join(dirpath, metadata.Filename)

		// Make sure all images that refer to a local registry are named with the local registry
		tags := make([]string, len(metadata.Tags))
		for i, tag := range metadata.Tags {
			imageID, err := commons.ParseImageID(tag)
			if err != nil {
				glog.Errorf("Could not parse %s: %s", tag, err)
				return err
			}

			if _, ok := tenants[imageID.User]; ok {
				imageID.Host, imageID.Port = dfs.dockerHost, dfs.dockerPort
			}
			tags[i] = imageID.String()
		}

		if err := loadImage(filename, metadata.UUID, tags); err != nil {
			glog.Errorf("Error loading %s (%s): %s", filename, metadata.UUID, err)
			return err
		}
	}
	return nil
}

func (dfs *DistributedFilesystem) registerImages(basename string) error {
	images, err := docker.Images()
	if err != nil {
		return err
	}

	for _, image := range images {
		imageID := image.ID
		if imageID.BaseName() == basename {
			imageID.Host, imageID.Port = dfs.dockerHost, dfs.dockerPort
			if t, err := image.Tag(imageID.String()); err != nil {
				glog.Errorf("Could not add tag %s to %s: %s", imageID, image.ID, err)
				return err
			} else if err := docker.PushImage(t.ID.String()); err != nil {
				glog.Errorf("Could not push image %s: %s", t.ID, err)
				return err
			}
		}
	}

	return nil
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

func searchImagesByTenantID(tenantID string) ([]*docker.Image, error) {
	images, err := docker.Images()
	if err != nil {
		return nil, err
	}

	var result []*docker.Image
	for i, image := range images {
		if image.ID.User == tenantID {
			result = append(result, images[i])
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

func getImageTags(templateRepos []string, serviceRepos []string, tags []string) (map[string][]string, error) {
	imagemap := make(map[string][]string)

	// find all the template repos
	for _, repo := range templateRepos {
		image, err := docker.FindImage(repo, false)
		if err == docker.ErrNoSuchImage {
			glog.Warningf("Could not find template image %s", repo)
			continue
		} else if err != nil {
			glog.Errorf("Could not look up repo %s: %s", repo, err)
			return nil, err
		}
		if image.ID.Tag == docker.DockerLatest {
			image.ID.Tag = ""
		}
		images := imagemap[image.UUID]
		imagemap[image.UUID] = append(images, image.ID.String())
	}

	// find all the service repos
	for _, repo := range serviceRepos {
		image, err := docker.FindImage(repo, false)
		if err != nil {
			glog.Errorf("Could not look up repo %s: %s", repo, err)
			return nil, err
		}
		if image.ID.Tag == docker.DockerLatest {
			image.ID.Tag = ""
		}
		images := imagemap[image.UUID]
		imagemap[image.UUID] = append(images, image.ID.String())

		for _, tag := range tags {
			image, err := docker.FindImage(commons.JoinRepoTag(repo, tag), false)
			if err == docker.ErrNoSuchImage {
				continue
			} else if err != nil {
				glog.Errorf("Could not look up repo %s: %s", commons.JoinRepoTag(repo, tag), err)
				return nil, err
			}
			images := imagemap[image.UUID]
			imagemap[image.UUID] = append(images, image.ID.String())
		}
	}

	return imagemap, nil
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

// DEPRECATED
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

func loadImage(filename string, uuid string, tags []string) error {
	// look up the image by UUID
	images, err := docker.Images()
	if err != nil {
		glog.Errorf("Could not look up images: %s", err)
		return err
	}

	var image *docker.Image
	for _, i := range images {
		if i.UUID == uuid {
			image = i
			break
		}
	}

	// image not found so import
	if image == nil {
		glog.Warningf("Importing image from file, don't forget to sync (serviced docker sync)")
		if name := filepath.Base(filename); name == "images.tar" {
			if err := docker.LoadImages(filename); err != nil {
				glog.Errorf("Could not import images from file %s: %s", filename, err)
				return err
			}
			return nil
		}

		if err := docker.ImportImage(tags[0], filename); err != nil {
			glog.Errorf("Could not import image from file %s: %s", filename, err)
			return err
		} else if image, err = docker.FindImage(tags[0], false); err != nil {
			glog.Errorf("Could not look up docker image %s: %s", tags[0], err)
			return err
		}
		glog.Infof("Tagging images %v at %s", tags, image.UUID)
		tags = tags[1:]
	}

	// tag the remaining images
	for _, tag := range tags {
		if _, err := image.Tag(tag); err != nil {
			glog.Errorf("Could not tag image %s as %s: %s", image.UUID, tag, err)
			return err
		}
	}

	return nil
}
