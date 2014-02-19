package snapshot

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
	"sort"
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
)

type DistributedFileSystem struct {
	client  dao.ControlPlane
	dclient *docker.Client
	lock    sync.Mutex
}

func (d *DistributedFileSystem) Lock() {
	d.lock.Lock()
}

func (d *DistributedFileSystem) Unlock() {
	d.lock.Unlock()
}

func (d *DistributedFileSystem) Pause(service *dao.Service) error {
	if service.Snapshot.Pause == "" {
		return nil
	}

	var states []*dao.ServiceState
	if err := d.client.GetServiceStates(service.Id, &states); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Pause service=%+v err=%s", service, err)
		return err
	}

	var state *dao.ServiceState
	for i, s := range states {
		glog.V(3).Infof("DEBUG states[%d]: service:%+v state:%+v", i, service, s)
		if s.DockerId != "" && s.ServiceId == service.Id {
			state = s
			break
		}
	}
	if state == nil {
		err := errors.New(fmt.Sprintf("unable to find docker id for service:%+v", service))
		glog.V(2).Infof("DistributedFileSystem.Pause service=%+v err=%s", service, err)
		return err
	}

	if output, err := runServiceCommand(service.Id, state.DockerId, service.Snapshot.Pause); err != nil {
		errmsg := fmt.Sprintf("output: %s, err: %s", output, err)
		glog.V(2).Infof("DistributedFileSystem.Pause service=%+v err=%s", service, err)
		return errors.New(errmsg)
	}

	return nil
}

func (d *DistributedFileSystem) Resume(service *dao.Service) error {
	if service.Snapshot.Resume == "" {
		return nil
	}

	var states []*dao.ServiceState
	if err := d.client.GetServiceStates(service.Id, &states); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Resume service=%+v err=%s", service, err)
		return err
	}

	var state *dao.ServiceState
	for i, s := range states {
		glog.V(3).Infof("DEBUG states[%d]: service:%+v state:%+v", i, service, s)
		if s.DockerId != "" && s.ServiceId == service.Id {
			state = s
			break
		}
	}
	if state == nil {
		err := errors.New(fmt.Sprintf("unable to find docker id for service:%+v", service))
		glog.V(2).Infof("DistributedFileSystem.Resume service=%+v err=%s", service, err)
		return err
	}

	if output, err := runServiceCommand(service.Id, state.DockerId, service.Snapshot.Pause); err != nil {
		errmsg := fmt.Sprintf("output: %s, err: %s", output, err)
		glog.V(2).Infof("DistributedFileSystem.Resume service=%+v err=%s", service, err)
		return errors.New(errmsg)
	}

	return nil
}

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
	if whoami, err := user.Current(); err != nil {
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
			if err := d.Pause(service); err != nil {
				glog.V(2).Infof("DistributedFileSystem.Snapshot service=%+v err=%s", serviceId, err)
				return err
			} else {
				defer d.Resume(service)
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

func (d *DistributedFileSystem) Commit(dockerId string, label *string) error {
	d.Lock()
	defer d.Unlock()

	// Get the container
	container, err := d.dclient.InspectContainer(dockerId)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	} else if container.State.Running {
		err := errors.New("cannot commit a running container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Get tag & repo information
	var (
		latestImages []*docker.APIImages
		image        *docker.APIImages
	)

	images, err := d.dclient.ListImages(true)
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}
	for _, i := range images {
		if DOCKER_LATEST == i.Tag {
			latestImages = append(latestImages, &i)
			if i.ID == container.Image {
				image = &i
			}
		}
	}
	if image == nil {
		err := errors.New("cannot commit a stale container")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Get the service id from the image (very expensive!)
	var (
		allservices []*dao.Service
		service     *dao.Service
		volume      volume.Volume
	)
	if err := d.client.GetServices(unused, &allservices); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	for _, s := range allservices {
		if s.ImageId == image.ID {
			// Get all the service states for that service id
			var states []*dao.ServiceState
			if err := d.client.GetServiceStates(s.Id, &states); err != nil {
				glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
				return err
			}
			// Do these states match my dockerId?
			for _, state := range states {
				if dockerId == state.DockerId {
					service = s
					break
				}
			}
			if service != nil {
				break
			}
		}
	}
	if service == nil {
		err := errors.New("could not map container to service id")
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Snapshot the DFS
	if err := d.Snapshot(service.Id, label); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit dockerId=%+v err=%s", dockerId, err)
		return err
	}

	// Commit the container to the image
	newImage, err := d.dclient.CommitContainer(docker.CommitContainerOptions{
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
	var tenantId string
	if err := d.client.GetTenantId(service.Id, &tenantId); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	if err := d.client.GetVolume(tenantId, &volume); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	snapshots, err := volume.Snapshots()
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}
	sort.Strings(snapshots)
	latestSnapshot := snapshots[len(snapshots)-1]
	config := path.Join(volume.Path(), latestSnapshot, DOCKER_IMAGEJSON)
	if data, err := json.Marshal(latestImages); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	} else if err := ioutil.WriteFile(config, data, 0644); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Commit container=%+v err=%s", dockerId, err)
		return err
	}

	return nil
}

func (d *DistributedFileSystem) Rollback(snapshotId string) error {
	d.Lock()
	defer d.Unlock()

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
	label := parts[1]
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

	var images []docker.APIImages
	config := path.Join(volume.Path(), snapshotId, DOCKER_IMAGEJSON)
	if data, err := ioutil.ReadFile(config); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	} else if err := json.Unmarshal(data, images); err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	// Check to see if all the images exist
	for _, image := range images {
		if _, err := d.dclient.InspectImage(image.ID); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
			return err
		}
	}

	d.client.StopService(tenantId, nil)
	// TODO: Wait for real event that confirms shutdown
	time.Sleep(time.Second * 5) // wait for shutdown

	// Retag with images from snapshot
	dockerBin, err := exec.LookPath("docker")
	if err != nil {
		glog.V(2).Infof("DistributedFileSystem.Rollback service=%+v err=%s", serviceId, err)
		return err
	}

	// TODO: Should we try to restore if tagging fails?
	for _, image := range images {
		// docker tag %{image.ID} %{image.Repo}:%{image.Tag}
		cmd := exec.Command(dockerBin, "tag", image.ID, fmt.Sprintf("%s:%s", image.Repository, image.Tag))
		if err := cmd.Run(); err != nil {
			glog.V(2).Infof("DistributedFileSystem.Rollback service=%s", serviceId, err)
			return err
		}
	}

	glog.V(2).Infof("performing rollback on %s to %s", tenantId, label)
	if err := volume.Rollback(snapshotId); err != nil {
		return err
	}
	var unusedStr string = ""

	return d.client.StartService(tenantId, &unusedStr)
}

func runServiceCommand(serviceId string, dockerId string, command string) ([]byte, error) {
	lxcAttach, err := exec.LookPath("lxc-attach")
	if err != nil {
		return []byte{}, err
	}
	cmd := exec.Command(lxcAttach, "-n", "dockerId", "-e", "--", "bin/bash", "-c", command)

	output, err := cmd.CombinedOutput()
	if err != nil {
		glog.Errorf("Error running command: `%s` for serviceId: %s out: %s err: %s", command, serviceId, output, err)
		return output, err
	}
	glog.V(0).Infof("Successfully ran command: `%s` for serviceId: %s out: %s", command, serviceId, output)
	return output, nil
}

func getSnapshotLabel(v *volume.Volume) string {
	format := "20060102-150405"
	loc := time.Now()
	utc := loc.UTC()
	return fmt.Sprintf("%s_%s", v.Name(), utc.Format(format))
}