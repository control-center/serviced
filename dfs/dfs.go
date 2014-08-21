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

package dfs

import (
	"github.com/control-center/serviced/commons"
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/commons/layer"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicestate"
	"github.com/control-center/serviced/facade"
	"github.com/control-center/serviced/node"
	"github.com/control-center/serviced/volume"
	"github.com/zenoss/glog"

	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

const (
	USER_ROOT        string = "root"
	DOCKER_ENDPOINT  string = "unix:///var/run/docker.sock"
	DOCKER_LATEST    string = "latest"
	DOCKER_IMAGEJSON string = "images.json"
)

var (
	unused interface{}
	// stubs
	getCurrentUser = user.Current
)

type DistributedFileSystem struct {
	sync.Mutex
	client  dao.ControlPlane
	facade  *facade.Facade
	timeout time.Duration
}

// Initiates a New Distributed Filesystem Object given an implementation of a control plane object
func NewDistributedFileSystem(client dao.ControlPlane, facade *facade.Facade, timeout time.Duration) (*DistributedFileSystem, error) {
	return &DistributedFileSystem{
		client:  client,
		facade:  facade,
		timeout: timeout,
	}, nil
}

// waitpause waits for a service's instances to pause
func (d *DistributedFileSystem) waitpause(cancel <-chan interface{}, serviceID string) error {

	for {
		var states []*servicestate.ServiceState
		if err := d.client.GetServiceStates(serviceID, &states); err != nil {
			return err
		}

		paused := true
		for _, state := range states {
			if !state.IsPaused() {
				paused = false
				break
			}
		}
		if paused {
			return nil
		}
		select {
		case <-time.After(time.Second):
			// pass
		case <-cancel:
			return fmt.Errorf("waitpause timeout")
		}
	}
}

// Snapshots the DFS
func (d *DistributedFileSystem) Snapshot(tenantId string) (string, error) {
	cancel := make(chan interface{})
	go func() {
		defer close(cancel)
		<-time.After(d.timeout)
	}()

	// Get the service
	var myService service.Service
	if err := d.client.GetService(tenantId, &myService); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot tenant=%+v err=%s", tenantId, err)
		return "", err
	}

	var servicesList []*service.Service
	if err := d.client.GetServices(unused, &servicesList); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", myService.ID, err)
		return "", err
	}

	// Pause all running services
	for _, svc := range servicesList {
		if svc.DesiredState == service.SVCRun {
			defer d.facade.StartService(datastore.Get(), svc.ID)
			if err := d.facade.PauseService(datastore.Get(), svc.ID); err != nil {
				err = fmt.Errorf("could not pause %s (%s): %s", svc.Name, svc.ID, err)
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", svc.ID, err)
				return "", err
			}

			if err := d.waitpause(cancel, svc.ID); err != nil {
				err = fmt.Errorf("could not pause %s (%s): %s", svc.Name, svc.ID, err)
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%s err %s", svc.ID, err)
				return "", err
			}
		}
	}

	// create a snapshot
	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", myService.ID, err)
		return "", err
	}

	label := node.GetLabel(tenantId)
	glog.Infof("DistributedFileSystem.Snapshot service=%+v label=%+v volume=%+v", myService.ID, label, theVolume)

	parts := strings.SplitN(label, "_", 2)
	if len(parts) < 2 {
		err := errors.New("invalid label")
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v label=%s err=%s", myService.ID, parts, err)
		return "", err
	}

	tag := parts[1]

	// Add snapshot to the volume
	if err := theVolume.Snapshot(label); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", myService.ID, err)
		return "", err
	}

	// Add tags to the images
	if err := d.tag(tenantId, DOCKER_LATEST, tag); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", myService.ID, err)
		return "", err
	}

	// Dump all service definitions
	snapshotPath := func(relPath ...string) string {
		return filepath.Join(append([]string{theVolume.SnapshotPath(label)}, relPath...)...)
	}
	if e := writeJsonToFile(servicesList, snapshotPath("services.json")); e != nil {
		glog.Errorf("Could not write services.json: %v", e)
		return "", e
	}

	glog.V(0).Infof("Successfully created snapshot for service Id:%s Name:%s Label:%s", myService.ID, myService.Name, label)
	return label, nil
}

