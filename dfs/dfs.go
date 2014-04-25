package dfs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	//"github.com/zenoss/serviced/datastore"
	"github.com/zenoss/serviced/domain/pool"
	"github.com/zenoss/serviced/volume"

	docker "github.com/zenoss/go-dockerclient"

	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path/filepath"
	"strings"
	"sync"
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

// runServiceCommand attaches to a service state container and executes an arbitrary bash command
var runServiceCommand = func(state *dao.ServiceState, command string) ([]byte, error) {
	nsinitPath, err := exec.LookPath("nsinit")
	if err != nil {
		return []byte{}, err
	}

	NSINIT_ROOT := "/var/lib/docker/execdriver/native" // has container.json

	hostCommand := []string{"/bin/bash", "-c",
		fmt.Sprintf("cd %s/%s && %s exec bash -c '%s'", NSINIT_ROOT, state.DockerId, nsinitPath, command)}
	glog.Infof("ServiceId: %s, Command: %s", state.ServiceId, strings.Join(hostCommand, " "))
	cmd := exec.Command(hostCommand[0], hostCommand[1:]...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command: `%s` for serviceId: %s out: %s err: %s", command, state.ServiceId, output, err)
		return output, err
	}
	glog.Infof("Successfully ran command: `%s` for serviceId: %s out: %s", command, state.ServiceId, output)
	return output, nil
}

type DistributedFileSystem struct {
	sync.Mutex
	client       dao.ControlPlane
	dockerClient *docker.Client
}

// Initiates a New Distributed Filesystem Object given an implementation of a control plane object
func NewDistributedFileSystem(client dao.ControlPlane) (*DistributedFileSystem, error) {
	dockerClient, err := docker.NewClient(DOCKER_ENDPOINT)
	if err != nil {
		glog.V(2).Infof("snapshot.NewDockerClient client=%+v err=%s", client, err)
		return nil, err
	}

	return &DistributedFileSystem{
		client:       client,
		dockerClient: dockerClient,
	}, nil
}

// Pauses a running service
func (d *DistributedFileSystem) Pause(service *dao.Service, state *dao.ServiceState) error {
	if output, err := runServiceCommand(state, service.Snapshot.Pause); err != nil {
		errmsg := fmt.Sprintf("output: %s, err: %s", output, err)
		glog.V(2).Infof("DistributedFileSystem.Pause service=%+v err=%s", service, err)
		return errors.New(errmsg)
	}
	return nil
}

// Resumes a paused service
func (d *DistributedFileSystem) Resume(service *dao.Service, state *dao.ServiceState) error {
	if output, err := runServiceCommand(state, service.Snapshot.Resume); err != nil {
		errmsg := fmt.Sprintf("output: %s, err: %s", output, err)
		glog.V(2).Infof("DistributedFileSystem.Resume service=%+v err=%s", service, err)
		return errors.New(errmsg)
	}
	return nil
}

