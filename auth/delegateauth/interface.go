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

// Package delegateauth implements JWT authentication for serviced delegates.
package delegateauth

import "net/http"

// DelegateAuthorizer is the interface for using JWT authentication between
// a downstream serviced delegate and an upstream serviced instance.
// Downstream serviced instances can use AddToken() to add a JWT token to
// an HTTP request.
// Upstream serviced instances can use ValidateToken() to authenticate
// requests from downstream instances.
type DelegateAuthorizer interface {
	// Build a JWT and add it to the request header
	AddToken(poolID string, request *http.Request, uriPrefix string) error

	// Validate the JWT token in the request header
	ValidateToken(request *http.Request) error
}