// Deletes a snapshot from the DFS
func (d *DistributedFileSystem) DeleteSnapshot(snapshotId string) error {
	d.Lock()
	defer d.Unlock()

	parts := strings.SplitN(snapshotId, "_", 2)
	if len(parts) < 2 {
		err := errors.New("malformed snapshot")
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s err=%s", snapshotId, err)
		return err
	}

	tenantId := parts[0]
	timestamp := parts[1]

	var service service.Service
	if err := d.client.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s err=%s", snapshotId, err)
		return err
	}

	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s service=%s err=%s", snapshotId, service.ID, err)
		return err
	}

	glog.V(2).Infof("Deleting snapshot %s", snapshotId)
	if err := theVolume.RemoveSnapshot(snapshotId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s err=%s", snapshotId, err)
		return err
	}

	glog.V(2).Infof("Removing snapshot tags (%s)", snapshotId)
	if images, err := d.findImages(tenantId, timestamp); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s err=%s", snapshotId, err)
		return err
	} else {
		for _, image := range images {
			ssid := image.ID
			ssid.Tag = timestamp
			ssimg, err := docker.FindImage(ssid.String(), false)
			if err != nil {
				glog.Errorf("unable to untag image: %s (%s)", image.ID, err)
				continue
			}
			ssimg.Delete()
		}
	}

	return nil
}

// Deletes snapshots of a service
func (d *DistributedFileSystem) DeleteSnapshots(tenantId string) error {
	d.Lock()
	defer d.Unlock()

	// Delete the snapshot subvolume
	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot tenant=%s err=%s", tenantId, err)
		return err
	} else if err := theVolume.Unmount(); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot tenant=%s err=%s", tenantId, err)
	}

	// Delete the docker repos
	images, err := d.findImages(tenantId, DOCKER_LATEST)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshots tenantId=%s err=%s", tenantId, err)
		return err
	}
	for _, image := range images {
		img, err := docker.FindImage(image.ID.String(), false)
		if err != nil {
			glog.Errorf("error trying to delete image %s, err=%s", image.ID, err)
			err = errors.New("error(s) while removing service images")
			continue
		}
		img.Delete()
	}
	if err != nil {
		glog.V(2).Infof("DistibutedFileSystem.DeleteSnapshots tenantId=%s err=%s", tenantId, err)
		return err
	}

	return nil
}

