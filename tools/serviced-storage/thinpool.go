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

package main

import (
	"bytes"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
)

type ThinPoolCreate struct {
	Args struct {
		Purpose string   `description:"Purpose of the thin pool (docker|serviced)"`
		Devices []string `description:"Block devices to use" required:"1"`
	} `positional-args:"yes" required:"yes"`
}

type LogicalVolumeInfo struct {
	Name        string
	KernelMajor uint
	KernelMinor uint
}

// runCommand runs the command and returns the stdout, stderr, exit code, and error.
// If the command ran but returned non-zero, the error is nil
func runCommand(cmd *exec.Cmd) (stdout string, stderr string, exitCode int, err error) {
	var stderrBuffer bytes.Buffer
	var stdoutBuffer bytes.Buffer
	cmd.Stderr = &stderrBuffer
	cmd.Stdout = &stdoutBuffer
	cmdErr := cmd.Run()
	exitCode, success := utils.GetExitStatus(cmdErr)
	if success {
		cmdErr = nil
	}
	return stdoutBuffer.String(), stderrBuffer.String(), exitCode, cmdErr
}

func (c *ThinPoolCreate) Execute(args []string) error {
	App.initializeLogging()
	purpose := c.Args.Purpose
	devices := c.Args.Devices
	logger := log.WithFields(log.Fields{
		"purpose": purpose,
		"devices": devices,
	})
	if purpose != "serviced" && purpose != "docker" {
		logger.Fatal("Purpose must be one of (docker, serviced)")
	}

	logger.Info("Creating thin-pool")
	if err := createThinPool(purpose, devices); err != nil {
		logger.Fatal(err)
	}

	return nil
}

func createThinPool(purpose string, devices []string) error {
	if err := ensurePhysicalDevices(devices); err != nil {
		return err
	}

	vg := purpose
	if err := createVolumeGroup(vg, devices); err != nil {
		return err
	}

	metadataVolume, err := createMetadataVolume(vg)
	if err != nil {
		return err
	}

	dataVolume, err := createDataVolume(vg)
	if err != nil {
		return err
	}

	err = convertToThinPool(vg, dataVolume, metadataVolume)
	if err != nil {
		return err
	}

	thinPoolName, err := getThinPoolNameForLogicalVolume(vg)
	if err != nil {
		return err
	}

	var msg string = ""
	switch purpose {
	case "docker":
		msg = "Please add the following to /etc/default/docker, stop Docker, " +
			"delete /var/lib/docker, and start it again:\n\n" +
			"DOCKER_OPTS=\"${DOCKER_OPTS} --storage-opt dm.thinpooldev=" + thinPoolName + "\"\n"
	case "serviced":
		msg = "Please add the following to /etc/default/serviced:\n\n" +
			"SERVICED_DM_THINPOOLDEV=\"" + thinPoolName + "\"\n"
	}
	fmt.Println(msg)
	return nil
}

func ensurePhysicalDevices(devices []string) error {
	for _, device := range devices {
		cmd := exec.Command("pvs", device)
		_, _, exitCode, err := runCommand(cmd)
		if err != nil {
			return err
		}
		if exitCode == 0 {
			continue
		}

		args := []string{"pvcreate", device}
		log.Info(strings.Join(args, " "))
		cmd = exec.Command(args[0], args[1:]...)
		stdout, stderr, exitCode, err := runCommand(cmd)
		if err != nil {
			return err
		}
		if exitCode != 0 {
			return fmt.Errorf("Error(%d) running '%s':\n%s",
				exitCode, strings.Join(args, " "), stderr)
		}
		log.Info(stdout)
	}
	return nil
}

func createVolumeGroup(vg string, devices []string) error {
	args := append([]string{"vgcreate", vg}, devices...)
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}
	log.Info(stdout)
	return nil
}

func createMetadataVolume(vg string) (string, error) {
	units := "s" // volume size will be measured in sectors
	totalSize, err := getVolumeGroupSize(vg, units)
	metadataSize := (totalSize + 999) / 1000
	metadataName := vg + "-meta"

	args := []string{"lvcreate",
		"--size", fmt.Sprintf("%d%s", metadataSize, units),
		"--name", metadataName,
		vg}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}
	log.Info(stdout)
	return metadataName, err
}

func createDataVolume(vg string) (string, error) {
	units := "b" // volume size will be measured in bytes
	totalSize, err := getVolumeGroupSize(vg, units)
	dataSize := (totalSize*90/100 + 511) &^ 511
	dataName := vg + "-pool"

	args := []string{"lvcreate",
		"--size", fmt.Sprintf("%d%s", dataSize, units),
		"--name", dataName,
		vg}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return "", err
	}
	if exitCode != 0 {
		return "", fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}
	log.Info(stdout)
	return dataName, err
}

func convertToThinPool(volumeGroup, dataVolume string, metadataVolume string) error {
	args := []string{"lvconvert",
		"--zero", "n",
		"--thinpool", fmt.Sprintf("%s/%s", volumeGroup, dataVolume),
		"--poolmetadata", fmt.Sprintf("%s/%s", volumeGroup, metadataVolume),
	}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return err
	}
	if exitCode != 0 {
		return fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}
	log.Info(stdout)
	return nil
}

func getVolumeGroupSize(vg string, units string) (uint64, error) {
	args := []string{"vgs",
		"--noheadings",
		"--nosuffix",
		"--units", units,
		"--options", "vg_free",
		vg}
	cmd := exec.Command(args[0], args[1:]...)
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return 0, err
	}
	if exitCode != 0 {
		return 0, fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}

	sizeString := strings.Trim(stdout, " \n")
	size, err := strconv.ParseUint(sizeString, 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func getInfoForLogicalVolume(name string) (LogicalVolumeInfo, error) {
	lvi := LogicalVolumeInfo{}
	args := []string{"pvs",
		"--noheadings",
		"--options", "lv_name,kernel_major,kernel_minor",
		name}
	cmd := exec.Command(args[0], args[1:]...)
	_, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return lvi, err
	}
	if exitCode != 0 {
		return lvi, fmt.Errorf("Error(%d) running '%s':\n%s",
			exitCode, strings.Join(args, " "), stderr)
	}
	// TODO: parse lines, match lv name, initialize retval fields

	return lvi, nil
}

func getThinPoolNameForLogicalVolume(name string) (string, error) {
	info, err := getInfoForLogicalVolume(name)
	if err != nil {
		return "", err
	}
	filename := fmt.Sprintf("/sys/dev/block/%d:%d/dm/name",
		info.KernelMajor, info.KernelMinor)
	contents := filename //TODO: read from filename
	return contents, nil
}
