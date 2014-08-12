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

package serviceconfigfile

import (
	"github.com/control-center/serviced/validation"
	"strings"
)

//ValidEntity check if fields are valid
func (scf SvcConfigFile) ValidEntity() error {
	vErr := validation.NewValidationError()
	vErr.Add(validation.NotEmpty("ID", scf.ID))
	vErr.Add(validation.NotEmpty("ServiceTenantID", scf.ServiceTenantID))
	vErr.Add(validation.NotEmpty("ServicePath", scf.ServicePath))

	//path must start with /
	if !strings.HasPrefix(scf.ServicePath, "/") {
		vErr.AddViolation("field ServicePath must start with /")
	}

	vErr.Add(validation.NotEmpty("Content", scf.ConfFile.Content))
	vErr.Add(validation.NotEmpty("FileName", scf.ConfFile.Filename))

	if vErr.HasError() {
		return vErr
	}
	return nil
}