// Snapshots the DFS
func (d *DistributedFileSystem) Snapshot(tenantId string) (string, error) {
	// Get the service
	var service dao.Service
	if err := d.client.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot tenant=%+v err=%s", tenantId, err)
		return "", err
	}

	iamRoot := false
	warnedAboutNonRoot := false

	// Only the root user can pause and resume services
	if whoami, err := getCurrentUser(); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
		return "", err
	} else if USER_ROOT == whoami.Username {
		iamRoot = true
	}

	var servicesList []*dao.Service
	if err := d.client.GetServices(unused, &servicesList); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
		return "", err
	}

	for _, service := range servicesList {
		if service.Snapshot.Pause == "" || service.Snapshot.Resume == "" {
			continue
		}

		var states []*dao.ServiceState
		if err := d.client.GetServiceStates(service.Id, &states); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v, err=%s", service.Id, err)
			return "", err
		}

		// Pause all running service states
		for i, state := range states {
			glog.V(3).Infof("DEBUG states[%d]: service:%+v state:%+v", i, service.Id, state.DockerId)
			if state.DockerId != "" {
				if iamRoot {
					err := d.Pause(service, state)
					defer d.Resume(service, state) // resume service state when snapshot is done
					if err != nil {
						glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
						return "", err
					}
				} else if !warnedAboutNonRoot {
					warnedAboutNonRoot = true
					glog.Warningf("Unable to pause/resume service - User is not %s", USER_ROOT)
				}
			}
		}
	}

	// create a snapshot
	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
		return "", err
	}

	label := serviced.GetLabel(tenantId)
	glog.Infof("DistributedFileSystem.Snapshot service=%+v label=%+v volume=%+v", service.Id, label, theVolume)

	parts := strings.SplitN(label, "_", 2)
	if len(parts) < 2 {
		err := errors.New("invalid label")
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v label=%s err=%s", service.Id, parts, err)
		return "", err
	}

	tag := parts[1]

	// Add tags to the images
	if err := d.tag(tenantId, DOCKER_LATEST, tag); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
		return "", err
	}

	// Add snapshot to the volume
	if err := theVolume.Snapshot(label); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", service.Id, err)
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

	glog.V(0).Infof("Successfully created snapshot for service Id:%s Name:%s Label:%s", service.Id, service.Name, label)
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

	var service dao.Service
	if err := d.client.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s err=%s", snapshotId, err)
		return err
	}

	var theVolume volume.Volume
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.DeleteSnapshot snapshotId=%s service=%s err=%s", snapshotId, service.Id, err)
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
			repo := image.Repository + ":" + timestamp
			if err := d.dockerClient.RemoveImage(repo); err != nil {
				glog.Errorf("unable to untag image: %s (%s)", image.ID, err)
			}
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
		if err := d.dockerClient.RemoveImage(image.Repository); err != nil {
			glog.Errorf("error trying to delete image %s, err=%s", image.Repository, err)
			err = errors.New("error(s) while removing service images")
		}
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
	container, err := d.dockerClient.InspectContainer(dockerId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	} else if container.State.Running {
		err := errors.New("cannot commit a running container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	// Parse the image information
	imageID := container.Config.Image
	repopath := strings.SplitN(imageID, ":", 2)[0]
	parts := strings.SplitN(repopath, "/", 3)
	reponame := parts[len(parts)-1]
	id := strings.SplitN(reponame, "_", 2)[0]

	// Verify the image exists and has the latest tag
	var image *docker.APIImages
	images, err := d.findImages(id, DOCKER_LATEST)
	glog.V(2).Infof("DistributedFileSystem.Commit found %d matching images: id=%s", len(images), id)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}
	for _, i := range images {
		if i.ID == container.Image {
			image = &i
			break
		}
	}
	// If not found or not tagged as latest, then the container is stale and cannot be committed.
	if image == nil {
		err := errors.New("cannot commit a stale container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return "", err
	}

	// Commit the container to the image and tag
	options := docker.CommitContainerOptions{
		Container:  container.ID,
		Repository: image.Repository,
	}
	if _, err := d.dockerClient.CommitContainer(options); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return "", err
	}

	// Update the dfs
	var theVolume volume.Volume
	if err := d.client.GetVolume(id, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return "", err
	}

	// Snapshot the filesystem and images
	return d.Snapshot(id)
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
		services  []*dao.Service
		theVolume volume.Volume
	)

	// Fail if any services have running instances
	glog.V(3).Infof("DistributedFileSystem.Rollback checking service states")
	if err := d.client.GetServices(unused, &services); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
		return err
	}
	for _, service := range services {
		var states []*dao.ServiceState
		if err := d.client.GetServiceStates(service.Id, &states); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
			return err
		}
		if numstates := len(states); numstates > 0 {
			err := errors.New(fmt.Sprintf("%s has %d running services. Stop all services before rolling back", service.Id, numstates))
			glog.V(2).Info("DistributedFileSystem.Rollback tenant=%+v err=%s", tenantId, err)
			return err
		}
	}

	// Validate existence of images for this snapshot
	glog.V(3).Infof("DistributedFileSystem.Rollback validating image for service instance: %s", tenantId)
	var service dao.Service
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
	glog.V(0).Infof("performing rollback on serviceId: %s to snaphotId: %s", service.Id, snapshotId)
	if err := theVolume.Rollback(snapshotId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.Id, err)
		return err
	}

	// Set tags on the images
	glog.V(3).Infof("DistributedFileSystem.Rollback retagging snapshots tenant=%s", tenantId)
	if err := d.tag(tenantId, timestamp, DOCKER_LATEST); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.Id, err)
		return err
	}

	// Restore service definitions and services
	if err := d.rollbackServices(theVolume.SnapshotPath(snapshotId)); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", service.Id, err)
		return err
	}

	return nil
}