// Commits a container to docker image and updates the DFS
func (d *DistributedFileSystem) Commit(dockerId string) (string, error) {
	d.Lock()
	defer d.Unlock()

	// Get the container, and verify that it is not running
	container, err := docker.FindContainer(dockerId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	if container.IsRunning() {
		err := errors.New("cannot commit a running container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	// Parse the image information
	imageId, err := commons.ParseImageID(container.Config.Image)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}
	tenantId := imageId.User

	// Verify the image exists and has the latest tag
	images, err := d.findImages(tenantId, DOCKER_LATEST)
	glog.V(2).Infof("DistributedFileSystem.Commit found %d matching images: id=%s", len(images), tenantId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	var image *docker.Image
	for _, i := range images {
		if i.UUID == container.Image {
			image = i
			break
		}
	}
	// If not found or not tagged as latest, then the container is stale and cannot be committed.
	if image == nil {
		err := errors.New("cannot commit a stale container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	// Check the number of image layers
	layers, err := image.History()
	if err != nil {
		glog.Errorf("Checking history on %s: %s", image.ID, err)
	}
	if len(layers) > layer.WARN_LAYER_COUNT {
		glog.Warningf("Image %s number of layers (%d) approaching maximum (%d).  Please squash image layers.",
			image.ID, len(layers), layer.MAX_LAYER_COUNT)
	} else {
		glog.Infof("Image '%s' number of layers: %d", image.ID, len(layers))
	}

	// Commit the container to the image and tag
	if _, err := container.Commit(image.ID.BaseName()); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return "", err
	}

	// Mark any running containers as out of sync
	if err := d.desynchronize(image.ID, time.Now()); err != nil {
		glog.Warningf("Could not mark all desynchronized services: %s", err)
	}

	// Update the dfs
	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return "", err
	}

	// Snapshot the filesystem and images
	output, err := d.Snapshot(tenantId)
	if err != nil {
		err = fmt.Errorf("failed to create snapshot: %s", err)
	}
	return output, err
}

// Rolls back the DFS to a specified state and retags the images
func (d *DistributedFileSystem) Rollback(snapshotId string) error {
	d.Lock()
	defer d.Unlock()

	// Get the tenant and the timestamp from the snapshotId
	parts := strings.SplitN(snapshotId, "_", 2)
	if len(parts) < 2 {
		err := errors.New("malformed snapshot id")
		glog.V(2).Infof("DistributedFileSystem.Rollback snapshot=%s, err=%s", snapshotId, err)
		return err
	}
	tenantId := parts[0]
	timestamp := parts[1]

	var (
		services  []*service.Service
		theVolume volume.Volume
	)

	// Fail if any services have running instances
	glog.V(3).Infof("DistributedFileSystem.Rollback checking service states")
	if err := d.client.GetServices(unused, &services); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
		return err
	}
	for _, service := range services {
		var states []*servicestate.ServiceState
		if err := d.client.GetServiceStates(service.ID, &states); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
			return err
		}
		if numstates := len(states); numstates > 0 {
			err := errors.New(fmt.Sprintf("%s has %d running services. Stop all services before rolling back", service.ID, numstates))
			glog.V(2).Info("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
			return err
		}
	}

	// Validate existence of images for this snapshot
	glog.V(3).Infof("DistributedFileSystem.Rollback validating image for service instance: %s", tenantId)
	var service service.Service
	err := d.client.GetService(tenantId, &service)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
		return err
	}

	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
		return err
	}

	// Rollback the dfs
	glog.V(0).Infof("performing rollback on serviceId: %s to snaphotId: %s", service.ID, snapshotId)
	if err := theVolume.Rollback(snapshotId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.ID, err)
		return err
	}

	// Set tags on the images
	glog.V(3).Infof("DistributedFileSystem.Rollback retagging snapshots tenant=%s", tenantId)
	if err := d.tag(tenantId, timestamp, DOCKER_LATEST); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.ID, err)
		return err
	}

	// Restore service definitions and services
	if err := d.RollbackServices(theVolume.SnapshotPath(snapshotId)); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.ID, err)
		return err
	}

	return nil
}

func (d *DistributedFileSystem) RollbackServices(restorePath string) error {
	glog.Infof("DistributedFileSystem.RollbackServices from path: %s", restorePath)

	var (
		existingServices []*service.Service
		services         []*service.Service
	)

	// Verify there are no running service instances
	var rss []*dao.RunningService
	if err := d.client.GetRunningServices(unused, &rss); err != nil {
		glog.Errorf("Could not get running services: %s", err)
		return err
	} else if count := len(rss); count > 0 {
		glog.Errorf("Found %d running services; stop all services before restoring from backup", count)
		return fmt.Errorf("found %d running services", count)
	}

	// Read the service definitions
	servicesPath := filepath.Join(restorePath, "services.json")
	if e := readJsonFromFile(&services, servicesPath); e != nil {
		glog.Errorf("Could not read services from %s: %v", servicesPath, e)
		return e
	}

	// Restore the services ...
	if err := d.client.GetServices(unused, &existingServices); err != nil {
		glog.Errorf("Could not get existing services: %s", err)
		return err
	}

	existingPools := make(map[string]*pool.ResourcePool)
	if pools, e := d.facade.GetResourcePools(datastore.Get()); e != nil {
		glog.Errorf("Could not get existing pools: %v", e)
		return e
	} else {
		for _, pool := range pools {
			glog.Infof("caching to existingPools: %v", pool)
			existingPools[pool.ID] = pool
		}
	}

	existingServiceMap := make(map[string]*service.Service)
	for _, service := range existingServices {
		existingServiceMap[service.ID] = service
	}
	for _, svc := range services {
		// Change the service read from file to have desired state of stopped
		if svc.DesiredState != service.SVCStop {
			glog.V(2).Infof("Service %s (%s) was running (svc.DesiredState:%v).  Setting desired state to SVCStop", svc.Name, svc.ID, svc.DesiredState)
			svc.DesiredState = service.SVCStop
		}

		if existingService := existingServiceMap[svc.ID]; existingService != nil {
			var unused *int
			svc.PoolID = existingService.PoolID
			if existingPools[svc.PoolID] == nil {
				glog.Infof("Changing PoolID of service %s from %s to default", svc.ID, svc.PoolID)
				svc.PoolID = "default"
			}
			if e := d.client.UpdateService(*svc, unused); e != nil {
				glog.Errorf("Could not update service %s: %v", svc.ID, e)
				return e
			}
		} else {
			if existingPools[svc.PoolID] == nil {
				glog.Infof("Changing PoolID of service %s from %s to default", svc.ID, svc.PoolID)
				svc.PoolID = "default"
			}
			var serviceId string
			if e := d.client.AddService(*svc, &serviceId); e != nil {
				glog.Errorf("Could not add service %s: %v", svc.ID, e)
				return e
			}
			if svc.ID != serviceId {
				msg := fmt.Sprintf("BUG!!! ADDED SERVICE %s, BUT WITH THE WRONG ID: %s", svc.ID, serviceId)
				glog.Errorf(msg)
				return errors.New(msg)
			}
			existingServiceMap[svc.ID] = svc
		}
	}

	return nil
}

