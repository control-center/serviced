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

package facade

import (
	"github.com/control-center/serviced/domain/host"
	"github.com/control-center/serviced/domain/pool"
	"github.com/control-center/serviced/domain/service"
	"github.com/control-center/serviced/domain/serviceimage"
	"github.com/control-center/serviced/domain/servicetemplate"
	"github.com/control-center/serviced/commons/docker"
)

// assert interface
var _ FacadeInterface = &Facade{}

// New creates an initialized Facade instance
func New(dockerRegistryName string) *Facade {
	return &Facade{
		hostStore:      host.NewStore(),
		imageStore:   	serviceimage.NewStore(),
		poolStore:      pool.NewStore(),
		serviceStore:   service.NewStore(),
		templateStore:  servicetemplate.NewStore(),
		registryName:   dockerRegistryName,
		registry:       &docker.DockerRegistry{},
	}
}

// Facade is an entrypoint to available controlplane methods
type Facade struct {
	hostStore      *host.HostStore
	imageStore     *serviceimage.ServiceImageStore
	poolStore      *pool.Store
	templateStore  *servicetemplate.Store
	serviceStore   *service.Store
	registryName   string
	registry       docker.DockerRegistryInterface
}
