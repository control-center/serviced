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

// +build integration

package dfs

import (
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/control-center/serviced/commons/docker"

	"github.com/zenoss/glog"
)

func newNopCloser(rw io.ReadWriter) *nopCloser {
	return &nopCloser{ReadWriter: rw}
}

type nopCloser struct {
	io.ReadWriter
}

func (rw *nopCloser) Close() error {
	return nil
}

type log interface {
	Log(args ...interface{})
}

func TestBackup_writeDirectoryToAndFromTgz(t *testing.T) {

	t.Skip("TODO: separate this integration test from the unit tests in this package")

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
	if e := exportTGZ(dataDir, tgzFile); e != nil {
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
	if e := importTGZ(dataDir, tgzFile); e != nil {
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

	t.Skip("TODO: separate this integration test from the unit tests in this package")

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

	buffer := newNopCloser(bytes.NewBuffer([]byte{}))
	if e := exportJSON(buffer, original); e != nil {
		t.Fatalf("Could not write data %+v to buffer: %s", original, e)
	}
	var retrieved map[string][]int
	if e := importJSON(buffer, &retrieved); e != nil {
		t.Fatalf("Could not read data from buffer: %s", e)
	}
	if !reflect.DeepEqual(retrieved, original) {
		t.Errorf("Expected data: %+v", original)
		t.Errorf("Actual data  : %+v", retrieved)
		t.Fatal("Unexpected difference")
	}
}

func TestBackup_parseDisks(t *testing.T) {

	t.Skip("TODO: separate this integration test from the unit tests in this package")

	output := bytes.NewBufferString(`
	Filesystem                   1K-blocks     Used Available Use% Mounted on
	/dev/mapper/vagrant--vg-root  36774596 33984408    899068  98% /
	udev                           2010500        4   2010496   1% /dev
	tmpfs                           404808      356    404452   1% /run
	none                           2024036     1076   2022960   1% /run/shm
	`)

	expected := []diskinfo{
		{"/dev/mapper/vagrant--vg-root", 36774596, 33984408, 899068, 98, "/"},
		{"udev", 2010500, 4, 2010496, 1, "/dev"},
		{"tmpfs", 404808, 356, 404452, 1, "/run"},
		{"none", 2024036, 1076, 2022960, 1, "/run/shm"},
	}

	actual, err := parseDisks(output.Bytes())
	if err != nil {
		t.Fatalf("Could not parse: %s", err)
	} else if !reflect.DeepEqual(expected, actual) {
		t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected, actual)
	}
}

func TestBackup_parseTarInfo(t *testing.T) {

	t.Skip("TODO: separate this integration test from the unit tests in this package")

	output := bytes.NewBufferString(`
	drwxr-xr-x root/root         0 2014-08-22 13:04 ./
	drwxr-xr-x root/root         0 2014-08-22 13:04 ./images/
	-rw-r--r-- root/root 1801870336 2014-08-22 13:04 ./images/2.tar
	-rw-r--r-- root/root  520972800 2014-08-22 13:04 ./images/0.tar
	`)

	tparse := func(ts string) time.Time {
		result, err := time.Parse("2006-01-02 15:04", ts)
		if err != nil {
			t.Fatalf("Could not parse time: %s", err)
		}
		return result
	}
	expected := []tarinfo{
		{"drwxr-xr-x", "root", "root", 0, tparse("2014-08-22 13:04"), "./"},
		{"drwxr-xr-x", "root", "root", 0, tparse("2014-08-22 13:04"), "./images/"},
		{"-rw-r--r--", "root", "root", 1801870336, tparse("2014-08-22 13:04"), "./images/2.tar"},
		{"-rw-r--r--", "root", "root", 520972800, tparse("2014-08-22 13:04"), "./images/0.tar"},
	}
	tf, err := new(tarfile).init(output.Bytes())
	if err != nil {
		glog.Fatalf("Could not parse: %s", err)
	}
	actual := []tarinfo(*tf)

	if len(expected) != len(actual) {
		t.Fatalf("Mismatch (Expected: %v) (Actual: %v)", expected, actual)
	}

	for i := range expected {
		if expected[i].Permission != actual[i].Permission {
			t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected[i], actual[i])
		} else if expected[i].Owner != actual[i].Owner {
			t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected[i], actual[i])
		} else if expected[i].Group != actual[i].Group {
			t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected[i], actual[i])
		} else if expected[i].Size != actual[i].Size {
			t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected[i], actual[i])
		} else if !expected[i].Timestamp.Equal(actual[i].Timestamp) {
			t.Errorf("Mismatch (Expected: %v) (Actual: %v)", expected[i], actual[i])
		} else if strings.TrimSpace(expected[i].Filename) != actual[i].Filename {
			t.Errorf("Mismatch (Expected: \"%v\") (Actual: \"%v\")", expected[i].Filename, actual[i].Filename)
		}
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

