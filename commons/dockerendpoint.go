// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package commons

import "os"

const (
	defaultDockerEndpoint = "unix:///var/run/docker.sock"
	dep                   = "SERVICED_DOCKER_ENDPOINT"
)

// DockerEndpoint returns a string designating the socket a Docker client should bind to, it
// uses the value it finds in the SERVICED_DOCKER_ENDPOINT environment variable, or, if that
// variable is unset, the default Docker endpoint.
func DockerEndpoint() string {
	ep := os.Getenv(dep)
	if len(ep) == 0 {
		return defaultDockerEndpoint
	}
	return ep
}
