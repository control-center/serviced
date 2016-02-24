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
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os/exec"
	"regexp"
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

func checkCommand(cmd *exec.Cmd) (stdout string, stderr string, err error) {
	stdout, stderr, exitCode, err := runCommand(cmd)
	if err != nil {
		return stdout, stderr, err
	}
	if exitCode != 0 {
		return stdout, stderr, fmt.Errorf("Error(%d) running command '%s':\n%s",
			exitCode, strings.Join(cmd.Args, " "), stderr)
	}
	return stdout, stderr, nil
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
	thinPoolName, err := createThinPool(purpose, devices)
	if err != nil {
		logger.Fatal(err)
	}

	fmt.Printf("Created thin-pool device '%s'\n", thinPoolName)

	return nil
}

func createThinPool(purpose string, devices []string) (string, error) {

	if err := EnsureNotLogicalVolumes(devices); err != nil {
		return "", err
	}

	if err := EnsurePhysicalDevices(devices); err != nil {
		return "", err
	}

	volumeGroup := purpose
	if err := CreateVolumeGroup(volumeGroup, devices); err != nil {
		return "", err
	}

	// TODO: Rather than creating separate data and metadata volumes, then
	//  converting the data volume to a thin pool, we should simplify and use
	//  a single command to do it all:
	//     sudo lvcreate --type thin-pool -L 20G --poolmetadatasize 1G vg0/tp1
	// http://man7.org/linux/man-pages/man7/lvmthin.7.html
	metadataVolume, err := CreateMetadataVolume(volumeGroup)
	if err != nil {
		return "", err
	}

	dataVolume, err := CreateDataVolume(volumeGroup)
	if err != nil {
		return "", err
	}

	err = ConvertToThinPool(volumeGroup, dataVolume, metadataVolume)
	if err != nil {
		return "", err
	}

	lvInfo, err := GetInfoForLogicalVolume(dataVolume)
	if err != nil {
		return "", err
	}

	thinPoolName, err := lvInfo.GetThinpoolName()
	if err != nil {
		return "", nil
	}

	return thinPoolName, nil
}

func EnsureNotLogicalVolumes(devices []string) error {
	for _, device := range devices {
		args := []string{
			"lvs",
			"--noheadings",
			"--separator=,",
			"-o",
			"lv_name,vg_name",
			device,
		}
		log.Info(strings.Join(args, " "))
		cmd := exec.Command(args[0], args[1:]...)
		stdout,_,_ := checkCommand(cmd)
		lvsCheck := strings.Split(strings.Trim(stdout, " "), ",")
		if len(lvsCheck) == 2 {
			return fmt.Errorf("Device %s is in logical volume %s part of volume group %s", device, lvsCheck[0], lvsCheck[1])
		}
	}
	return nil
}

func EnsurePhysicalDevices(devices []string) error {
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
		stdout, _, err := checkCommand(cmd)
		if err != nil {
			return err
		}
		log.Info(stdout)
	}
	return nil
}

func CreateVolumeGroup(volumeGroup string, devices []string) error {
	args := append([]string{"vgcreate", volumeGroup}, devices...)
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	_, _, err := checkCommand(cmd)
	if err != nil {
		return err
	}
	return nil
}

func CreateMetadataVolume(volumeGroup string) (string, error) {
	units := "s" // volume size will be measured in sectors
	totalSize, err := getVolumeGroupSize(volumeGroup, units)
	if err != nil {
		return "", err
	}
	metadataSize := (totalSize + 999) / 1000
	metadataName := volumeGroup + "-meta"

	args := []string{"lvcreate",
		"--size", fmt.Sprintf("%d%s", metadataSize, units),
		"--name", metadataName,
		volumeGroup}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, _, err := checkCommand(cmd)
	if err != nil {
		return "", err
	}
	log.Info(stdout)
	return metadataName, err
}

func CreateDataVolume(volumeGroup string) (string, error) {
	units := "b" // volume size will be measured in bytes
	totalSize, err := getVolumeGroupSize(volumeGroup, units)
	dataSize := (totalSize*90/100 + 511) &^ 511
	dataName := volumeGroup + "-pool"

	args := []string{"lvcreate",
		"--size", fmt.Sprintf("%d%s", dataSize, units),
		"--name", dataName,
		volumeGroup}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, _, err := checkCommand(cmd)
	if err != nil {
		return "", err
	}
	log.Info(stdout)
	return dataName, err
}