func (d *DistributedFileSystem) rollbackServices(restorePath string) error {
	glog.Infof("DistributedFileSystem.rollbackServices from path: %s", restorePath)

	var (
		existingServices []*dao.Service
		existingPools    map[string]*pool.ResourcePool
		services         []*dao.Service
	)

	// Read the service definitions
	servicesPath := filepath.Join(restorePath, "services.json")
	if e := readJsonFromFile(&services, servicesPath); e != nil {
		glog.Errorf("Could not read services from %s: %v", servicesPath, e)
		return e
	}

	// Restore the services ...
	var request dao.EntityRequest
	if e := d.client.GetServices(request, &existingServices); e != nil {
		glog.Errorf("Could not get existing services: %v", e)
		return e
	}

	/*
		// TODO: populate existingPools using facade
		if pools, e := this.facade.GetResourcePools(datastore.Get()); e != nil {
			glog.Errorf("Could not get existing pools: %v", e)
			return e
		} else {
			for _, pool := range pools {
				existingPools[pool.ID] = pool
			}
		}
	*/

	existingServiceMap := make(map[string]*dao.Service)
	for _, service := range existingServices {
		existingServiceMap[service.Id] = service
	}
	for _, service := range services {
		if existingService := existingServiceMap[service.Id]; existingService != nil {
			var unused *int
			if e := d.client.StopService(service.Id, unused); e != nil {
				glog.Errorf("Could not stop service %s: %v", service.Id, e)
				return e
			}
			service.PoolId = existingService.PoolId
			if existingPools[service.PoolId] == nil {
				glog.Infof("Changing PoolId of service %s from %s to default", service.Id, service.PoolId)
				service.PoolId = "default"
			}
			if e := d.client.UpdateService(*service, unused); e != nil {
				glog.Errorf("Could not update service %s: %v", service.Id, e)
				return e
			}
		} else {
			if existingPools[service.PoolId] == nil {
				glog.Infof("Changing PoolId of service %s from %s to default", service.Id, service.PoolId)
				service.PoolId = "default"
			}
			var serviceId string
			if e := d.client.AddService(*service, &serviceId); e != nil {
				glog.Errorf("Could not add service %s: %v", service.Id, e)
				return e
			}
			if service.Id != serviceId {
				msg := fmt.Sprintf("BUG!!! ADDED SERVICE %s, BUT WITH THE WRONG ID: %s", service.Id, serviceId)
				glog.Errorf(msg)
				return errors.New(msg)
			}
			existingServiceMap[service.Id] = service
		}
	}

	return nil
}

func (d *DistributedFileSystem) findImages(id, tag string) (images []docker.APIImages, err error) {
	if all, err := d.dockerClient.ListImages(false); err != nil {
		return images, err
	} else {
		for _, image := range all {
			for _, repotag := range image.RepoTags {
				// check if the tags match
				if !strings.HasSuffix(repotag, ":"+tag) {
					continue
				}

				// figure out the repo
				repo := strings.TrimSuffix(repotag, ":"+tag)

				// verify that the repo matches
				repoparts := strings.SplitN(repo, "/", 3)
				reponame := repoparts[len(repoparts)-1]
				if strings.HasPrefix(reponame, id+"_") {
					image.Repository = repo
					image.Tag = tag
					images = append(images, image)
					break
				}
			}
		}
	}

	return
}

func (d *DistributedFileSystem) tag(id, oldtag, newtag string) error {
	images, err := d.findImages(id, oldtag)
	if err != nil {
		return err
	}

	for i, image := range images {
		options := docker.TagImageOptions{
			Repo: image.Repository,
			Tag:  newtag,
		}

		glog.V(3).Infof("Adding tag to image %s: %+v", image.ID, options)
		if err := d.dockerClient.TagImage(image.ID, options); err != nil {
			glog.Errorf("error while adding tags, rolling back...")
			for j := 0; j < i; j++ {
				repotag := images[j].Repository + ":" + newtag
				if err := d.dockerClient.RemoveImage(repotag); err != nil {
					glog.Errorf("cannot untag image %s: (%s)", repotag, err)
				}
			}

			return err
		}
	}

	return nil
}

var osCreate = func(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

var writeJsonToFile = func(v interface{}, filename string) (err error) {
	file, e := osCreate(filename)
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

var osOpen = func(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

var readJsonFromFile = func(v interface{}, filename string) error {
	file, e := osOpen(filename)
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
