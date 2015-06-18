// Copyright 2015 The Serviced Authors.
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

package facade

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/addressassignment"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/utils"
	"github.com/zenoss/glog"
	dockerclient "github.com/zenoss/go-dockerclient"
)

const (
	// The mount point in the service migration docker image
	MIGRATION_MOUNT_POINT = "/migration"

	// The well-known path within the service's docker image of the directory which contains the service's migration script
	EMBEDDED_MIGRATION_DIRECTORY = "/opt/serviced/migration"
)

func (f *Facade) ServiceUse(ctx datastore.Context, serviceID string, imageName string, registry string, noOp bool) (string, error) {
	result, err := docker.ServiceUse(serviceID, imageName, registry, noOp)
	if err != nil {
		return "", err
	}
	return result, nil
}

// TODO: Should we use a lock to serialize migration for a given service? ditto for Add and UpdateService?
func (f *Facade) RunMigrationScript(ctx datastore.Context, request dao.RunMigrationScriptRequest) error {
	svc, err := f.GetService(datastore.Get(), request.ServiceID)
	if err != nil {
		glog.Errorf("ControlPlaneDao.RunMigrationScript: could not find service id %+v: %s", request.ServiceID, err)
		return err
	}

	glog.V(2).Infof("Facade:RunMigrationScript: start for service id %+v (dry-run=%v, sdkVersion=%s)",
		svc.ID, request.DryRun, request.SDKVersion)

	var migrationDir, inputFileName, scriptFileName, outputFileName string
	migrationDir, err = createTempMigrationDir(svc.ID)
	defer os.RemoveAll(migrationDir)
	if err != nil {
		return err
	}

	_, svcs, err2 := f.GetServicesByTenant(ctx, svc.ID)
	if err2 != nil {
		return err2
	}

	glog.V(3).Infof("Facade:RunMigrationScript: temp directory for service migration: %s", migrationDir)
	inputFileName, err = f.createServiceMigrationInputFile(migrationDir, svcs)
	if err != nil {
		return err
	}

	if request.ScriptBody != "" {
		scriptFileName, err = createServiceMigrationScriptFile(migrationDir, request.ScriptBody)
		if err != nil {
			return err
		}

		_, scriptFile := path.Split(scriptFileName)
		containerScript := path.Join(MIGRATION_MOUNT_POINT, scriptFile)
		outputFileName, err = executeMigrationScript(svc.ID, nil, migrationDir, containerScript, inputFileName, request.SDKVersion)
		if err != nil {
			return err
		}
	} else {
		container, err := createServiceContainer(svc)
		if err != nil {
			return err
		} else {
			defer func() {
				if err := container.Delete(true); err != nil {
					glog.Errorf("Could not remove container %s (%s): %s", container.ID, svc.ImageID, err)
				}
			}()
		}

		containerScript := path.Join(EMBEDDED_MIGRATION_DIRECTORY, request.ScriptName)
		outputFileName, err = executeMigrationScript(svc.ID, container, migrationDir, containerScript, inputFileName, request.SDKVersion)
		if err != nil {
			return err
		}
	}

	migrationRequest, err := readServiceMigrationRequestFromFile(outputFileName)
	if err != nil {
		return err
	}

	migrationRequest.DryRun = request.DryRun

	err = f.MigrateServices(ctx, *migrationRequest)

	return err

}

