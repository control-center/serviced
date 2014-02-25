package dfs

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/volume"

	docker "github.com/fsouza/go-dockerclient"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os/exec"
	"os/user"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	USER_ROOT        string = "root"
	DOCKER_ENDPOINT  string = "unix:///var/run/docker.sock"
	DOCKER_LATEST    string = "latest"
	DOCKER_IMAGEJSON string = "images.json"

	ERR_STATENOTFOUND string = "Service state not found for serviceId: %s"
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
	glog.V(3).Infof("Command: %s %s", lxcAttach, argv)
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

func (d *DistributedFileSystem) getServiceState(serviceId string, state *dao.ServiceState) error {
	var states []*dao.ServiceState
	if err := d.client.GetServiceStates(serviceId, &states); err != nil {
		return err
	}
	for i, s := range states {
		glog.V(3).Infof("DEBUG states[%d]: service:%+v state:%+v", i, serviceId, s)
		if s.DockerId != "" {
			*state = *s
			return nil
		}
	}
	return errors.New(fmt.Sprintf(ERR_STATENOTFOUND, serviceId))
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

	// simplest case - do everything here

	// call quiesce for services with 'DistributedFileSystem.Pause' and
	// 'DistributedFileSystem.Resume' definition.  Only root can run
	// lxc-attach
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

			var state dao.ServiceState
			if err := d.getServiceState(service.Id, &state); err != nil {
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
				return err
			}

			err := d.Pause(service, &state)
			defer d.Resume(service, &state)
			if err != nil {
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
				return err
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
		images []docker.APIImages
		image  *docker.APIImages
	)

	if err := d.getLatestImages(&images); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}
	for _, i := range images {
		if i.ID == container.Image {
			image = &i
			break
		}
	}
	if image == nil {
		err := errors.New("cannot commit a stale container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Snapshot the DFS
	if err := d.Snapshot(serviceId, label); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Commit the container to the image
	newImage, err := d.dockerClient.CommitContainer(docker.CommitContainerOptions{
		Container:  container.ID,
		Repository: image.Repository,
		Tag:        image.Tag,
	})
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	// Copy images to the DFS
	*image = docker.APIImages{
		ID:          newImage.ID,
		RepoTags:    image.RepoTags,
		Created:     newImage.Created.Unix(),
		Size:        newImage.Size,
		VirtualSize: image.VirtualSize,
		ParentId:    newImage.Parent,
		Repository:  image.Repository,
		Tag:         image.Tag,
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

	config := path.Join(volume.Path(), *label, DOCKER_IMAGEJSON)
	if data, err := json.Marshal(images); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	} else if err := ioutil.WriteFile(config, data, 0644); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	return nil
}

func (d *DistributedFileSystem) getLatestImages(images *[]docker.APIImages) error {
	// Get all of the images from docker and find the ones tagged as the latest
	if allImages, err := d.dockerClient.ListImages(true); err != nil {
		return err
	} else {
		for _, image := range allImages {
			if image.Tag == DOCKER_LATEST {
				*images = append(*images, image)
			}
		}
	}

	return nil
}

// Gets the images from the snapshot; returns error if image does not exist in docker
func (d *DistributedFileSystem) getSnapshotImages(snapshotId string, volume *volume.Volume, images *[]docker.APIImages) error {
	config := path.Join(volume.Path(), snapshotId, DOCKER_IMAGEJSON)

	if data, err := ioutil.ReadFile(config); err != nil {
		return err
	} else if err := json.Unmarshal(data, images); err != nil {
		return err
	}

	// Check if the images still exist
	for _, image := range *images {
		if _, err := d.dockerClient.InspectImage(image.ID); err != nil {
			return err
		}
	}

	return nil
}

// Retags containers with the given snapshot
func (d *DistributedFileSystem) retag(images *[]docker.APIImages, volume *volume.Volume) error {

	// Set the tag of the new image
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		return err
	}

	for _, image := range *images {
		// docker tag %{image.ID} %{image.Repo}:%{image.Tag}
		cmd := exec.Command(dockerBin, "tag", image.ID, fmt.Sprintf("%s:%s", image.Repository, image.Tag))
		if err := cmd.Run(); err != nil {
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
		tenantId string
		volume   volume.Volume
	)

	parts := strings.Split(snapshotId, "_")
	if len(parts) != 2 {
		glog.V(2).Infof("DistributedFileSystem.Rollback malformed snapshot Id: %s", snapshotId)
		return errors.New("malformed snapshotId")
	}
	serviceId := parts[0]
	if err := d.client.GetTenantId(serviceId, &tenantId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	// Validate existance of images for this snapshot
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

	var latestImages []docker.APIImages
	if err := d.getLatestImages(&latestImages); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	var snapshotImages []docker.APIImages
	if err := d.getSnapshotImages(snapshotId, &volume, &snapshotImages); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	d.client.StopService(tenantId, nil)
	// TODO: Wait for real event that confirms shutdown
	time.Sleep(time.Second * 5) // wait for shutdown

	if err := d.retag(&snapshotImages, &volume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		if err := d.retag(&latestImages, &volume); err != nil {
			glog.Errorf("DistributedFileSystem.Rollback unable to restore images service=%+v err=%s", serviceId, err)
		}
		return err
	}

	glog.V(2).Infof("performing rollback on %s to %s", tenantId, snapshotId)
	if err := volume.Rollback(snapshotId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		if err := d.retag(&latestImages, &volume); err != nil {
			glog.Errorf("DistributedFileSystem.Rollback unable to restore images service=%+v err=%s", serviceId, err)
		}
		return err
	}
	var unusedStr string = ""

	return d.client.StartService(tenantId, &unusedStr)
}

func getSnapshotLabel(v *volume.Volume) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return fmt.Sprintf("%s_%s", v.Name(), utc.Format(format))
}