func (d *DistributedFileSystem) findImages(id, tag string) ([]*docker.Image, error) {
	result := []*docker.Image{}

	images, err := docker.Images()
	if err != nil {
		return result, err
	}

	for _, image := range images {
		if image.ID.Tag == tag && image.ID.User == id {
			result = append(result, image)
		}
	}

	return result, nil
}

func (d *DistributedFileSystem) tag(id, oldtag, newtag string) error {
	images, err := d.findImages(id, oldtag)
	if err != nil {
		return err
	}

	tagged := []*docker.Image{}
	for _, image := range images {
		ti, err := image.Tag(fmt.Sprintf("%s:%s", image.ID.BaseName(), newtag))
		if err != nil {
			glog.Errorf("error (%v) while adding tags, rolling back ...", err)
			for _, taggedimg := range tagged {
				if delerr := taggedimg.Delete(); delerr != nil {
					glog.Errorf("cannont untag image %s: %v", taggedimg.ID, delerr)
				}
			}
			return err
		}

		tagged = append(tagged, ti)
	}

	return nil
}

// desynchronize marks all service states using a particular ImageID as out of
// sync if started before the time of commit
func (d *DistributedFileSystem) desynchronize(imageID commons.ImageID, commit time.Time) error {
	svcs, err := d.facade.GetServices(datastore.Get())
	if err != nil {
		return err
	}

	for _, svc := range svcs {
		// figure out which services use the provided image
		if svcImage, err := commons.ParseImageID(svc.ImageID); err != nil {
			return err
		} else if !svcImage.Equals(imageID) {
			continue
		}

		var states []*servicestate.ServiceState
		if err := d.client.GetServiceStates(svc.ID, &states); err != nil {
			return err
		}
		for _, state := range states {
			// check if the instance has been running since before the commit
			if state.IsRunning() && state.Started.Before(commit) {
				state.InSync = false
				if err := d.client.UpdateServiceState(*state, new(int)); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var writeJsonToFile = func(v interface{}, filename string) (err error) {
	file, e := os.Create(filename)
	if e != nil {
		glog.Errorf("Could not create file %s: %v", filename, e)
		return e
	}
	defer func() {
		if e := file.Close(); e != nil {
			glog.Errorf("Error while closing file %s: %v", filename, e)
			if err == nil {
				err = e
			}
		}
	}()
	encoder := json.NewEncoder(file)
	if e := encoder.Encode(v); e != nil {
		glog.Errorf("Could not write JSON data to %s: %v", filename, e)
		return e
	}
	return nil
}

var readJsonFromFile = func(v interface{}, filename string) error {
	file, e := os.Open(filename)
	if e != nil {
		glog.Errorf("Could not open file %s: %v", filename, e)
		return e
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	if e := decoder.Decode(v); e != nil {
		glog.Errorf("Could not read JSON data from %s: %v", filename, e)
		return e
	}
	return nil
}
