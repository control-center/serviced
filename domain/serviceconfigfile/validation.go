// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by the Apache 2.0
// license that can be found in the LICENSE file.

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