func ConvertToThinPool(volumeGroup, dataVolume string, metadataVolume string) error {
	args := []string{"lvconvert",
		"-y",
		"--zero", "n",
		"--thinpool", fmt.Sprintf("%s/%s", volumeGroup, dataVolume),
		"--poolmetadata", fmt.Sprintf("%s/%s", volumeGroup, metadataVolume),
	}
	log.Info(strings.Join(args, " "))
	cmd := exec.Command(args[0], args[1:]...)
	stdout, _, err := checkCommand(cmd)
	if err != nil {
		return err
	}
	log.Info(stdout)
	return nil
}

func getVolumeGroupSize(volumeGroup string, units string) (uint64, error) {
	args := []string{"vgs",
		"--noheadings",
		"--nosuffix",
		"--units", units,
		"--options", "vg_free",
		volumeGroup}
	cmd := exec.Command(args[0], args[1:]...)
	stdout, _, err := checkCommand(cmd)
	if err != nil {
		return 0, err
	}

	sizeString := strings.TrimSpace(stdout)
	if sizeString == "" {
		return 0, fmt.Errorf("invalid volume group")
	}
	size, err := strconv.ParseUint(sizeString, 10, 64)
	if err != nil {
		return 0, err
	}
	return size, nil
}

func GetInfoForLogicalVolume(logicalVolume string) (LogicalVolumeInfo, error) {
	lvi := LogicalVolumeInfo{}
	args := []string{"lvs",
		"--noheadings",
		"--nameprefixes",
		"--options", "lv_name,lv_kernel_major,lv_kernel_minor",
	}
	cmd := exec.Command(args[0], args[1:]...)
	stdout, _, err := checkCommand(cmd)
	if err != nil {
		return lvi, err
	}

	parseError := fmt.Errorf("Failed to parse command output:\n'%s'\n%s",
		strings.Join(args, " "), stdout)

	// Example command output:
	// LVM2_LV_NAME='docker-pool' LVM2_LV_KERNEL_MAJOR='252' LVM2_LV_KERNEL_MINOR='4'
	regexName := regexp.MustCompile("LVM2_LV_NAME='(.+?)'")
	regexMajor := regexp.MustCompile("LVM2_LV_KERNEL_MAJOR='(.+?)'")
	regexMinor := regexp.MustCompile("LVM2_LV_KERNEL_MINOR='(.+?)'")
	for _, line := range strings.Split(stdout, "\n") {
		match := regexName.FindStringSubmatch(line)
		if len(match) != 2 || match[1] != logicalVolume {
			continue
		}

		match = regexMajor.FindStringSubmatch(line)
		if len(match) != 2 {
			return lvi, parseError
		}
		major, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return lvi, parseError
		}

		match = regexMinor.FindStringSubmatch(line)
		if len(match) != 2 {
			return lvi, parseError
		}
		minor, err := strconv.ParseUint(match[1], 10, 32)
		if err != nil {
			return lvi, parseError
		}

		lvi.Name = logicalVolume
		lvi.KernelMajor = uint(major)
		lvi.KernelMinor = uint(minor)
		return lvi, nil
	}

	return lvi, fmt.Errorf("Failed to find logical volume: '%s'", logicalVolume)
}

func (info *LogicalVolumeInfo) GetThinpoolName() (string, error) {
	filename := fmt.Sprintf("/sys/dev/block/%d:%d/dm/name",
		info.KernelMajor, info.KernelMinor)
	contents, err := ioutil.ReadFile(filename)
	if err != nil {
		return "", fmt.Errorf("Error reading %s: %s", filename, err)
	}
	// Get the base name
	basename := strings.TrimSpace(string(contents))
	// Now figure out if it's the pool itself or a subdevice called -tpool
	out, err := exec.Command("dmsetup", "ls", "--target", "thin-pool").CombinedOutput()
	if err != nil {
		return "", err
	}
	scanner := bufio.NewScanner(bytes.NewReader(out))
	var devname string
	for scanner.Scan() {
		s := scanner.Text()
		if strings.HasPrefix(s, basename) {
			devname = strings.Fields(s)[0]
			break
		}
	}
	if devname == "" {
		return "", fmt.Errorf("Unable to determine thin pool name")
	}
	return "/dev/mapper/" + devname, nil
}
