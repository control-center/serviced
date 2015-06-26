package migrate

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"

	"github.com/control-center/serviced/commons/docker"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/facade"
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

func RunMigrationScript(ctx datastore.Context, fcd *facade.Facade, request dao.RunMigrationScriptRequest) error {
	svc, err := fcd.GetService(datastore.Get(), request.ServiceID)
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

	svcs, err2 := fcd.GetServiceList(ctx, svc.ID)
	if err2 != nil {
		return err2
	}

	glog.V(3).Infof("Facade:RunMigrationScript: temp directory for service migration: %s", migrationDir)
	inputFileName, err = createServiceMigrationInputFile(migrationDir, svcs)
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

	err = fcd.MigrateServices(ctx, *migrationRequest)

	return err

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
func createServiceMigrationInputFile(tmpDir string, svcs []*service.Service) (string, error) {
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
