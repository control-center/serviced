// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package elasticsearch

import (
	"fmt"
	"github.com/zenoss/serviced/dao"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestBackup_writeDirectoryToAndFromTgz(t *testing.T) {
	//FIXME: Should also test that files are restored with the original owner
	// and permissions, even if the UID/GID is not a UID/GID on this system.

	//Setup...
	tgzDir, e := ioutil.TempDir("", "test-tgz")
	if e != nil {
		t.Fatalf("Failed to create temporary directory: %s", e)
	}
	defer os.RemoveAll(tgzDir)

	dataDir, e := ioutil.TempDir("", "test-data")
	if e != nil {
		t.Fatalf("Failed to create temporary directory: %s", e)
	}
	defer os.RemoveAll(dataDir)

	dataFile := filepath.Join(dataDir, "data.txt")
	data := []byte("macaroni and cheese")
	if e := ioutil.WriteFile(dataFile, data, 0600); e != nil {
		t.Fatalf("Failed writing file %s: %s", dataFile, e)
	}

	//Create tgz...
	tgzFile := filepath.Join(tgzDir, "data.tgz")
	if e := writeDirectoryToTgz(dataDir, tgzFile); e != nil {
		t.Fatalf("Failed writing directory %s to %s: %s", dataDir, tgzFile, e)
	}

	//More setup...
	if e := os.RemoveAll(dataDir); e != nil {
		t.Fatalf("Failed to remove directory %s: %s", dataDir, e)
	}
	if _, e := os.Stat(dataFile); !os.IsNotExist(e) {
		t.Fatal("Failed to prove that dataFile is missing")
	}

	//Restore from tgz...
	if e := writeDirectoryFromTgz(dataDir, tgzFile); e != nil {
		t.Fatalf("Failed writing directory %s from %s: %s", dataDir, tgzFile, e)
	}

	//Check...
	data, e = ioutil.ReadFile(dataFile)
	if e != nil {
		t.Fatalf("Failed to read file %s: %s", dataFile, e)
	}
	if string(data) != "macaroni and cheese" {
		t.Fatalf("Failed to restore the data: %s", e)
	}
}

func TestBackup_writeAndReadJsonToAndFromFile(t *testing.T) {
	original := make(map[string][]int)
	original["a"] = []int{0, 1, 2}
	original["b"] = []int{3, 4, 5}
	original["c"] = []int{6, 7, 8}
	original["d"] = []int{9}

	tempDir, e := ioutil.TempDir("", "test-json")
	if e != nil {
		t.Fatalf("Failed to create temporary directory: %s", e)
	}
	defer os.RemoveAll(tempDir)

	tempFile := filepath.Join(tempDir, "data.json")
	if e := writeJsonToFile(original, tempFile); e != nil {
		t.Fatalf("Failed to write data %+v to file %s: %s", original, tempFile, e)
	}

	json, e := ioutil.ReadFile(tempFile)
	if e != nil {
		t.Fatalf("Failed to read contents of file %s: %s", tempFile, e)
	}

	expected_json := "{\"a\":[0,1,2],\"b\":[3,4,5],\"c\":[6,7,8],\"d\":[9]}\n"
	if string(json) != expected_json {
		t.Errorf("Expected JSON: %s", expected_json)
		t.Errorf("Actual JSON  : %s", string(json))
		t.Fatal("Unexpected difference")
	}

	var retrieved map[string][]int
	if e := readJsonFromFile(&retrieved, tempFile); e != nil {
		t.Fatalf("Failed to read data from file %s: %s", tempFile, e)
	}

	if !reflect.DeepEqual(retrieved, original) {
		t.Errorf("Expected data: %+v", original)
		t.Errorf("Actual data  : %+v", retrieved)
		t.Fatal("Unexpected difference")
	}
}

func TestBackup_getDockerImageNameIds(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_exportDockerImageToFile(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_importDockerImageFromFile(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_repoAndTag(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_dockerImageSet(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_readDirFileNames(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_Backup(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func TestBackup_Restore(t *testing.T) {
	t.Skip("TODO: write unit test")
}

func docker_scratch_image() (string, error) {
	var e error
	tarCmd := exec.Command("tar", "cv", "--files-from", "/dev/null")
	dockerCmd := exec.Command("docker", "import", "-", "scratch")
	if dockerCmd.Stdin, e = tarCmd.StdoutPipe(); e != nil {
		return "", e
	}
	dockerOut, e := dockerCmd.StdoutPipe()
	if e != nil {
		return "", e
	}
	dockerErr, e := dockerCmd.StderrPipe()
	if e != nil {
		return "", e
	}
	dockerCmd.Start()
	tarCmd.Run()
	output := io.MultiReader(dockerOut, dockerErr)
	imageId, e := ioutil.ReadAll(output)
	if e != nil {
		return "", e
	}
	if e = dockerCmd.Wait(); e != nil {
		return "", e
	}
	return strings.TrimSpace(string(imageId)), nil
}

func delete_docker_image(t *testing.T, imageId string) error {
	dockerCmd := exec.Command("docker", "rmi", imageId)
	if out, e := dockerCmd.CombinedOutput(); e != nil {
		t.Log(out)
		return e
	}
	return nil
}

func all_docker_images(t *testing.T) (map[string]bool, error) {
	dockerCmd := exec.Command("docker", "images", "-q", "-a")
	out, e := dockerCmd.CombinedOutput()
	if e != nil {
		t.Log(out)
		return nil, e
	}
	result := make(map[string]bool)
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result[line[0:12]] = true
	}
	return result, nil
}

func get_docker_image_tags(t *testing.T, imageId string) (map[string]bool, error) {
	client, e := newDockerExporter()
	if e != nil {
		t.Log("Failure getting docker client")
		return nil, e
	}
	images, e := client.ListImages(true)
	if e != nil {
		t.Log("Failure to list docker images")
		return nil, e
	}
	for _, image := range images {
		if image.ID[0:12] == imageId[0:12] {
			result := make(map[string]bool)
			for _, repoTag := range image.RepoTags {
				result[repoTag] = true
			}
			return result, nil
		}
	}
	return nil, fmt.Errorf("No such docker image: %s", imageId)
}

func TestBackup_IntegrationTest(t *testing.T) {
	var (
		unused         int
		request        dao.EntityRequest
		templateId     string
		serviceId      string
		backupFilePath string
		templates      map[string]*dao.ServiceTemplate
		services       []*dao.Service
	)

	// Create a minimal docker image
	imageId, e := docker_scratch_image()
	if e != nil {
		t.Fatalf("Failed to create a scratch image: %s", e)
	}
	defer delete_docker_image(t, imageId)

	// Clean up old templates...
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failure getting service templates with error: %s", e)
	}
	for id, _ := range templates {
		if e := controlPlaneDao.RemoveServiceTemplate(id, &unused); e != nil {
			t.Fatalf("Failure removing service template %s with error: %s", id, e)
		}
	}

	// Clean up old services...
	if e := controlPlaneDao.GetServices(request, &services); e != nil {
		t.Fatalf("Failure getting services: %s", e)
	}
	for _, service := range services {
		if e := controlPlaneDao.RemoveService(service.Id, &unused); e != nil {
			t.Fatalf("Failure removing service (%s): %s", service.Id, e)
		}
	}

	template_volume := dao.Volume{
		Owner:         "root:root",
		Permission:    "0755",
		ResourcePath:  "backup_test",
		ContainerPath: "/tmp/backup_test",
	}

	template_service := dao.ServiceDefinition{
		Name:    "test",
		Command: "echo",
		ImageId: imageId,
		Launch:  "MANUAL",
		Volumes: []dao.Volume{template_volume},
	}

	template := dao.ServiceTemplate{
		Id:          "",
		Name:        "test_template",
		Description: "test template",
		Services:    []dao.ServiceDefinition{template_service},
	}

	service := dao.Service{
		Id:           "testservice", //FIXME: Can't snapshot with a "_" in it.
		Name:         "test_service",
		Startup:      "echo",
		Instances:    0,
		ImageId:      imageId,
		PoolId:       "default",
		DesiredState: 0,
		Volumes:      []dao.Volume{template_volume},
		DeploymentId: "backup_test",
	}

	originalImageIds, e := all_docker_images(t)
	if e != nil {
		t.Fatalf("Failure getting list of docker images: %s", e)
	}

	originalTags, e := get_docker_image_tags(t, imageId)
	if e != nil {
		t.Fatalf("Failure getting docker image %s tags: %s", imageId, e)
	}

	// Create a minimal service template which uses the image and a DFS.
	if e := controlPlaneDao.AddServiceTemplate(template, &templateId); e != nil {
		t.Fatalf("Failed to add service template (%+v): %s", template, e)
	}
	template.Id = templateId
	defer controlPlaneDao.RemoveServiceTemplate(templateId, &unused)

	// Create a minimal service, based on the template.
	if e := controlPlaneDao.AddService(service, &serviceId); e != nil {
		t.Fatalf("Failed to add service (%+v): %s", service, e)
	}
	defer controlPlaneDao.RemoveService(serviceId, &unused)
	if e := controlPlaneDao.GetService(serviceId, &service); e != nil {
		t.Fatalf("Failed to find serviced that was just added: %s", e)
	}

	// Write some data to the DFS
	volume, e := getSubvolume(controlPlaneDao.vfs, "default", serviceId)
	if e != nil {
		t.Fatalf("Failed to get subvolume: %s", e)
	}
	volumePath := volume.Path()
	dataFile := filepath.Join(volumePath, "backedup.txt")
	data := []byte("cheese and crackers")
	if e := ioutil.WriteFile(dataFile, data, 0600); e != nil {
		t.Fatalf("Failed writing file %s: %s", dataFile, e)
	}
	defer os.RemoveAll(dataFile)

	// Backup
	if e := controlPlaneDao.Backup("", &backupFilePath); e != nil {
		t.Fatalf("Failed while making a backup: %s", e)
	}
	defer os.RemoveAll(backupFilePath)

	// Write some other data to the DFS.
	otherFile := filepath.Join(volumePath, "notbackuped.txt")
	otherData := []byte("peanut butter and jelly")
	if e := ioutil.WriteFile(otherFile, otherData, 0600); e != nil {
		t.Fatalf("Failed writing file %s: %s", otherFile, e)
	}
	defer os.RemoveAll(otherFile)

	controlPlaneDao.Restore(backupFilePath, &unused)

	// Check: old docker image still there, no new docker images
	currentImageIds, e := all_docker_images(t)
	if e != nil {
		t.Fatalf("Failure getting list of docker images: %s", e)
	}
	if !reflect.DeepEqual(originalImageIds, currentImageIds) {
		t.Errorf("Expected docker images: %+v", originalImageIds)
		t.Errorf("  Actual docker images: %+v", currentImageIds)
		t.Fatal("Unexpected difference")
	}

	// Check: find the old service, and no new services
	if e := controlPlaneDao.GetServices(request, &services); e != nil {
		t.Fatalf("Failure getting services: %s", e)
	}
	if len(services) != 1 {
		t.Fatalf("Expected just one service. Found %d", len(services))
	}
	if services[0].Id != serviceId {
		t.Fatalf("Expecting service %s, but found %s", serviceId, services[0].Id)
	}

	// Check: find the old template, and no new templates
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failed to get templates: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected just one template. Found %d", len(templates))
	}
	if _, found := templates[templateId]; !found {
		t.Fatalf("Template %s has gone missing", templateId)
	}

	// Check: DFS has pre-backup data, but no post-backup data.
	if data, e := ioutil.ReadFile(dataFile); e != nil {
		t.Fatalf("Failed while reading file %s: %s", dataFile, e)
	} else if string(data) != "cheese and crackers" {
		t.Fatalf("File %s has unexpected contents: %s", dataFile, e)
	}
	if _, e := os.Stat(otherFile); e == nil {
		t.Fatalf("Expected file %s to be missing, but I found it!", otherFile)
	} else if !os.IsNotExist(e) {
		t.Fatalf("Failed while checking to see if file %s exists: %s", otherFile, e)
	}

	// Delete the service
	if e := controlPlaneDao.RemoveService(serviceId, &unused); e != nil {
		t.Fatalf("Failure removing service %s: %s", serviceId, e)
	}

	// Delete the template
	if e := controlPlaneDao.RemoveServiceTemplate(templateId, &unused); e != nil {
		t.Fatalf("Failure removing template %s: %s", templateId, e)
	}

	// Delete the docker image
	if e := delete_docker_image(t, imageId); e != nil {
		t.Fatalf("Failure deleting docker image %s: %s", imageId, e)
	}

	// Ensure the DFS is gone.
	if _, e := os.Stat(dataFile); e == nil {
		if e = os.RemoveAll(dataFile); e != nil {
			t.Fatalf("Unable to remove the file %s: %s", dataFile, e)
		}
	}

	controlPlaneDao.Restore(backupFilePath, &unused)

	// Check: new docker image imported, with same tags as old.
	// TODO: (Is there some way to compare the contents of the images?)
	currentImageIds, e = all_docker_images(t)
	if e != nil {
		t.Fatalf("Failure getting list of docker images: %s", e)
	}
	if _, found := currentImageIds[imageId[0:12]]; found {
		t.Fatal("Unexpectedly found original docker image still exists!")
	}

	delete(originalImageIds, imageId[0:12])
	for imageId, _ := range originalImageIds {
		if _, found := currentImageIds[imageId]; !found {
			t.Fatalf("An unrelated docker image %s went missing!", imageId)
		}
		delete(currentImageIds, imageId)
	}
	if len(currentImageIds) != 1 {
		t.Fatalf("Expected to find one (1) new docker image. Found %d", len(currentImageIds))
	}
	for imageId, _ := range currentImageIds {
		currentTags, e := get_docker_image_tags(t, imageId)
		if e != nil {
			t.Fatalf("Failure getting docker image %s tags: %s", imageId, e)
		}
		for tag, _ := range originalTags {
			if _, found := currentTags[tag]; !found {
				t.Fatalf("Imported image is missing original image's tag %s", tag)
			}
		}
	}

	// Check: find the old service, and no new services
	if e = controlPlaneDao.GetServices(request, &services); e != nil {
		t.Fatalf("Failure getting services: %s", e)
	}
	if len(services) != 1 {
		t.Fatalf("Expected just one service. Found %d", len(services))
	}
	if services[0].Id != serviceId {
		t.Fatalf("Expecting service %s, but found %s", serviceId, services[0].Id)
	}

	// Check: find the old template, and no new templates
	if e := controlPlaneDao.GetServiceTemplates(0, &templates); e != nil {
		t.Fatalf("Failed to get templates: %s", e)
	}
	if len(templates) != 1 {
		t.Fatalf("Expected just one template. Found %d", len(templates))
	}
	if _, found := templates[templateId]; !found {
		t.Fatalf("Template %s has gone missing", templateId)
	}

	// Check: DFS has pre-backup data, but no post-backup data.
	if data, e := ioutil.ReadFile(dataFile); e != nil {
		t.Fatalf("Failed while reading file %s: %s", dataFile, e)
	} else if string(data) != "cheese and crackers" {
		t.Fatalf("File %s has unexpected contents: %s", dataFile, e)
	}
	if _, e := os.Stat(otherFile); e == nil {
		t.Fatalf("Expected file %s to be missing, but I found it!", otherFile)
	} else if !os.IsNotExist(e) {
		t.Fatalf("Failed while checking to see if file %s exists: %s", otherFile, e)
	}
}