func (f *Facade) MigrateServices(ctx datastore.Context, request dao.ServiceMigrationRequest) error {
	var err error

	// Validate the modified services.
	for _, svc := range request.Modified {
		if err = f.verifyServiceForUpdate(ctx, svc, nil); err != nil {
			return err
		}
	}

	// Make required mutations to the added services.
	for _, svc := range request.Added {
		svc.ID, err = utils.NewUUID36()
		if err != nil {
			return err
		}
		now := time.Now()
		svc.CreatedAt = now
		svc.UpdatedAt = now
		for _, ep := range svc.Endpoints {
			ep.AddressAssignment = addressassignment.AddressAssignment{}
		}
	}

	if err = f.validateAddedMigrationServices(ctx, request.Added); err != nil {
		return err
	}

	if err = f.validateServiceDeploymentRequests(ctx, request.Deploy); err != nil {
		return err
	}

	// If this isn't a dry run, make the changes.
	if !request.DryRun {

		// Add the added services.
		for _, svc := range request.Added {
			if err = f.AddService(ctx, *svc, false, true); err != nil {
				return err
			}
		}

		// Migrate the modified services.
		for _, svc := range request.Modified {
			if err = f.stopServiceForUpdate(ctx, *svc); err != nil {
				return err
			}
			if err = f.UpdateService(ctx, *svc, false); err != nil {
				return err
			}
		}

		// Deploy the service definitions.
		for _, request := range request.Deploy {
			parent, err := f.serviceStore.Get(ctx, request.ParentID)
			if err != nil {
				glog.Errorf("Could not get parent service %s", request.ParentID)
				return err
			}
			_, err = f.DeployService(ctx, parent.PoolID, request.ParentID, false, request.Service)
			if err != nil {
				glog.Errorf("Could not deploy service definition: %+v", request.Service)
				return err
			}

		}
	}

	return nil
}

