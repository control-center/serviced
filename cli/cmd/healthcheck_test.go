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

// +build unit

package cmd

import (
	"fmt"

	"github.com/control-center/serviced/cli/api"
	"github.com/control-center/serviced/dao"
	"github.com/control-center/serviced/domain"
	"github.com/control-center/serviced/utils"
)

var DefaultHealthCheckAPITest = HealthCheckAPITest{apiResults: DefaultTestHealthCheckResults}

var DefaultHealthStatus = domain.HealthCheckStatus{
	Name:      "running",
	Status:    "passed",
	Timestamp: 0,
	Interval:  0,
	StartedAt: 0,
	Failure:   "",
}

var FailedHealthStatus = domain.HealthCheckStatus{
	Name:      "running",
	Status:    "failed",
	Timestamp: 0,
	Interval:  0,
	StartedAt: 0,
	Failure:   "something went wrong",
}

var StoppedHealthStatus = domain.HealthCheckStatus{
	Name:      "running",
	Status:    "stopped",
	Timestamp: 0,
	Interval:  0,
	StartedAt: 0,
	Failure:   "",
}

var UnknownHealthStatus = domain.HealthCheckStatus{
	Name:      "running",
	Status:    "unknown",
	Timestamp: 0,
	Interval:  0,
	StartedAt: 0,
	Failure:   "",
}

var DefaultTestHealthCheckResults = []dao.IServiceHealthResult{
	dao.IServiceHealthResult{
		ServiceName:    "test-iservice-1",
		ContainerName:  "container-1",
		ContainerID:    "id1",
		HealthStatuses: []domain.HealthCheckStatus{DefaultHealthStatus},
	},
	dao.IServiceHealthResult{
		ServiceName:    "test-iservice-2",
		ContainerName:  "container-2",
		ContainerID:    "id2",
		HealthStatuses: []domain.HealthCheckStatus{DefaultHealthStatus},
	},
	dao.IServiceHealthResult{
		ServiceName:    "test-iservice-failed",
		ContainerName:  "container-failed",
		ContainerID:    "id-failed",
		HealthStatuses: []domain.HealthCheckStatus{FailedHealthStatus},
	},
	dao.IServiceHealthResult{
		ServiceName:    "test-iservice-stopped",
		ContainerName:  "container-stopped",
		ContainerID:    "id-stopped",
		HealthStatuses: []domain.HealthCheckStatus{StoppedHealthStatus},
	},
	dao.IServiceHealthResult{
		ServiceName:    "test-iservice-unknown",
		ContainerName:  "container-unknown",
		ContainerID:    "id-unknown",
		HealthStatuses: []domain.HealthCheckStatus{UnknownHealthStatus},
	},
}

type HealthCheckAPITest struct {
	api.API
	apiResults []dao.IServiceHealthResult
}

func InitHealthCheckAPITest(args ...string) {
	c := New(DefaultHealthCheckAPITest, utils.TestConfigReader(make(map[string]string)))
	c.exitDisabled = true
	c.Run(args)
}

func (t HealthCheckAPITest) ServicedHealthCheck(IServiceNames []string) ([]dao.IServiceHealthResult, error) {
	mockResults := make([]dao.IServiceHealthResult, 0)
	for _, serviceName := range IServiceNames {
		found := false
		for _, result := range t.apiResults {
			if serviceName == result.ServiceName {
				found = true
				mockResults = append(mockResults, result)
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("could not find isvc %q", serviceName)
		}
	}
	return mockResults, nil
}

func ExampleServicedCLI_CmdHealthCheck_oneService() {
	InitHealthCheckAPITest("serviced", "healthcheck", "test-iservice-1")

	// Output:
	// Service Name     Container Name  Container ID  Health Check  Status
	// test-iservice-1  container-1     id1           running       passed
}

func ExampleServicedCLI_CmdHealthCheck_twoServices() {
	InitHealthCheckAPITest("serviced", "healthcheck", "test-iservice-1", "test-iservice-2")

	// Output:
	// Service Name     Container Name  Container ID  Health Check  Status
	// test-iservice-1  container-1     id1           running       passed
	// test-iservice-2  container-2     id2           running       passed
}

func ExampleServicedCLI_CmdHealthCheck_undefinedService() {
	pipeStderr(InitHealthCheckAPITest, "serviced", "healthcheck", "undefined-iservice")

	// Output:
	// could not find isvc "undefined-iservice"
	// exit code 2
}

func ExampleServicedCLI_CmdHealthCheck_failedStatus() {
	pipeStderr(InitHealthCheckAPITest, "serviced", "healthcheck", "test-iservice-failed")

	// Output:
	// Service Name          Container Name    Container ID  Health Check  Status
	// test-iservice-failed  container-failed  id-failed     running       failed - something went wrong
	// exit code 1
}

func ExampleServicedCLI_CmdHealthCheck_stoppedStatus() {
	pipeStderr(InitHealthCheckAPITest, "serviced", "healthcheck", "test-iservice-stopped")

	// Output:
	// Service Name           Container Name     Container ID  Health Check  Status
	// test-iservice-stopped  container-stopped  id-stopped    running       stopped
	// exit code 1
}

func ExampleServicedCLI_CmdHealthCheck_unknownStatus() {
	pipeStderr(InitHealthCheckAPITest, "serviced", "healthcheck", "test-iservice-unknown")

	// Output:
	// Service Name           Container Name     Container ID  Health Check  Status
	// test-iservice-unknown  container-unknown  id-unknown    running       unknown
	// exit code 1
}
