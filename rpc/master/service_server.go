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

import ()

// Use a new image for a given service - this will pull the image and tag it
func (s *Server) ServiceUse(request *ServiceUseRequest, response *string) error {
	if err := s.f.ServiceUse(s.context(), request.ServiceID, request.ImageID, request.Registry, request.ReplaceImgs, request.NoOp); err != nil {
		return err
	}
	*response = ""
	return nil
}

// Wait on specified services to be in the given state
func (s *Server) WaitService(request *WaitServiceRequest, throwaway *string) error {
	err := s.f.WaitService(s.context(), request.State, request.Timeout, request.Recursive, request.ServiceIDs...)
	return err
}
