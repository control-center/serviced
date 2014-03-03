package dfs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"

	docker "github.com/zenoss/docker-go"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"os/user"
	"path"
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
	Lock   *sync.Mutex = new(sync.Mutex)
	// stubs
	getCurrentUser = user.Current
)

var runServiceCommand = func(state *dao.ServiceState, command string) ([]byte, error) {
	lxcAttach, err := exec.LookPath("lxc-attach")
	if err != nil {
		return []byte{}, err
	}
	argv := []string{"-n", state.DockerId, "-e", "--", "/bin/bash", "-c", command}
	glog.V(0).Infof("ServiceId: %s, Command: `%s %s`", state.ServiceId, lxcAttach, argv)
	cmd := exec.Command(lxcAttach, argv...)

	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command: `%s` for serviceId: %s out: %s err: %s", command, state.ServiceId, output, err)
		return output, err
	}
	glog.V(0).Infof("Successfully ran command: `%s` for serviceId: %s out: %s", command, state.ServiceId, output)
	return output, nil
}

type DistributedFileSystem struct {
	client       dao.ControlPlane
	dockerClient docker.Client
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
func (d *DistributedFileSystem) Snapshot(serviceId string, label *string) error {
	var tenantId string
	if err := d.client.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	var service dao.Service
	if err := d.client.GetService(tenantId, &service); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
		return err
	}

	if whoami, err := getCurrentUser(); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
		return err
	} else if USER_ROOT != whoami.Username {
		glog.Warningf("Unable to pause/resume service - User is not %s - whoami:%+v", USER_ROOT, whoami)
	} else {

		var servicesList []*dao.Service
		if err := d.client.GetServices(unused, &servicesList); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
			return err
		}

		for _, service := range servicesList {
			if service.Snapshot.Pause == "" || service.Snapshot.Resume == "" {
				continue
			}

			var states []*dao.ServiceState
			if err := d.client.GetServiceStates(service.Id, &states); err != nil {
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v, err=%s", serviceId, err)
				return err
			}

			// Pause all running service states
			for i, state := range states {
				glog.V(3).Infof("DEBUG states[%d]: service:%+v state:%+v", i, serviceId, state.DockerId)
				if state.DockerId != "" {
					err := d.Pause(service, state)
					defer d.Resume(service, state) // resume service state when snapshot is done
					if err != nil {
						glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
						return err
					}
				}
			}
		}
	}

	// create a snapshot
	var (
		theVolume volume.Volume
	)
	if err := d.client.GetVolume(tenantId, &theVolume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
		return err
	} else {
		glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v volume=%+v", serviceId, theVolume)
		*label = getSnapshotLabel(&theVolume)
		if err := theVolume.Snapshot(*label); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
			return err
		}
	}

	glog.V(2).Infof("Successfully created snapshot for service Id:%s Name:%s Label:%s", service.Id, service.Name, *label)
	return nil
}

