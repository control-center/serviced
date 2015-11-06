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

package master

import (
	"github.com/control-center/serviced/domain/applicationendpoint"
)

// Get the endpoints for one or more services
func (s *Server) GetServiceEndpoints(request *EndpointRequest, reply *[]applicationendpoint.EndpointReport) error {
	endpoints, err := s.f.GetServiceEndpoints(s.context(), request.ServiceIDs[0], request.ReportImports, request.ReportExports, request.Validate)
	if err != nil {
		return err
	}

	*reply = endpoints
	return nil
}
