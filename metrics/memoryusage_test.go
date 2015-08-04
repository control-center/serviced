// Copyright 2014 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

// +build unit

package metrics

import (
	"encoding/json"
	"reflect"
	"testing"
	"time"
)

func TestConvertMemoryUsage(t *testing.T) {
	testData := []byte(`
	{"clientId":"not-specified","endTime":"now","endTimeActual":1427487495,"returnset":"exact","series":true,"source":"OpenTSDB","startTime":"1m-ago","startTimeActual":1427487435,"results":[{"datapoints":[{"timestamp":1427487441,"value":2.28245504E8},{"timestamp":1427487451,"value":2.28245504E8},{"timestamp":1427487461,"value":2.28245504E8},{"timestamp":1427487471,"value":2.28245504E8},{"timestamp":1427487481,"value":2.28245504E8},{"timestamp":1427487491,"value":2.28282368E8}],"metric":"series1","tags":{"controlplane_instance_id":["0"],"controlplane_service_id":["d2b1sxt1jh7ryzuv4kszg14zv"]},"queryStatus":{"message":"","status":"SUCCESS"}},{"datapoints":[{"timestamp":1427487441,"value":9.79058688E8},{"timestamp":1427487451,"value":9.79058688E8},{"timestamp":1427487461,"value":9.78763776E8},{"timestamp":1427487471,"value":9.7875968E8},{"timestamp":1427487481,"value":9.7875968E8},{"timestamp":1427487491,"value":9.7875968E8}],"metric":"series2","tags":{"controlplane_instance_id":["0"],"controlplane_service_id":["8frik9ddntu38e3urbeimiqnf"]},"queryStatus":{"message":"","status":"SUCCESS"}}]}
	`)

	var perfdata PerformanceData
	if err := json.Unmarshal(testData, &perfdata); err != nil {
		t.Fatalf("Could not unmarshal testData: %s", err)
	}

	actual := convertMemoryUsage(&perfdata)
	expected := []MemoryUsageStats{
		{
			StartDate:  time.Unix(1427487435, 0),
			EndDate:    time.Unix(1427487495, 0),
			ServiceID:  "d2b1sxt1jh7ryzuv4kszg14zv",
			InstanceID: "0",
			Last:       int64(228282368),
			Max:        int64(228282368),
			Average:    int64(228251648),
		}, {
			StartDate:  time.Unix(1427487435, 0),
			EndDate:    time.Unix(1427487495, 0),
			ServiceID:  "8frik9ddntu38e3urbeimiqnf",
			InstanceID: "0",
			Last:       int64(978759680),
			Max:        int64(979058688),
			Average:    int64(978860032),
		},
	}

	if !reflect.DeepEqual(actual, expected) {
		t.Errorf("Expected does not equal actual: %+v", actual)
	}
}