func delete_docker_image(t log, imageId string) error {
	dockerCmd := exec.Command("docker", "rmi", imageId)
	if out, e := dockerCmd.CombinedOutput(); e != nil {
		t.Log(out)
		return e
	}
	return nil
}

func all_docker_images(t log) (map[string]bool, error) {
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

func get_docker_image_tags(t log, imageId string) (map[string]bool, error) {
	images, err := docker.Images()
	if err != nil {
		t.Log("Failure to list docker images")
		return nil, err
	}

	result := make(map[string]bool)
	for _, image := range images {
		if image.ID.BaseName() == imageId {
			result[image.ID.Tag] = true
		}
	}
	if len(result) == 0 {
		return nil, fmt.Errorf("No such docker image: %s", imageId)
	}

	return result, nil
}

func TestBackup_IntegrationTest(t *testing.T) {
	t.Skip("TODO: Fix this broken test. Maybe a race condition?")
	/*
		var (
			unused         int
			request        dao.ServiceRequest
			templateId     string
			serviceId      string
			backupFilePath string
			templates      map[string]servicetemplate.ServiceTemplate
			services       []service.Service
		)

		// Create a minimal docker image
		imageId, e := docker_scratch_image()
		if e != nil {
			t.Fatalf("Failed to create a scratch image: %s", e)
		}
		defer delete_docker_image(t, imageId)


			// Clean up old templates...
			if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
				t.Fatalf("Failure getting service templates with error: %s", e)
			}
			for id, _ := range templates {
				if e := dt.Dao.RemoveServiceTemplate(id, &unused); e != nil {
					t.Fatalf("Failure removing service template %s with error: %s", id, e)
				}
			}

			// Clean up old services...
			if e := dt.Dao.GetServices(request, &services); e != nil {
				t.Fatalf("Failure getting services: %s", e)
			}
			for _, service := range services {
				if e := dt.Dao.RemoveService(service.ID, &unused); e != nil {
					t.Fatalf("Failure removing service (%s): %s", service.ID, e)
				}
			}


		template_volume := servicedefinition.Volume{
			Owner:         "root:root",
			Permission:    "0755",
			ResourcePath:  "backup_test",
			ContainerPath: "/tmp/backup_test",
		}

		template_service := servicedefinition.ServiceDefinition{
			Name:    "test",
			Command: "echo",
			ImageID: imageId,
			Launch:  commons.MANUAL,
			Volumes: []servicedefinition.Volume{template_volume},
		}

		template := servicetemplate.ServiceTemplate{
			ID:          "",
			Name:        "test_template",
			Description: "test template",
			Services:    []servicedefinition.ServiceDefinition{template_service},
		}

		svc := service.Service{
			ID:             "testservice", //FIXME: Can't snapshot with a "_" in it.
			Name:           "test_service",
			Startup:        "echo",
			Instances:      0,
			InstanceLimits: domain.MinMax{Min: 0, Max: 0, Default: 0},
			ImageID:        imageId,
			Launch:         commons.MANUAL,
			PoolID:         "default",
			DesiredState:   service.SVCStop,
			Volumes:        []servicedefinition.Volume{template_volume},
			DeploymentID:   "backup_test",
		}

		originalImageIDs, e := all_docker_images(t)
		if e != nil {
			t.Fatalf("Failure getting list of docker images: %s", e)
		}

		originalTags, e := get_docker_image_tags(t, imageId)
		if e != nil {
			t.Fatalf("Failure getting docker image %s tags: %s", imageId, e)
		}


				// Create a minimal service template which uses the image and a DFS.
				if e := dt.Dao.AddServiceTemplate(template, &templateId); e != nil {
					t.Fatalf("Failed to add service template (%+v): %s", template, e)
				}
				template.ID = templateId
				defer dt.Dao.RemoveServiceTemplate(templateId, &unused)

				// Create a minimal service, based on the template.
				if e := dt.Dao.AddService(svc, &serviceId); e != nil {
					t.Fatalf("Failed to add service (%+v): %s", svc, e)
				}
				defer dt.Dao.RemoveService(serviceId, &unused)
				if e := dt.Dao.GetService(serviceId, &svc); e != nil {
					t.Fatalf("Failed to find serviced that was just added: %s", e)
				}

				// Write some data to the DFS
				volume, e := getSubvolume(dt.Dao.fsType, "default", serviceId)
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
				if e := dt.Dao.Backup("", &backupFilePath); e != nil {
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

			if e := dt.Dao.Restore(backupFilePath, &unused); e != nil {
				t.Fatalf("Failed restore from backup %s  error: %s", backupFilePath, e)
			}

			// Check: old docker image still there, no new docker images
			currentImageIDs, e := all_docker_images(t)
			if e != nil {
				t.Fatalf("Failure getting list of docker images: %s", e)
			}
			if !reflect.DeepEqual(originalImageIDs, currentImageIDs) {
				t.Errorf("Expected docker images: %+v", originalImageIDs)
				t.Errorf("  Actual docker images: %+v", currentImageIDs)
				t.Fatal("Unexpected difference")
			}

			// Check: find the old service, and no new services
			if e := dt.Dao.GetServices(request, &services); e != nil {
				t.Fatalf("Failure getting services: %s", e)
			}
			if len(services) != 1 {
				t.Fatalf("Expected just one service. Found %d", len(services))
			}
			if services[0].ID != serviceId {
				t.Fatalf("Expecting service %s, but found %s", serviceId, services[0].ID)
			}

			// Check: find the old template, and no new templates
			if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
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
			if e := dt.Dao.RemoveService(serviceId, &unused); e != nil {
				t.Fatalf("Failure removing service %s: %s", serviceId, e)
			}

			// Delete the template
			if e := dt.Dao.RemoveServiceTemplate(templateId, &unused); e != nil {
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

			dt.Dao.Restore(backupFilePath, &unused)

		// Check: new docker image imported, with same tags as old.
		// TODO: (Is there some way to compare the contents of the images?)
		currentImageIDs, e = all_docker_images(t)
		if e != nil {
			t.Fatalf("Failure getting list of docker images: %s", e)
		}
		if _, found := currentImageIDs[imageId[0:12]]; found {
			t.Fatal("Unexpectedly found original docker image still exists!")
		}

		delete(originalImageIDs, imageId[0:12])
		for imageId, _ := range originalImageIDs {
			if _, found := currentImageIDs[imageId]; !found {
				t.Fatalf("An unrelated docker image %s went missing!", imageId)
			}
			delete(currentImageIDs, imageId)
		}
		if len(currentImageIDs) != 1 {
			t.Fatalf("Expected to find one (1) new docker image. Found %d", len(currentImageIDs))
		}
		for imageId, _ := range currentImageIDs {
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
		if e = dt.Dao.GetServices(request, &services); e != nil {
			t.Fatalf("Failure getting services: %s", e)
		}
		if len(services) != 1 {
			t.Fatalf("Expected just one service. Found %d", len(services))
		}
		if services[0].ID != serviceId {
			t.Fatalf("Expecting service %s, but found %s", serviceId, services[0].ID)
		}

		// Check: find the old template, and no new templates
		if e := dt.Dao.GetServiceTemplates(0, &templates); e != nil {
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
	*/
}
