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

package master

import (
	"github.com/control-center/serviced/domain/applicationendpoint"
)

// Defines a request to get a list of endpoints for one or more services
type EndpointRequest struct {
	ServiceIDs []string
	ReportImports bool
	ReportExports bool
	Validate   bool
}

// GetServiceEndpoints gets the endpoints for one or more services
func (c *Client) GetServiceEndpoints(serviceIDs []string, reportImports, reportExports bool, validate bool) ([]applicationendpoint.EndpointReport, error) {
	request := &EndpointRequest{
		ServiceIDs:    serviceIDs,
		ReportImports: reportImports,
		ReportExports: reportExports,
		Validate:      validate,
	}
	result := make([]applicationendpoint.EndpointReport, 0)
	err := c.call("GetServiceEndpoints", request, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