func (f *Facade) stopServiceForUpdate(ctx datastore.Context, svc service.Service) error {
	//cannot update service without validating it.
	if svc.DesiredState != int(service.SVCStop) {
		if err := f.canStartService(ctx, svc.ID, false); err != nil {
			glog.Warningf("Could not validate service %s (%s) for starting: %s", svc.Name, svc.ID, err)
			svc.DesiredState = int(service.SVCStop)
		}

		for _, ep := range svc.GetServiceVHosts() {
			for _, vh := range ep.VHosts {
				//check that vhosts aren't already started elsewhere
				if err := zkAPI(f).CheckRunningVHost(vh, svc.ID); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func (f *Facade) validateServiceDeploymentRequests(ctx datastore.Context, requests []*dao.ServiceDeploymentRequest) error {
	for _, request := range requests {

		// Make sure the parent exists.
		parent, err := f.serviceStore.Get(ctx, request.ParentID)
		if err != nil {
			glog.Errorf("Could not get parent service %s", request.ParentID)
			return err
		}

		// Make sure we can build the service definition into a service.
		_, err = service.BuildService(request.Service, request.ParentID, parent.PoolID, int(service.SVCStop), parent.DeploymentID)
		if err != nil {
			glog.Errorf("Could not create service: %s", err)
			return err
		}

	}

	return nil
}

func (f *Facade) validateAddedMigrationServices(ctx datastore.Context, addedSvcs []*service.Service) error {

	// Create a list of all endpoint Application names.
	existing, err := f.serviceStore.GetServices(ctx)
	if err != nil {
		return err
	}
	apps := map[string]bool{}
	for _, svc := range existing {
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				apps[ep.Application] = true
			}
		}
	}

	// Perform the validation.
	for _, svc := range addedSvcs {

		// verify the service with name and parent does not collide with another existing service
		if s, err := f.serviceStore.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
			return err
		} else if s != nil {
			if s.ID != svc.ID {
				return fmt.Errorf("ValidationError: Duplicate name detected for service %s found at %s", svc.Name, svc.ParentServiceID)
			}
		}

		// Make sure no endpoint apps are duplicated.
		for _, ep := range svc.Endpoints {
			if ep.Purpose == "export" {
				if _, ok := apps[ep.Application]; ok {
					return fmt.Errorf("ValidationError: Duplicate Application detected for endpoint %s found for service %s id %s", ep.Name, svc.Name, svc.ID)
				}
			}
		}
	}

	return nil
}

// Verify that the svc is valid for update.
// Should be called for all updated (edited), and migrated services.
// This method is only responsible for validation.
func (f *Facade) verifyServiceForUpdate(ctx datastore.Context, svc *service.Service, oldSvc **service.Service) error {
	glog.V(2).Infof("Facade:verifyServiceForUpdate: service ID %+v", svc.ID)

	id := strings.TrimSpace(svc.ID)
	if id == "" {
		return errors.New("empty Service.ID not allowed")
	}
	svc.ID = id

	svcStore := f.serviceStore
	currentSvc, err := svcStore.Get(ctx, svc.ID)
	if err != nil {
		return err
	}

	err = svc.ValidEntity()
	if err != nil {
		return err
	}

	// verify the service with name and parent does not collide with another existing service
	if s, err := svcStore.FindChildService(ctx, svc.DeploymentID, svc.ParentServiceID, svc.Name); err != nil {
		glog.Errorf("Could not verify service path for %s: %s", svc.Name, err)
		return err
	} else if s != nil {
		if s.ID != svc.ID {
			err := fmt.Errorf("service %s found at %s", svc.Name, svc.ParentServiceID)
			glog.Errorf("Cannot update service %s: %s", svc.Name, err)
			return err
		}
	}

	// Primary service validation
	if err := svc.ValidEntity(); err != nil {
		return err
	}

	// make sure that the tenant ID and path are valid
	_, _, err = f.GetServicePath(ctx, svc.ID)
	if err != nil {
		return err
	}

	if oldSvc != nil {
		*oldSvc = currentSvc
	}
	return nil
}

// Creates a temporary directory to hold files related to service migration
func createTempMigrationDir(serviceID string) (string, error) {
	tmpParentDir := utils.TempDir("service-migration")
	err := os.MkdirAll(tmpParentDir, 0750)
	if err != nil {
		return "", fmt.Errorf("Unable to create temporary directory: %s", err)
	}

	var migrationDir string
	dirPrefix := fmt.Sprintf("%s-", serviceID)
	migrationDir, err = ioutil.TempDir(tmpParentDir, dirPrefix)
	if err != nil {
		return "", fmt.Errorf("Unable to create temporary directory: %s", err)
	}

	return migrationDir, nil
}

// Write out the service definition as a JSON file for use as input to the service migration
func (f *Facade) createServiceMigrationInputFile(tmpDir string, svcs []service.Service) (string, error) {
	inputFileName := path.Join(tmpDir, "input.json")
	jsonServices, err := json.MarshalIndent(svcs, " ", "  ")
	if err != nil {
		return "", fmt.Errorf("error marshalling service: %s", err)
	}

	err = ioutil.WriteFile(inputFileName, jsonServices, 0440)
	if err != nil {
		return "", fmt.Errorf("error writing service to temp file: %s", err)
	}

	return inputFileName, nil
}

// Write out the body of the script to a file
func createServiceMigrationScriptFile(tmpDir, scriptBody string) (string, error) {
	scriptFileName := path.Join(tmpDir, "migrate.py")
	err := ioutil.WriteFile(scriptFileName, []byte(scriptBody), 0440)
	if err != nil {
		return "", fmt.Errorf("error writing to script file: %s", err)
	}

	return scriptFileName, nil
}

func createServiceContainer(service *service.Service) (*docker.Container, error) {
	var emptyStruct struct{}
	containerName := fmt.Sprintf("%s-%s", service.Name, "migration")
	containerDefinition := &docker.ContainerDefinition{
		dockerclient.CreateContainerOptions{
			Name: containerName,
			Config: &dockerclient.Config{
				User:       "root",
				WorkingDir: "/",
				Image:      service.ImageID,
				Volumes:    map[string]struct{}{EMBEDDED_MIGRATION_DIRECTORY: emptyStruct},
			},
		},
		dockerclient.HostConfig{},
	}

	container, err := docker.NewContainer(containerDefinition, false, 0, nil, nil)
	if err != nil {
		glog.Errorf("Error trying to create container %v: %v", containerDefinition, err)
		return nil, err
	}

	glog.V(1).Infof("Created container %s named %s based on image %s", container.ID, containerName, service.ImageID)
	return container, nil
}

// executeMigrationScript executes containerScript in a docker container based
// the service migration SDK image.
//
// tmpDir is the temporary directory that is mounted into the service migration container under
// the directory identified by MIGRATON_MOUNT_POINT. Both the input and output files are written to
// tmpDir/MIGRATION_MOUNT_POINT
//
// The value of containerScript should be always be a fully qualified, container-local path
// to the service migration script, though the path may vary depending on the value of serviceContainer.
// If serviceContainer is not specified, then containerScript should start with MIGRATON_MOUNT_POINT
// If serviceContainer is specified, then the service-migration container will be run
// with volume(s) mounted from serviceContainer. This allows for cases where containerScript physically
// resides in the serviceContainer; i.e. under the directory specified by EMBEDDED_MIGRATION_DIRECTORY
//
// Returns the name of the file under tmpDir containing the output from the migration script
func executeMigrationScript(serviceID string, serviceContainer *docker.Container, tmpDir, containerScript, inputFilePath, sdkVersion string) (string, error) {
	const SERVICE_MIGRATION_IMAGE_NAME = "zenoss/service-migration_v1"
	const SERVICE_MIGRATION_TAG_NAME = "1.0.0"
	const OUTPUT_FILE = "output.json"

	// get the container-local path names for the input and output files.
	_, inputFile := path.Split(inputFilePath)
	containerInputFile := path.Join(MIGRATION_MOUNT_POINT, inputFile)
	containerOutputFile := path.Join(MIGRATION_MOUNT_POINT, OUTPUT_FILE)

	tagName := SERVICE_MIGRATION_TAG_NAME
	if sdkVersion != "" {
		tagName = sdkVersion
	} else if tagOverride := os.Getenv("SERVICED_SERVICE_MIGRATION_TAG"); tagOverride != "" {
		tagName = tagOverride
	}

	glog.V(2).Infof("Facade:executeMigrationScript: using docker tag=%q", tagName)
	dockerImage := fmt.Sprintf("%s:%s", SERVICE_MIGRATION_IMAGE_NAME, tagName)

	mountPath := fmt.Sprintf("%s:%s:rw", tmpDir, MIGRATION_MOUNT_POINT)
	runArgs := []string{
		"run", "--rm", "-t",
		"--name", "service-migration",
		"-e", fmt.Sprintf("MIGRATE_INPUTFILE=%s", containerInputFile),
		"-e", fmt.Sprintf("MIGRATE_OUTPUTFILE=%s", containerOutputFile),
		"-v", mountPath,
	}
	if serviceContainer != nil {
		runArgs = append(runArgs, "--volumes-from", serviceContainer.ID)
	}
	runArgs = append(runArgs, dockerImage)
	runArgs = append(runArgs, "python", containerScript)

	cmd := exec.Command("docker", runArgs...)

	glog.V(2).Infof("Facade:executeMigrationScript: service ID %+v: cmd: %v", serviceID, cmd)

	cmdMessages, err := cmd.CombinedOutput()
	if exitStatus, _ := utils.GetExitStatus(err); exitStatus != 0 {
		err := fmt.Errorf("migration script failed: %s", err)
		if cmdMessages != nil {
			glog.Errorf("Service migration script for %s reported: %s", serviceID, string(cmdMessages))
		}
		return "", err
	}
	if cmdMessages != nil {
		glog.V(1).Infof("Service migration script for %s reported: %s", serviceID, string(cmdMessages))
	}

	return path.Join(tmpDir, OUTPUT_FILE), nil
}

func readServiceMigrationRequestFromFile(outputFileName string) (*dao.ServiceMigrationRequest, error) {
	data, err := ioutil.ReadFile(outputFileName)
	if err != nil {
		return nil, fmt.Errorf("could not read new service definition: %s", err)
	}

	var request dao.ServiceMigrationRequest
	if err = json.Unmarshal(data, &request); err != nil {
		return nil, fmt.Errorf("could not unmarshall new service definition: %s", err)
	}
	return &request, nil
}