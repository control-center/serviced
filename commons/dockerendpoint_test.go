// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package commons

import (
	"os"
	"testing"
)

func TestDefaultDockerEndpoint(t *testing.T) {
	dockersock := "unix:///var/run/docker.sock"

	if err := os.Setenv("SERVICED_DOCKER_ENDPOINT", ""); err != nil {
		t.Fatalf("unable to clear SERVICED_DOCKER_ENDPOINT: %v", err)
	}

	if dep := DockerEndpoint(); dep != dockersock {
		t.Fatalf("expected %s got %s", dockersock, dep)
	}
}

func TestDockerEndpoint(t *testing.T) {
	tcpep := "http://127.0.0.1:3006"

	if err := os.Setenv("SERVICED_DOCKER_ENDPOINT", tcpep); err != nil {
		t.Fatalf("unable to set SERVICED_DOCKER_ENDPOINT: %v", err)
	}

	if snr := DockerEndpoint(); snr != tcpep {
		t.Fatalf("expected %s got %s", tcpep, snr)
	}
}
