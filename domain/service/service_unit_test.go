// Copyright 2016 The Serviced Authors.
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

// +build unit

package service_test

import (
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/servicedefinition"
	"github.com/control-center/serviced/utils"
	. "gopkg.in/check.v1"
)

// Test that simple fields are correctly copied from ServiceDefinition
func (s *ServiceDomainUnitTestSuite) TestBuildServiceSimple(t *C) {
	name := "Name"
	title := "title"
	version := "version"
	command := "command"
	description := "description"
	environment := []string{"enviroment:foobar", "a:b", "c:d"}
	tags := []string{"tags", "foo", "bar"}
	imageID := "imageid"
	changeOptions := []string{"changeoptions", "boo"}
	launch := "launch"
	hostname := "hostname"
	privileged := true
	context := map[string]interface{}{"context": 1, "foo": "bar"}
	ramCommitment := utils.NewEngNotation(1024)
	cpuCommitment := uint64(112233)
	disableShell := true
	runs := map[string]string{"runs": "foo"}
	actions := map[string]string{"action": "reaction", "foo": "barr"}
	memoryLimit := 123.456
	cpuShares := int64(123456)
	pidFile := "pidfile"
	startLevel := uint(1234)
	shutdownLevel := uint(4567)

	sd := servicedefinition.ServiceDefinition{
		Name:        name,
		Title:       title,
		Version:     version,
		Command:     command,
		Description: description,
		Environment: environment,
		Tags:        tags,
		ImageID:     imageID,
		// Instances: instances,
		ChangeOptions: changeOptions,
		Launch:        launch,
		Hostname:      hostname,
		Privileged:    privileged,
		// ConfigFiles: configfiles,
		Context: context,
		// Endpoints: endpoints,
		// Services: services,
		// Volumes: volumes,
		// LogConfigs: logconfigs,
		// SnapShot: snapshot,
		RAMCommitment: ramCommitment,
		CPUCommitment: cpuCommitment,
		DisableShell:  disableShell,
		Runs:          runs,
		// Commands: commands,
		Actions: actions,
		// HealthChecks: healthchecks,
		// Prereqs: prereqs,
		// MonitoringProfile: monitoringprofile,
		MemoryLimit:   memoryLimit,
		CPUShares:     cpuShares,
		PIDFile:       pidFile,
		StartLevel:    startLevel,
		EmergencyShutdownLevel: shutdownLevel,
	}
	actual, err := service.BuildService(sd, "", "", 0, "")

	t.Assert(err, IsNil)
	t.Check(actual.Name, Equals, name)
	t.Check(actual.Title, Equals, title)
	t.Check(actual.Startup, Equals, command)
	t.Check(actual.Description, Equals, description)
	t.Check(actual.Environment, DeepEquals, environment)
	t.Check(actual.Tags, DeepEquals, tags)
	t.Check(actual.ImageID, Equals, imageID)
	t.Check(actual.ChangeOptions, DeepEquals, changeOptions)
	t.Check(actual.Launch, Equals, launch)
	t.Check(actual.Hostname, Equals, hostname)
	t.Check(actual.Privileged, Equals, privileged)
	t.Check(actual.Context, DeepEquals, context)
	t.Check(actual.RAMCommitment, Equals, ramCommitment)
	t.Check(actual.CPUCommitment, Equals, cpuCommitment)
	t.Check(actual.DisableShell, Equals, disableShell)
	t.Check(actual.Runs, DeepEquals, runs)
	t.Check(actual.Actions, DeepEquals, actions)
	t.Check(actual.MemoryLimit, Equals, memoryLimit)
	t.Check(actual.CPUShares, Equals, cpuShares)
	t.Check(actual.PIDFile, Equals, pidFile)
	t.Check(actual.StartLevel, Equals, startLevel)
	t.Check(actual.EmergencyShutdownLevel, Equals, shutdownLevel)
}
