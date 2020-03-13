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

package hostkey

import (
	"encoding/pem"
	"fmt"

	"github.com/control-center/serviced/validation"
)

const excludeChars = " \t\r\n\v\f/?"

func isRSAPublicKey(value string) error {
	block, remaining := pem.Decode([]byte(value))
	if block == nil {
		return validation.NewViolation("Invalid public key PEM block")
	}
	if block.Type != "PUBLIC KEY" {
		msg := fmt.Sprintf("Unexpected public key PEM type: %s", block.Type)
		return validation.NewViolation(msg)
	}
	if len(remaining) > 0 {
		return validation.NewViolation("Unexpected characters following public key PEM block")
	}
	return nil
}

// ValidEntity validates the RSAKey.
func (rsaKey *RSAKey) ValidEntity() error {
	violations := validation.NewValidationError()
	violations.Add(isRSAPublicKey(rsaKey.PEM))
	if violations.HasError() {
		return violations
	}
	return nil
}
