// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/zenoss/glog"
	docker "github.com/zenoss/go-dockerclient"
	"github.com/zenoss/serviced/dao"
	"github.com/zenoss/serviced/domain/service"
	"github.com/zenoss/serviced/domain/servicedefinition"
	"github.com/zenoss/serviced/domain/servicetemplate"

	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var commandAsRoot = func(name string, arg ...string) (*exec.Cmd, error) {
	user, e := user.Current()
	if e != nil {
		return nil, e
	}
	if user.Uid == "0" {
		return exec.Command(name, arg...), nil
	}
	cmd := exec.Command("sudo", "-n", "echo")
	if output, err := cmd.CombinedOutput(); err != nil {
		glog.Errorf("Unable to run as root cmd:%+v  error:%v  output:%s", cmd, err, string(output))
		return nil, err
	}
	return exec.Command("sudo", append([]string{"-n", name}, arg...)...), nil //Go, you make me sad.
}

var writeDirectoryToTgz = func(src, filename string) error {
	//FIXME: Tar file should put all contents below a sub-directory (rather than directly in current directory).
	cmd, e := commandAsRoot("tar", "-czf", filename, "-C", src, ".")
	if e != nil {
		return e
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		glog.Errorf("Unable to writeDirectoryToTgz cmd:%+v  error:%v  output:%s", cmd, err, string(output))
		return err
	}
	return nil
}

