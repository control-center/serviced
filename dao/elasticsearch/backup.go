// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/facade"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"

	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"strings"
	"time"
)

var backupOutput chan string = nil
var backupError chan string = nil
var restoreOutput chan string = nil
var restoreError chan string = nil

func commandAsRoot(name string, arg ...string) (*exec.Cmd, error) {
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

func writeDirectoryToTgz(src, filename string) error {
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

func writeDirectoryFromTgz(dest, filename string) (err error) {
	if _, e := os.Stat(dest); e != nil {
		if !os.IsNotExist(e) {
			glog.Errorf("Could not stat %s: %v", dest, e)
			return e
		}
		if e := os.MkdirAll(dest, os.ModeDir|0755); e != nil {
			glog.Errorf("Could not find nor create %s: %v", dest, e)
			return e
		}
		defer func() {
			if err != nil {
				if e := os.RemoveAll(dest); e != nil {
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

func writeJSONToFile(v interface{}, filename string) (err error) {
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

func readJSONFromFile(v interface{}, filename string) error {
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

func getDockerImageNameIds() (map[string]string, error) {
	images, e := docker.Images()
	if e != nil {
		return nil, e
	}
	result := make(map[string]string)
	for _, image := range images {
		result[image.UUID] = image.UUID
		switch image.ID.Tag {
		case "", "latest":
			result[image.ID.BaseName()] = image.UUID
		default:
			result[image.ID.String()] = image.UUID
		}
	}
	return result, nil
}

func exportDockerImageToFile(imageID, filename string) error {
	file, err := os.Create(filename)
	if err != nil {
		glog.Errorf("Could not create file %s: %v", filename, err)
		return err
	}

	// Close (and perhaps delete) file on the way out
	defer func() {
		if e := file.Close(); e != nil {
			glog.Errorf("Error while closing file %s: %v", filename, e)
		}
		if err != nil && file != nil {
			if e := os.RemoveAll(filename); e != nil {
				glog.Errorf("Error while removing file %s: %v", filename, e)
			}
		}
	}()

	cd := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Config: &dockerclient.Config{
				Cmd:   []string{"echo ''"},
				Image: imageID,
			},
		},
		dockerclient.HostConfig{},
	}

	container, e := docker.NewContainer(cd, false, 600*time.Second, nil, nil)
	if e != nil {
		glog.Errorf("Could not create container from image %s: %v", imageID, e)
		return e
	}

	glog.Infof("Created container %s based on image %s", container.ID, imageID)

	// Remove container on the way out
	defer func() {
		if e := container.Kill(); e != nil {
			glog.Errorf("Could not remove container %s: %v", container.ID, e)
		} else {
			glog.Infof("Removed container %s", container.ID)
		}
	}()

	if err = container.Export(file); err != nil {
		glog.Errorf("Could not export container %s: %v", container.ID, err)
		return err
	}

	glog.Infof("Exported container %s (based on image %s) to %s", container.ID, imageID, filename)
	return nil
}

func repoAndTag(imageID string) (string, string) {
	i := strings.LastIndex(imageID, ":")
	if i < 0 {
		return imageID, ""
	}
	tag := imageID[i+1:]
	if strings.Contains(tag, "/") {
		return imageID, ""
	}
	return imageID[:i], tag
}

func importDockerImageFromFile(imageID, filename string) error {
	if err := docker.ImportImage(imageID, filename); err != nil {
		return err
	}
	return nil
}

func utcNow() time.Time {
	return time.Now().UTC()
}

// Find all docker images referenced by a template or service
func dockerImageSet(templates map[string]*servicetemplate.ServiceTemplate, services []*service.Service) map[string]bool {
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
		if service.ImageID != "" {
			imageSet[service.ImageID] = true
		}
	}
	return imageSet
}

func (this *ControlPlaneDao) AsyncBackup(backupsDirectory string, backupFilePath *string) (err error) {
	go func() {
		this.Backup(backupsDirectory, backupFilePath)
	}()

	return nil
}

func (this *ControlPlaneDao) BackupStatus(notUsed string, backupStatus *string) (err error) {
	select {
	case *backupStatus = <-backupOutput:
	case <-time.After(10 * time.Second):
		*backupStatus = "timeout"
	case *backupStatus = <-backupError:
		err = errors.New(*backupStatus)
		return err
	}

	return nil
}

// Backup saves the service templates, services, and related docker images and shared filesystems to a tgz file.
func (cp *ControlPlaneDao) Backup(backupsDirectory string, backupFilePath *string) (err error) {
	// Lock the error and output channels to ensure that only one backup runs at any given time.
	// Done in an anonymous function so we can ensure unlocking of the channel when we are done.
	err = func() error {
		cp.backupLock.Lock()

		//ensure that the backupLock is unlocked after this function exits
		defer cp.backupLock.Unlock()

		backupError = make(chan string, 100)

		//open a channel for asynchronous Backup calls
		if backupOutput != nil {
			e := errors.New("Another backup is currently in progress")
			glog.Errorf("An error occured when starting backup: %v", e)
			backupError <- e.Error()
			return e
		}
		backupOutput = make(chan string, 100)
		return nil
	}()

	defer func() {
		//close the channel for asynchronous calls to Backup
		close(backupOutput)
		backupOutput = nil
	}()

	// check the error status of the channel creation if there was an error, return it now.
	if err != nil {
		return err
	}

	backupOutput <- "Starting backup"

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
	if e := os.MkdirAll(backupPath("images"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		backupError <- e.Error()
		return e
	}
	defer func() {
		if e := os.RemoveAll(backupPath()); e != nil {
			glog.Errorf("Could not remove %s: %v", backupPath(), e)
			if err == nil {
				err = e
			}
		}
	}()
	if e := os.MkdirAll(backupPath("snapshots"), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", backupPath(), e)
		backupError <- e.Error()
		return e
	}

	// Retrieve all service definitions
	var request dao.EntityRequest
	if e := cp.GetServices(request, &services); e != nil {
		glog.Errorf("Could not get services: %v", e)
		backupError <- e.Error()
		return e
	}

	// Dump all template definitions
	if e := cp.GetServiceTemplates(0, &templates); e != nil {
		glog.Errorf("Could not get templates: %v", e)
		backupError <- e.Error()
		return e
	}
	if e := writeJSONToFile(templates, backupPath("templates.json")); e != nil {
		glog.Errorf("Could not write templates.json: %v", e)
		backupError <- e.Error()
		return e
	}

	// Export each of the referenced docker images
	imageNameIds, e := getDockerImageNameIds()
	if e != nil {
		glog.Errorf("Could not get image tags from docker: %v", e)
		backupError <- e.Error()
		return e
	}

	imageIDTags := make(map[string][]string)

	imageNameSet := dockerImageSet(templates, services)

	for imageName := range imageNameSet {
		imageID := imageNameIds[imageName]
		imageIDTags[imageID] = []string{}
	}

	for imageName, imageID := range imageNameIds {
		if imageName == imageID {
			continue
		}
		tags := imageIDTags[imageID]
		if tags == nil {
			continue
		}
		imageIDTags[imageID] = append(tags, imageName)
	}

	i := 0
	for imageID, imageTags := range imageIDTags {
		filename := backupPath("images", fmt.Sprintf("%d.tar", i))
		backupOutput <- fmt.Sprintf("Exporting docker image: %v", imageID)
		if e := exportDockerImageToFile(imageID, filename); e != nil {
			if e == dockerclient.ErrNoSuchImage {
				glog.Infof("Docker image %s was referenced, but does not exist. Ignoring.", imageID)
			} else {
				glog.Errorf("Error while exporting docker image %s: %v", imageID, e)
				backupError <- e.Error()
				return e
			}
		} else {
			imageNameWithTags := append([]string{imageID}, imageTags...)
			imagesNameTags = append(imagesNameTags, imageNameWithTags)
			i++
		}
	}

	if e := writeJSONToFile(imagesNameTags, backupPath("images.json")); e != nil {
		glog.Errorf("Could not write images.json: %v", e)
		backupError <- e.Error()
		return e
	}

	// Dump all snapshots
	snapshotToTgzFile := func(service *service.Service) (filename string, err error) {
		glog.V(0).Infof("snapshotToTgzFile(%v)", service.ID)
		backupOutput <- fmt.Sprintf("Taking snapshot of service: %v", service.Name)
		var snapshotID string
		if e := cp.Snapshot(service.ID, &snapshotID); e != nil {
			glog.Errorf("Could not snapshot service %s: %v", service.ID, e)
			backupError <- e.Error()
			return "", e
		}

		// Delete snapshot on the way out
		defer func() {
			var unused int
			if e := cp.DeleteSnapshot(snapshotID, &unused); e != nil {
				glog.Errorf("Error while deleting snapshot %s: %v", snapshotID, e)
				if err == nil {
					err = e
				}
			}
		}()
		snapDir, e := getSnapshotPath(cp.vfs, service.PoolID, service.ID, snapshotID)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolID, service.ID, e)
			backupError <- e.Error()
			return "", e
		}
		snapFile := backupPath("snapshots", fmt.Sprintf("%s.tgz", snapshotID))
		if e := writeDirectoryToTgz(snapDir, snapFile); e != nil {
			glog.Errorf("Could not write %s to %s: %v", snapDir, snapFile, e)
			backupError <- e.Error()
			return "", e
		}

		glog.V(2).Infof("Saved snapshot of service:%v from dir:%v to snapFile:%v", service.ID, snapDir, snapFile)
		return snapFile, nil
	}

	glog.Infof("Snapshot all top level services (count:%d)", len(services))

	for _, svc := range services {
		// Make sure you back up the service with desired state as stopped
		svc.DesiredState = service.SVCStop

		if svc.ParentServiceID == "" {
			if _, e := snapshotToTgzFile(svc); e != nil {
				glog.Errorf("Could not save snapshot of service %s: %v", svc.ID, e)
				backupError <- e.Error()
				return e
			}
			// Note: the deferred RemoveAll (above) will cleanup the file.
		}
	}

	if e := writeDirectoryToTgz(backupPath(), *backupFilePath); e != nil {
		glog.Errorf("Could not write %s to %s: %v", backupPath(), *backupFilePath, e)
		backupError <- e.Error()
		return e
	}

	glog.Infof("Created backup from dir:%s to file:%s", backupPath(), *backupFilePath)
	return nil
}

func getSnapshotPath(vfs, poolId, serviceID, snapshotID string) (string, error) {
	volume, e := getSubvolume(vfs, poolId, serviceID)
	if e != nil {
		return "", e
	}
	return volume.SnapshotPath(snapshotID), nil
}

func (this *ControlPlaneDao) AsyncRestore(backupFilePath string, unused *int) (err error) {
	go func() {
		this.Restore(backupFilePath, unused)
	}()

	return nil
}

func (this *ControlPlaneDao) RestoreStatus(notUsed string, restoreStatus *string) (err error) {
	select {
	case *restoreStatus = <-restoreOutput:
	case <-time.After(10 * time.Second):
		*restoreStatus = "timeout"
	case *restoreStatus = <-restoreError:
		err = errors.New(*restoreStatus)
		return err
	}

	return nil
}

// Restore replaces or restores the service templates, services, and related
// docker images and shared file systmes, as extracted from a tgz backup file.
func (cp *ControlPlaneDao) Restore(backupFilePath string, unused *int) (err error) {
	// Lock the error and output channels to ensure that only one restore runs at any given time.
	// Done in an anonymous function so we can ensure unlocking of the channel when we are done.
	err = func() error {
		cp.restoreLock.Lock()

		//ensure that restoreLock is unlocked when this function exits
		defer cp.restoreLock.Unlock()
		restoreError = make(chan string, 100)

		if restoreOutput != nil {
			e := errors.New("Another restore is currently in progress")
			glog.Errorf("An error occured when starting restore: %v", e)
			restoreError <- e.Error()
			return e
		}
		restoreOutput = make(chan string, 100)
		return nil
	}()

	defer func() {
		//close the channel for asynchronous calls to Backup
		close(restoreOutput)
		restoreOutput = nil
	}()

	//check the error status from channel creation here and return the error if it exists
	if err != nil {
		return err
	}

	restoreOutput <- "Starting restore"

	//TODO: acquire restore mutex, defer release
	var (
		doReloadLogstashContainer bool
		templates                 map[string]*servicetemplate.ServiceTemplate
		imagesNameTags            [][]string
	)
	defer func() {
		if doReloadLogstashContainer {
			go facade.LogstashContainerReloader(datastore.Get(), cp.facade) // don't block the main thread
		}
	}()
	restorePath := func(relPath ...string) string {
		return filepath.Join(append([]string{varPath(), "restore"}, relPath...)...)
	}

	if e := os.RemoveAll(restorePath()); e != nil {
		glog.Errorf("Could not remove %s: %v", restorePath(), e)
		restoreError <- e.Error()
		return e
	}

	if e := os.MkdirAll(restorePath(), os.ModeDir|0755); e != nil {
		glog.Errorf("Could not find nor create %s: %v", restorePath(), e)
		restoreError <- e.Error()
		return e
	}

	defer func() {
		if e := os.RemoveAll(restorePath()); e != nil {
			glog.Errorf("Could not remove %s: %v", restorePath(), e)
			if err == nil {
				err = e
			}
		}
	}()

	if e := writeDirectoryFromTgz(restorePath(), backupFilePath); e != nil {
		glog.Errorf("Could not expand %s to %s: %v", backupFilePath, restorePath(), e)
		restoreError <- e.Error()
		return e
	}

	if e := readJSONFromFile(&templates, restorePath("templates.json")); e != nil {
		glog.Errorf("Could not read templates from %s: %v", restorePath("templates.json"), e)
		restoreError <- e.Error()
		return e
	}

	if e := readJSONFromFile(&imagesNameTags, restorePath("images.json")); e != nil {
		glog.Errorf("Could not read images from %s: %v", restorePath("images.json"), e)
		restoreError <- e.Error()
		return e
	}

	// Restore the service templates ...
	for templateID, template := range templates {
		template.ID = templateID
		restoreOutput <- fmt.Sprintf("Restoring service template: %v", template.ID)
		if e := cp.UpdateServiceTemplate(*template, unused); e != nil {
			glog.Errorf("Could not update template %s: %v", templateID, e)
			restoreError <- e.Error()
			return e
		}
		doReloadLogstashContainer = true
	}

	// Restore the docker images ...
	for i, imageNameWithTags := range imagesNameTags {
		imageID := imageNameWithTags[0]
		imageTags := imageNameWithTags[1:]
		imageName := "imported:" + imageID
		restoreOutput <- fmt.Sprintf("Restoring Docker image: %v", imageName)
		image, e := docker.FindImage(imageID, false)
		if e != nil {
			if e != docker.ErrNoSuchImage {
				glog.Errorf("Unexpected error when inspecting docker image %s: %v", imageID, e)
				restoreError <- e.Error()
				return e
			}
			filename := restorePath("images", fmt.Sprintf("%d.tar", i))
			if e := importDockerImageFromFile(imageName, filename); e != nil {
				glog.Errorf("Could not import docker image %s (%+v) from file %s: %v", imageID, imageTags, filename, e)
				restoreError <- e.Error()
				return e
			}
			image, e = docker.FindImage(imageName, false)
			if e != nil {
				glog.Errorf("Could not find imported docker image %s (%+v): %v", imageName, imageTags, e)
				restoreError <- e.Error()
				return e
			}
		}

		//		if e := client.TagImage(imageID, dockerclient.TagImageOptions{Repo: "imported", Tag: imageID, Force: true}); e != nil {
		if _, e := image.Tag(imageID); e != nil {
			glog.Errorf("Found image %s already exists, but could not tag it: %s", imageID, e)
			restoreError <- e.Error()
			return e
		}

		for _, imageTag := range imageTags {
			img, e := docker.FindImage(imageName, false)
			if e != nil {
				return e
			}

			_, e = img.Tag(imageTag)
			if e != nil {
				return e
			}
		}
	}

	// Restore the snapshots ...
	snapFiles, e := readDirFileNames(restorePath("snapshots"))
	if e != nil {
		glog.Errorf("Could not list contents of %s: %v", restorePath("snapshots"), e)
		restoreError <- e.Error()
		return e
	}
	for _, snapFile := range snapFiles {
		snapshotID := strings.TrimSuffix(snapFile, ".tgz")
		restoreOutput <- fmt.Sprintf("Restoring snapshot: %v", snapshotID)
		if snapshotID == snapFile {
			continue //the filename does not end with .tgz
		}
		parts := strings.Split(snapshotID, "_")
		if len(parts) != 2 {
			glog.Warningf("Skipping restoration of snapshot %s, due to malformed ID!", snapshotID)
			continue
		}
		serviceID := parts[0]

		snapFilePath := restorePath("snapshots", snapFile)
		snapDirTemp := restorePath("snapshots", snapshotID)
		if e := writeDirectoryFromTgz(snapDirTemp, snapFilePath); e != nil {
			glog.Errorf("Could not write %s from %s: %v", snapDirTemp, snapFilePath, e)
			restoreError <- e.Error()
			return e
		}
		if e := cp.dfs.RollbackServices(snapDirTemp); e != nil {
			glog.Errorf("Could not rollback services: %s", e)
			restoreError <- e.Error()
			return e
		}

		var service service.Service
		if e := cp.GetService(serviceID, &service); e != nil {
			glog.Errorf("Could not find service %s for snapshot %s: %s", serviceID, snapshotID, e)
			restoreError <- e.Error()
			return e
		}

		snapDir, e := getSnapshotPath(cp.vfs, service.PoolID, service.ID, snapshotID)
		if e != nil {
			glog.Errorf("Could not get subvolume %s:%s: %v", service.PoolID, service.ID, e)
			restoreError <- e.Error()
			return e
		}

		if e = os.Rename(snapDirTemp, snapDir); e != nil {
			glog.Errorf("Could not move %s to %s: %s", snapDirTemp, snapDir, e)
			restoreError <- e.Error()
			return e
		}

		defer func() {
			var unused int
			if e := cp.DeleteSnapshot(snapshotID, &unused); e != nil {
				glog.Errorf("Couldn't delete snapshot %s: %v", snapshotID, e)
				if err == nil {
					err = e
				}
			}
		}()

		if e := cp.Rollback(snapshotID, unused); e != nil {
			glog.Errorf("Could not rollback to snapshot %s: %v", snapshotID, e)
			restoreError <- e.Error()
			return e
		}
	}

	//TODO: garbage collect (http://jimhoskins.com/2013/07/27/remove-untagged-docker-images.html)
	return nil
}

func readDirFileNames(dirname string) ([]string, error) {
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
