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
	"regexp"
	"strconv"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/utils"
	"github.com/docker/go-units"
)

type ThinPoolCreate struct {
	Args struct {
		Purpose string   `description:"Purpose of the thin pool (docker|serviced)"`
		Devices []string `description:"Block devices to use" required:"1"`
	} `positional-args:"yes" required:"yes"`
	Size string `short:"s" long:"size" default:"90%" description:"Size of the thin pool to be created. May be specified either as a fixed amount (e.g., \"20G\") or a percentage of the free space in the volume group (e.g., \"90%\")"`
}

type LogicalVolumeInfo struct {
	Name        string
	VGName      string
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
	size := c.Size
	logger := log.WithFields(log.Fields{
		"purpose": purpose,
		"devices": devices,
	})

	// Validate the purpose
	if purpose != "serviced" && purpose != "docker" {
		logger.Fatal("Purpose must be one of (docker, serviced)")
	}

	// Validate the devices
	if len(devices) == 0 {
		logger.Fatal("Must specify devices or a volume group in which to create a pool")
	}

	// Validate the size
	if _, err := strconv.Atoi(strings.TrimSuffix(size, "%")); err != nil {
		if _, err := units.RAMInBytes(size); err != nil {
			logger.WithField("size", size).Fatal("Invalid size. Please specify size either in bytes (e.g. \"60GB\") or as a percentage of free space in the volume group (e.g. \"90%\")")
		}
	}

	var vg string
	if len(devices) > 1 || !IsVolumeGroup(devices[0]) {
		if err := createVolumeGroup(purpose, devices); err != nil {
			logger.WithError(err).Fatal("Unable to create volume group")
		}
		vg = purpose
		logger = log.WithField("volumegroup", purpose)
		logger.Info("Created volume group")
	} else {
		vg = devices[0]
		logger = log.WithField("volumegroup", vg)
		logger.Info("Using specified volume group")
	}

	thinPoolName, err := createThinPool(purpose, vg, c.Size)
	if err != nil {
		logger.WithError(err).Fatal("Unable to create thin pool")
	}

	logger.WithField("thinpool", thinPoolName).Info("Created thin pool")

	// Output to stdout for scriptability
	fmt.Println(thinPoolName)
	return nil
}

func createVolumeGroup(name string, devices []string) error {
	// Make sure we don't create a volume group on top of LVs
	if err := EnsureNotLogicalVolumes(devices); err != nil {
		return err
	}

	// Make sure we can create LVM2 PVs from the devices passed in
	if err := EnsurePhysicalDevices(devices); err != nil {
		return err
	}

	// Create the volume group
	if err := CreateVolumeGroup(name, devices); err != nil {
		return err
	}
	return nil
}

func getMetadataSize(purpose string, dataSize int64) int64 {
	// The default metadata calculation used by lvcreate is, very roughly,
	//
	//     metadataSize = floor(dataSize/4) / 1024
	//
	// We want to pad that number by a considerable margin, depending on
	// whether the pool being created is for docker or serviced.
	// We'll use the default for pools less than 25GB, then ~1% for serviced
	// pools, ~2% for Docker pools.
	if dataSize <= 25*units.GiB {
		return -1
	}

	var ds int64
	if purpose == "docker" {
		ds = dataSize / 50
	} else {
		ds = dataSize / 100
	}
	return (ds + 511) &^ 511
}

func createThinPool(purpose, volumeGroup, size string) (string, error) {
	logger := log.WithFields(log.Fields{
		"purpose":     purpose,
		"volumegroup": volumeGroup,
		"size":        size,
	})

	args := []string{"lvcreate", "-T"}

	vgSize, err := getVolumeGroupSize(volumeGroup, "b")
	if err != nil {
		return "", err
	}

	var dataSize int64

	if strings.HasSuffix(size, "%") {
		// Size is a percentage of free space
		percentage, err := strconv.Atoi(strings.TrimSuffix(size, "%"))
		if err != nil {
			return "", err
		}
		dataSize = int64(vgSize) * int64(percentage) / 100
		args = append(args, "-l", fmt.Sprintf("%sFREE", size))
	} else {
		// Size is an absolute amount
		dataSize, _ = units.RAMInBytes(size)
		args = append(args, "-L", size)

	}

	mdbytes := getMetadataSize(purpose, dataSize)
	if mdbytes > -1 {
		args = append(args, "--poolmetadatasize", fmt.Sprintf("%dB", mdbytes))
	}

	poolName := fmt.Sprintf("%s-pool", purpose)
	fullName := fmt.Sprintf("%s/%s", volumeGroup, poolName)
	args = append(args, fullName)

	logger.WithFields(log.Fields{
		"command": strings.Join(args, " "),
		"pool":    fullName,
	}).Debug("Creating thin pool")
	cmd := exec.Command(args[0], args[1:]...)
	if _, _, err := checkCommand(cmd); err != nil {
		return "", err
	}

	lvInfo, err := GetInfoForLogicalVolume(poolName)
	if err != nil {
		return "", err
	}

	thinPoolName, err := lvInfo.GetThinpoolName()
	if err != nil {
		return "", nil
	}

	return thinPoolName, nil
}

func IsVolumeGroup(device string) bool {
	cmd := exec.Command("vgs", device)
	_, _, rc, _ := runCommand(cmd)
	return rc == 0
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
		cmd := exec.Command(args[0], args[1:]...)
		stdout, _, _ := checkCommand(cmd)
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
		cmd = exec.Command(args[0], args[1:]...)
		_, _, err = checkCommand(cmd)
		if err != nil {
			return err
		}
	}
	return nil
}

func CreateVolumeGroup(volumeGroup string, devices []string) error {
	args := append([]string{"vgcreate", volumeGroup}, devices...)
	cmd := exec.Command(args[0], args[1:]...)
	_, _, err := checkCommand(cmd)
	if err != nil {
		return err
	}
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
		"--options", "lv_name,vg_name,lv_kernel_major,lv_kernel_minor",
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
	regexVGName := regexp.MustCompile("LVM2_VG_NAME='(.+?)'")
	regexMajor := regexp.MustCompile("LVM2_LV_KERNEL_MAJOR='(.+?)'")
	regexMinor := regexp.MustCompile("LVM2_LV_KERNEL_MINOR='(.+?)'")
	for _, line := range strings.Split(stdout, "\n") {
		match := regexName.FindStringSubmatch(line)
		if len(match) != 2 || match[1] != logicalVolume {
			continue
		}

		match = regexVGName.FindStringSubmatch(line)
		if len(match) != 2 {
			return lvi, parseError
		}
		vgName := match[1]

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
		lvi.VGName = vgName
		lvi.KernelMajor = uint(major)
		lvi.KernelMinor = uint(minor)

		log.WithFields(log.Fields{
			"lvname": lvi.Name,
			"major":  lvi.KernelMajor,
			"minor":  lvi.KernelMinor,
		}).Debug("Logical Volume Info")
		return lvi, nil
	}

	return lvi, fmt.Errorf("Failed to find logical volume: '%s'", logicalVolume)
}

func (info *LogicalVolumeInfo) GetThinpoolName() (string, error) {
	out, err := exec.Command("dmsetup", "info", "-c", "-j", fmt.Sprintf("%d", info.KernelMajor), "-m", fmt.Sprintf("%d", info.KernelMinor), "-o", "name", "--noheadings").CombinedOutput()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("/dev/mapper/%s", bytes.TrimSpace(out)), nil
}