var writeDirectoryFromTgz = func(dest, filename string) (err error) {
	if _, e := osStat(dest); e != nil {
		if !os.IsNotExist(e) {
			glog.Errorf("Could not stat %s: %v", dest, e)
			return e
		}
		if e := osMkdirAll(dest, os.ModeDir|0755); e != nil {
			glog.Errorf("Could not find nor create %s: %v", dest, e)
			return e
		}
		defer func() {
			if err != nil {
				if e := osRemoveAll(dest); e != nil {
					glog.Errorf("Could not remove %s: %v", dest, e)
				}
			}
		}()
	}
	cmd, e := commandAsRoot("tar", "-xpUf", filename, "-C", dest, "--numeric-owner")
	if e != nil {
		return e
	}
	if output, err := cmd.CombinedOutput(); err != nil {
		glog.Errorf("Unable to writeDirectoryToTgz cmd:%+v  error:%v  output:%s", cmd, err, string(output))
		return err
	}
	return nil
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

var newDockerImporter = func() (dockerImporter, error) {
	return docker.NewClient(DOCKER_ENDPOINT)
}

var newDockerExporter = func() (dockerExporter, error) {
	return docker.NewClient(DOCKER_ENDPOINT)
}

type dockerExporter interface {
	CreateContainer(docker.CreateContainerOptions) (*docker.Container, error)
	RemoveContainer(docker.RemoveContainerOptions) error
	ExportContainer(docker.ExportContainerOptions) error
	ListImages(bool) ([]docker.APIImages, error)
}

type dockerImporter interface {
	ImportImage(docker.ImportImageOptions) error
	InspectImage(string) (*docker.Image, error)
	TagImage(string, docker.TagImageOptions) error
}

var getDockerImageNameIds = func(client dockerExporter) (map[string]string, error) {
	images, e := client.ListImages(true)
	if e != nil {
		return nil, e
	}
	result := make(map[string]string)
	for _, image := range images {
		result[image.ID] = image.ID
		for _, repotag := range image.RepoTags {
			repo, tag := repoAndTag(repotag)
			if tag == "" || tag == "latest" {
				result[repo] = image.ID
			} else {
				result[repotag] = image.ID
			}
		}
	}
	return result, nil
}

var exportDockerImageToFile = func(client dockerExporter, imageId, filename string) (err error) {
	file, e := osCreate(filename)
	if e != nil {
		glog.Errorf("Could not create file %s: %v", filename, e)
		return e
	}

	// Close (and perhaps delete) file on the way out
	defer func() {
		if e := file.Close(); e != nil {
			glog.Errorf("Error while closing file %s: %v", filename, e)
			if err == nil {
				err = e
			}
		}
		if err != nil && file != nil {
			if e := osRemoveAll(filename); e != nil {
				glog.Errorf("Error while removing file %s: %v", filename, e)
			}
		}
	}()

	createOpts := docker.CreateContainerOptions{
		Config: &docker.Config{
			Cmd:   []string{"echo ''"},
			Image: imageId,
		},
	}

	container, e := client.CreateContainer(createOpts)
	if e != nil {
		glog.Errorf("Could not create container from image %s: %v", imageId, e)
		return e
	}

	glog.Infof("Created container %s based on image %s", container.ID, imageId)

	// Remove container on the way out
	defer func() {
		removeOpts := docker.RemoveContainerOptions{ID: container.ID}
		if e := client.RemoveContainer(removeOpts); e != nil {
			glog.Errorf("Could not remove container %s: %v", container.ID, e)
			if err == nil {
				err = e
			}
		} else {
			glog.Infof("Removed container %s", container.ID)
		}
	}()

	exportOpts := docker.ExportContainerOptions{
		ID:           container.ID,
		OutputStream: file,
	}

	if e = client.ExportContainer(exportOpts); e != nil {
		glog.Errorf("Could not export container %s: %v", container.ID, e)
		return e
	}

	glog.Infof("Exported container %s (based on image %s) to %s", container.ID, imageId, filename)
	return nil
}

var repoAndTag = func(imageId string) (string, string) {
	i := strings.LastIndex(imageId, ":")
	if i < 0 {
		return imageId, ""
	}
	tag := imageId[i+1:]
	if strings.Contains(tag, "/") {
		return imageId, ""
	}
	return imageId[:i], tag
}

var importDockerImageFromFile = func(client dockerImporter, imageId, filename string) (err error) {
	file, e := os.Open(filename)
	if e != nil {
		return e
	}
	defer file.Close()
	repo, tag := repoAndTag(imageId)
	importOpts := docker.ImportImageOptions{
		Repository:  repo,
		Source:      "-",
		InputStream: file,
		Tag:         tag,
	}
	if e = client.ImportImage(importOpts); e != nil {
		return e
	}
	return nil
}

var utcNow = func() time.Time {
	return time.Now().UTC()
}

// Find all docker images referenced by a template or service
var dockerImageSet = func(templates map[string]*servicetemplate.ServiceTemplate, services []*service.Service) map[string]bool {
	imageSet := make(map[string]bool)
	var visit func(*[]servicedefinition.ServiceDefinition)
	visit = func(defs *[]servicedefinition.ServiceDefinition) {
		for _, serviceDefinition := range *defs {
			if serviceDefinition.ImageID != "" {
				imageSet[serviceDefinition.ImageID] = true
			}
			visit(&serviceDefinition.Services)
		}
	}
	for _, template := range templates {
		visit(&template.Services)
	}
	for _, service := range services {
		if service.ImageId != "" {
			imageSet[service.ImageId] = true
		}
	}
	return imageSet
}

func (this *ControlPlaneDao) Backup(backupsDirectory string, backupFilePath *string) (err error) {
	var (
		templates      map[string]*servicetemplate.ServiceTemplate
		services       []*service.Service
		imagesNameTags [][]string
	)
	backupName := utcNow().Format("backup-2006-01-02-150405")
	if backupsDirectory == "" {
		backupsDirectory = filepath.Join(varPath(), "backups")
	}
	*backupFilePath = path.Join(backupsDirectory, backupName+".tgz")
	defer func() {
		// Zero-value the backupFilePath if we're returning an error
		if err != nil && backupFilePath != nil && *backupFilePath != "" {
			*backupFilePath = ""
		}
	}()
	backupPath := func(relPath ...string) string {
		return filepath.Join(append([]string{backupsDirectory, backupName}, relPath...)...)
	}
	if e := osMkdirAll(backupPath("images"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		return e
	}
	defer func() {
		if e := osRemoveAll(backupPath()); e != nil {
			glog.Errorf("Could not remove %s: %v", backupPath(), e)
			if err == nil {
				err = e
			}
		}
	}()
	if e := osMkdirAll(backupPath("snapshots"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		return e
	}

	// Retrieve all service definitions
	var request dao.EntityRequest
	if e := this.GetServices(request, &services); e != nil {
		glog.Errorf("Could not get services: %v", e)
		return e
	}

	// Dump all template definitions
	if e := this.GetServiceTemplates(0, &templates); e != nil {
		glog.Errorf("Could not get templates: %v", e)
		return e
	}
	if e := writeJsonToFile(templates, backupPath("templates.json")); e != nil {
		glog.Errorf("Could not write templates.json: %v", e)
		return e
	}

	// Export each of the referenced docker images
	client, e := newDockerExporter()
	if e != nil {
		glog.Errorf("Could not connect to docker: %v", e)
		return e
	}
	// Note: client does not need to be .Close()'d

	imageNameIds, e := getDockerImageNameIds(client)
	if e != nil {
		glog.Errorf("Could not get image tags from docker: %v", e)
		return e
	}

	imageIdTags := make(map[string][]string)

	imageNameSet := dockerImageSet(templates, services)

	for imageName, _ := range imageNameSet {
		imageId := imageNameIds[imageName]
		imageIdTags[imageId] = []string{}
	}

	for imageName, imageId := range imageNameIds {
		if imageName == imageId {
			continue
		}
		tags := imageIdTags[imageId]
		if tags == nil {
			continue
		}
		imageIdTags[imageId] = append(tags, imageName)
	}

	i := 0
	for imageId, imageTags := range imageIdTags {
		filename := backupPath("images", fmt.Sprintf("%d.tar", i))
		if e := exportDockerImageToFile(client, imageId, filename); e != nil {
			if e == docker.ErrNoSuchImage {
				glog.Infof("Docker image %s was referenced, but does not exist. Ignoring.", imageId)
			} else {
				glog.Errorf("Error while exporting docker image %s: %v", imageId, e)
				return e
			}
		} else {
			imageNameWithTags := append([]string{imageId}, imageTags...)
			imagesNameTags = append(imagesNameTags, imageNameWithTags)
			i++
		}
	}

	if e := writeJsonToFile(imagesNameTags, backupPath("images.json")); e != nil {
		glog.Errorf("Could not write images.json: %v", e)
		return e
	}

	// Dump all snapshots
	snapshotToTgzFile := func(service *service.Service) (filename string, err error) {
		glog.V(0).Infof("snapshotToTgzFile(%v)", service.Id)
		var snapshotId string
		if e := this.Snapshot(service.Id, &snapshotId); e != nil {
			glog.Errorf("Could not snapshot service %s: %v", service.Id, e)
			return "", e
		}

		// Delete snapshot on the way out
		defer func() {
			var unused int
			if e := this.DeleteSnapshot(snapshotId, &unused); e != nil {
				glog.Errorf("Error while deleting snapshot %s: %v", snapshotId, e)
				if err == nil {
					err = e
				}
			}
		}()
		snapDir, e := getSnapshotPath(this.vfs, service.PoolId, service.Id, snapshotId)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolId, service.Id, e)
			return "", e
		}
		snapFile := backupPath("snapshots", fmt.Sprintf("%s.tgz", snapshotId))
		if e := writeDirectoryToTgz(snapDir, snapFile); e != nil {
			glog.Errorf("Could not write %s to %s: %v", snapDir, snapFile, e)
			return "", e
		}

		glog.V(2).Infof("Saved snapshot of service:%v from dir:%v to snapFile:%v", service.Id, snapDir, snapFile)
		return snapFile, nil
	}

	glog.Infof("Snapshot all top level services (count:%d)", len(services))
	for _, service := range services {
		if service.ParentServiceId == "" {
			if _, e := snapshotToTgzFile(service); e != nil {
				glog.Errorf("Could not save snapshot of service %s: %v", service.Id, e)
				return e
			}
			// Note: the deferred RemoveAll (above) will cleanup the file.
		}
	}

	if e := writeDirectoryToTgz(backupPath(), *backupFilePath); e != nil {
		glog.Errorf("Could not write %s to %s: %v", backupPath(), backupFilePath, e)
		return e
	}

	glog.Infof("Created backup from dir:%s to file:%s", backupPath(), backupFilePath)
	return nil
}

var getSnapshotPath = func(vfs, poolId, serviceId, snapshotId string) (string, error) {
	volume, e := getSubvolume(vfs, poolId, serviceId)
	if e != nil {
		return "", e
	}
	return volume.SnapshotPath(snapshotId), nil
}

func (this *ControlPlaneDao) Restore(backupFilePath string, unused *int) (err error) {
	//TODO: acquire restore mutex, defer release
	var (
		doReloadLogstashContainer bool
		templates                 map[string]*servicetemplate.ServiceTemplate
		imagesNameTags            [][]string
	)
	defer func() {
		if doReloadLogstashContainer {
			go this.reloadLogstashContainer() // don't block the main thread
		}
	}()
	restorePath := func(relPath ...string) string {
		return filepath.Join(append([]string{varPath(), "restore"}, relPath...)...)
	}

	if e := osRemoveAll(restorePath()); e != nil {
		glog.Errorf("Could not remove %s: %v", restorePath(), e)
		return e
	}

	if e := osMkdirAll(restorePath(), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", restorePath(), e)
		return e
	}

	glog.Infof("HERE IS YOUR DIRECTORY: %v", restorePath())

	defer func() {
		if e := osRemoveAll(restorePath()); e != nil {
			glog.Errorf("Could not remove %s: %v", restorePath(), e)
			if err == nil {
				err = e
			}
		}
	}()

	if e := writeDirectoryFromTgz(restorePath(), backupFilePath); e != nil {
		glog.Errorf("Could not expand %s to %s: %v", backupFilePath, restorePath(), e)
		return e
	}

	if e := readJsonFromFile(&templates, restorePath("templates.json")); e != nil {
		glog.Errorf("Could not read templates from %s: %v", restorePath("templates.json"), e)
		return e
	}

	if e := readJsonFromFile(&imagesNameTags, restorePath("images.json")); e != nil {
		glog.Errorf("Could not read images from %s: %v", restorePath("images.json"), e)
		return e
	}

	// Restore the service templates ...
	for templateId, template := range templates {
		template.ID = templateId
		if e := this.UpdateServiceTemplate(*template, unused); e != nil {
			glog.Errorf("Could not update template %s: %v", templateId, e)
			return e
		}
		doReloadLogstashContainer = true
	}

	// Restore the docker images ...
	client, e := newDockerImporter()
	// Note: client does not need to be .Close()'d
	if e != nil {
		glog.Errorf("Could not connect to docker: %v", e)
		return e
	}
	for i, imageNameWithTags := range imagesNameTags {
		imageId := imageNameWithTags[0]
		imageTags := imageNameWithTags[1:]
		imageName := "imported:" + imageId
		image, e := client.InspectImage(imageId)
		if e != nil {
			if e != docker.ErrNoSuchImage {
				glog.Errorf("Unexpected error when inspecting docker image %s: %v", imageId, e)
				return e
			}
			filename := restorePath("images", fmt.Sprintf("%d.tar", i))
			if e := importDockerImageFromFile(client, imageName, filename); e != nil {
				glog.Errorf("Could not import docker image %s (%+v) from file %s: %v", imageId, imageTags, filename, e)
				return e
			}
			image, e = client.InspectImage(imageName)
			if e != nil {
				glog.Errorf("Could not find imported docker image %s (%+v): %v", imageName, imageTags, e)
				return e
			}
		}

		for _, imageTag := range imageTags {
			repo, tag := repoAndTag(imageTag)
			options := docker.TagImageOptions{
				Repo:  repo,
				Tag:   tag,
				Force: true,
			}
			if e := client.TagImage(image.ID, options); e != nil {
				glog.Errorf("Could not tag image %s (%s) options: %+v: %v", image.ID, imageName, options, e)
				return e
			}
		}
	}

	// Restore the snapshots ...
	snapFiles, e := readDirFileNames(restorePath("snapshots"))
	if e != nil {
		glog.Errorf("Could not list contents of %s: %v", restorePath("snapshots"), e)
		return e
	}
	for _, snapFile := range snapFiles {
		snapshotId := strings.TrimSuffix(snapFile, ".tgz")
		if snapshotId == snapFile {
			continue //the filename does not end with .tgz
		}
		parts := strings.Split(snapshotId, "_")
		if len(parts) != 2 {
			glog.Warningf("Skipping restoration of snapshot %s, due to malformed ID!", snapshotId)
			continue
		}
		serviceId := parts[0]

		snapFilePath := restorePath("snapshots", snapFile)
		snapDirTemp := restorePath("snapshots", snapshotId)
		if e := writeDirectoryFromTgz(snapDirTemp, snapFilePath); e != nil {
			glog.Errorf("Could not write %s from %s: %v", snapDirTemp, snapFilePath, e)
			return e
		}

		servicesPath := restorePath("snapshots", snapshotId, "services.json")
		var services []*service.Service
		if e := readJsonFromFile(&services, servicesPath); e != nil {
			glog.Errorf("Could not read services from %s: %v", servicesPath, e)
			continue
		}

		var service *service.Service = nil
		for _, svc := range services {
			if serviceId == svc.Id {
				service = svc
				break
			}
		}
		if service == nil {
			glog.Warningf("Could not find service %s for snapshot %s. Skipping!", serviceId, snapshotId)
			continue
		}

		snapDir, e := getSnapshotPath(this.vfs, service.PoolId, service.Id, snapshotId)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolId, service.Id, e)
			return e
		}

		// TODO: mv snapDirTemp to snapDir instead of restoring a 2nd time
		glog.Errorf("TODO: mv snapDirTemp:%s to snapDir:%s instead of duplicate untar", snapDirTemp, snapDir)
		if e := writeDirectoryFromTgz(snapDir, snapFilePath); e != nil {
			glog.Errorf("Could not write %s from %s: %v", snapDir, snapFilePath, e)
			return e
		}

		defer func() {
			var unused int
			if e := this.DeleteSnapshot(snapshotId, &unused); e != nil {
				glog.Errorf("Couldn't delete snapshot %s: %v", snapshotId, e)
				if err == nil {
					err = e
				}
			}
		}()

		if e := this.Rollback(snapshotId, unused); e != nil {
			glog.Errorf("Could not rollback to snapshot %s: %v", snapshotId, e)
			return e
		}
	}

	//TODO: garbage collect (http://jimhoskins.com/2013/07/27/remove-untagged-docker-images.html)
	return nil
}

var readDirFileNames = func(dirname string) ([]string, error) {
	files, e := ioutil.ReadDir(dirname)
	result := make([]string, len(files))
	if e != nil {
		return result, e
	}
	for i, file := range files {
		result[i] = file.Name()
	}
	return result, nil
}

var osOpen = func(name string) (io.ReadCloser, error) {
	return os.Open(name)
}

var osCreate = func(name string) (io.WriteCloser, error) {
	return os.Create(name)
}

var osStat = func(name string) (os.FileInfo, error) {
	return os.Stat(name)
}

var osMkdirAll = func(path string, perm os.FileMode) error {
	return os.MkdirAll(path, perm)
}

var osRemoveAll = func(path string) error {
	return os.RemoveAll(path)
}