// Commits a container to docker image and updates the DFS
func (d *DistributedFileSystem) Commit(dockerId string, label *string) error {
	Lock.Lock()
	defer Lock.Unlock()

	// Get the container
	container, err := d.dockerClient.InspectContainer(dockerId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	} else if container.State.Running {
		err := errors.New("cannot commit a running container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Get the service id
	name := strings.Split(container.Name, "_")
	serviceId := name[0]

	// Get tag & repo information
	var (
		images []docker.Image
		image  *docker.Image
	)

	if err := d.getLatestImages(&images); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}
	for _, i := range images {
		if i.Id == container.Image {
			image = &i
			break
		}
	}
	if image == nil {
		err := errors.New("cannot commit a stale container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Commit the container to the image
	imageId, err := d.dockerClient.Commit(container.Id, container.Config.Image, DOCKER_LATEST, "", "", docker.Config{})
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	newImage, err := d.dockerClient.InspectImage(*imageId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	// Copy images to the DFS
	*image = docker.Image{
		RepoTags: image.RepoTags,
		Id:       newImage.Id,
	}

	// Get the path to the volume and write the images
	var (
		tenantId string
		volume   volume.Volume
	)

	if err := d.client.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	if err := d.client.GetVolume(tenantId, &volume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	config := path.Join(volume.Path(), DOCKER_IMAGEJSON)
	if data, err := json.Marshal(images); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	} else if err := ioutil.WriteFile(config, data, 0644); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	// Snapshot the DFS
	if err := d.Snapshot(serviceId, label); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	return nil
}

// Gets the images from docker and filters those marked as latest
func (d *DistributedFileSystem) getLatestImages(images *[]docker.Image) error {
	// Get all of the images from docker and find the ones tagged as the latest
	if allImages, err := d.dockerClient.ListImages(); err != nil {
		return err
	} else {
		for _, image := range allImages {
			if image.HasTag(DOCKER_LATEST) {
				*images = append(*images, image)
			}
		}
	}
	glog.V(3).Infof("Found %d images: %+v", len(*images), *images)
	return nil
}

// Gets the images from the snapshot; returns error if image does not exist in docker
func (d *DistributedFileSystem) getSnapshotImages(snapshotId string, volume *volume.Volume, images *[]docker.Image) error {
	config := path.Join(path.Dir(volume.Path()), snapshotId, DOCKER_IMAGEJSON)

	if data, err := ioutil.ReadFile(config); err != nil {
		return err
	} else if err := json.Unmarshal(data, &images); err != nil {
		return err
	}

	// Check if the images still exist
	for _, image := range *images {
		if _, err := d.dockerClient.InspectImage(image.Id); err != nil {
			return err
		}
	}
	glog.V(3).Infof("Found %d images: %+v", len(*images), *images)
	return nil
}

// Retags containers with the given snapshot
func (d *DistributedFileSystem) retag(images *[]docker.Image, volume *volume.Volume, force bool) error {

	// Set the tag of the new image
	for _, image := range *images {
		repo := fmt.Sprintf("%s:%s", image.Repository(), DOCKER_LATEST)
		if err := d.dockerClient.TagImage(image.Id, repo, force); err != nil {
			return err
		}
	}

	return nil
}

// Rolls back the DFS to a specified state and retags the images
func (d *DistributedFileSystem) Rollback(snapshotId string) error {
	Lock.Lock()
	defer Lock.Unlock()

	var (
		services []*dao.Service
		tenantId string
		volume   volume.Volume
	)

	parts := strings.Split(snapshotId, "_")
	if len(parts) != 2 {
		glog.V(2).Infof("DistributedFileSystem.Rollback malformed snapshot Id: %s", snapshotId)
		return errors.New("malformed snapshotId")
	}
	serviceId := parts[0]

	// Fail if any services have running instances
	if err := d.client.GetServices(unused, &services); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}
	for _, service := range services {
		var states []*dao.ServiceState
		if err := d.client.GetServiceStates(service.Id, &states); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
			return err
		}
		if numstates := len(states); numstates > 0 {
			err := errors.New(fmt.Sprintf("%s has %d running services. Stop all services before rolling back", service.Id, numstates))
			glog.V(2).Info("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
			return err
		}
	}

	if err := d.client.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	// Validate existence of images for this snapshot
	var service dao.Service
	err := d.client.GetService(tenantId, &service)
	glog.V(2).Infof("Getting service instance: %s", tenantId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	if err := d.client.GetVolume(tenantId, &volume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	var latestImages []docker.Image
	if err := d.getLatestImages(&latestImages); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	var snapshotImages []docker.Image
	if err := d.getSnapshotImages(snapshotId, &volume, &snapshotImages); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	if err := d.retag(&snapshotImages, &volume, false); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		if err := d.retag(&latestImages, &volume, true); err != nil {
			glog.Errorf("DistributedFileSystem.Rollback unable to restore images service=%+v err=%s", serviceId, err)
		}
		return err
	}

	glog.V(0).Infof("performing rollback on serviceId: %s to snaphotId: %s", tenantId, snapshotId)
	if err := volume.Rollback(snapshotId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		if err := d.retag(&latestImages, &volume, true); err != nil {
			glog.Errorf("DistributedFileSystem.Rollback unable to restore images service=%+v err=%s", serviceId, err)
		}
		return err
	}
	return nil
}

func getSnapshotLabel(v *volume.Volume) string {
	return serviced.GetLabel(v.Name())
}